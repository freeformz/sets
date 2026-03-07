package sets

import (
	"cmp"
	"encoding/csv"
	"fmt"
	"math"
	"os"
	"runtime"
	"slices"
	"strconv"
	"testing"
	"time"
)

// TODO: Add parallel benchmarks for concurrent-safe types
// (SyncMap, Locked, LockedOrdered) to measure contention under concurrent access.

const defaultSamples = 30

// measure returns per-call timing in nanoseconds for fn.
// It auto-calibrates repetitions so each sample measures at least 1ms of work.
func benchMeasure(samples int, fn func()) []float64 {
	// Calibrate reps
	reps := 1
	for {
		start := time.Now()
		for range reps {
			fn()
		}
		d := time.Since(start)
		if d >= time.Millisecond {
			break
		}
		if reps > 1_000_000 {
			break
		}
		if d <= 0 {
			reps *= 100
		} else {
			reps = int(float64(reps) * float64(2*time.Millisecond) / float64(d))
			reps = max(reps, 1)
		}
	}

	results := make([]float64, samples)
	for i := range samples {
		runtime.GC()
		start := time.Now()
		for range reps {
			fn()
		}
		results[i] = float64(time.Since(start).Nanoseconds()) / float64(reps)
	}
	return results
}

// benchMeasureWithSetup measures fn after calling setup each sample.
// No repetition calibration — used when fn has side effects that prevent repetition (e.g., Remove).
// It adapts the sample count based on how long a single call takes, and respects a total time budget.
func benchMeasureWithSetup(maxSamples int, setup func(), fn func()) []float64 {
	const maxTotalTime = 60 * time.Second // total time budget for this measurement

	// Warmup run to estimate per-sample time
	setup()
	runtime.GC()
	start := time.Now()
	fn()
	warmup := time.Since(start)

	// Adaptive sample count based on per-sample cost
	samples := maxSamples
	if warmup > 0 {
		budgetSamples := int(maxTotalTime / warmup)
		samples = min(samples, max(budgetSamples, 1))
	}

	results := make([]float64, 0, samples)
	results = append(results, float64(warmup.Nanoseconds())) // include warmup

	for range samples - 1 {
		setup()
		runtime.GC()
		start := time.Now()
		fn()
		results = append(results, float64(time.Since(start).Nanoseconds()))
	}
	return results
}

type benchStat struct {
	min, max, avg, stddev, p50, p95, p99 float64
}

func computeStats(samples []float64) benchStat {
	slices.Sort(samples)
	n := float64(len(samples))

	var sum float64
	for _, s := range samples {
		sum += s
	}
	avg := sum / n

	var variance float64
	for _, s := range samples {
		d := s - avg
		variance += d * d
	}
	variance /= n

	return benchStat{
		min:    samples[0],
		max:    samples[len(samples)-1],
		avg:    avg,
		stddev: math.Sqrt(variance),
		p50:    benchPercentile(samples, 0.50),
		p95:    benchPercentile(samples, 0.95),
		p99:    benchPercentile(samples, 0.99),
	}
}

func benchPercentile(sorted []float64, p float64) float64 {
	idx := p * float64(len(sorted)-1)
	lower := int(math.Floor(idx))
	upper := int(math.Ceil(idx))
	if lower == upper || upper >= len(sorted) {
		return sorted[lower]
	}
	frac := idx - float64(lower)
	return sorted[lower]*(1-frac) + sorted[upper]*frac
}

type benchRecord struct {
	op, impl, typ string
	size          int
	unit          string
	stats         benchStat
}

func writeStatsCSV(path string, records []benchRecord) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}

	w := csv.NewWriter(f)

	w.Write([]string{"operation", "impl", "type", "size", "unit", "min", "max", "avg", "stddev", "p50", "p95", "p99"})
	for _, r := range records {
		w.Write([]string{
			r.op, r.impl, r.typ, strconv.Itoa(r.size), r.unit,
			fmt.Sprintf("%.2f", r.stats.min),
			fmt.Sprintf("%.2f", r.stats.max),
			fmt.Sprintf("%.2f", r.stats.avg),
			fmt.Sprintf("%.2f", r.stats.stddev),
			fmt.Sprintf("%.2f", r.stats.p50),
			fmt.Sprintf("%.2f", r.stats.p95),
			fmt.Sprintf("%.2f", r.stats.p99),
		})
	}

	w.Flush()
	if err := w.Error(); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

// scalePerElem divides each sample by n (for per-element timing).
func scalePerElem(raw []float64, n int) []float64 {
	fn := float64(n)
	for i := range raw {
		raw[i] /= fn
	}
	return raw
}

func collectAllStats[M cmp.Ordered](
	t *testing.T,
	records *[]benchRecord,
	implName string,
	typeName string,
	newSet func() Set[M],
	genElems func(int) []M,
	sizes []int,
	samples int,
) {
	for _, size := range sizes {
		elems := genElems(size)

		t.Logf("  %s/%s size=%d", implName, typeName, size)

		// --- Per-element operations (ns/elem) ---

		// Add: create set + add N elements
		raw := benchMeasure(samples, func() {
			s := newSet()
			for _, e := range elems {
				s.Add(e)
			}
		})
		*records = append(*records, benchRecord{"Add", implName, typeName, size, "ns/elem", computeStats(scalePerElem(raw, size))})

		// Contains: lookup every element in pre-populated set
		s := newSet()
		for _, e := range elems {
			s.Add(e)
		}
		raw = benchMeasure(samples, func() {
			for _, e := range elems {
				s.Contains(e)
			}
		})
		*records = append(*records, benchRecord{"Contains", implName, typeName, size, "ns/elem", computeStats(scalePerElem(raw, size))})

		// Remove: needs per-sample setup (repopulate after each removal pass)
		raw = benchMeasureWithSetup(samples,
			func() {
				for _, e := range elems {
					s.Add(e)
				}
			},
			func() {
				for _, e := range elems {
					s.Remove(e)
				}
			},
		)
		*records = append(*records, benchRecord{"Remove", implName, typeName, size, "ns/elem", computeStats(scalePerElem(raw, size))})

		// Re-populate s for batch ops
		s = newSet()
		for _, e := range elems {
			s.Add(e)
		}

		// --- Batch operations (ns/op) ---

		// Clone
		raw = benchMeasure(samples, func() { s.Clone() })
		*records = append(*records, benchRecord{"Clone", implName, typeName, size, "ns/op", computeStats(raw)})

		// Filter (~50% of elements)
		raw = benchMeasure(samples, func() {
			cnt := 0
			Filter(s, func(M) bool {
				cnt++
				return cnt%2 == 0
			})
		})
		*records = append(*records, benchRecord{"Filter", implName, typeName, size, "ns/op", computeStats(raw)})

		// MapBy (identity)
		raw = benchMeasure(samples, func() {
			MapBy(s, func(m M) M { return m })
		})
		*records = append(*records, benchRecord{"MapBy", implName, typeName, size, "ns/op", computeStats(raw)})

		// Chunk
		chunkSize := max(size/10, 1)
		raw = benchMeasure(samples, func() {
			for range Chunk(s, chunkSize) {
			}
		})
		*records = append(*records, benchRecord{"Chunk", implName, typeName, size, "ns/op", computeStats(raw)})

		// --- Two-set operations (ns/op) ---
		// Both sets have N elements with 50% overlap.
		allElems := genElems(size + size/2)
		bSet := newSet()
		for _, e := range allElems[size/2 : size+size/2] {
			bSet.Add(e)
		}

		raw = benchMeasure(samples, func() { Union(s, bSet) })
		*records = append(*records, benchRecord{"Union", implName, typeName, size, "ns/op", computeStats(raw)})

		raw = benchMeasure(samples, func() { Intersection(s, bSet) })
		*records = append(*records, benchRecord{"Intersection", implName, typeName, size, "ns/op", computeStats(raw)})

		raw = benchMeasure(samples, func() { Difference(s, bSet) })
		*records = append(*records, benchRecord{"Difference", implName, typeName, size, "ns/op", computeStats(raw)})

		raw = benchMeasure(samples, func() { SymmetricDifference(s, bSet) })
		*records = append(*records, benchRecord{"SymmetricDifference", implName, typeName, size, "ns/op", computeStats(raw)})
	}
}

func TestBenchStats(t *testing.T) {
	if os.Getenv("BENCH_STATS") == "" {
		t.Skip("set BENCH_STATS=1 to run statistical benchmarks")
	}

	outFile := os.Getenv("BENCH_STATS_OUT")
	if outFile == "" {
		outFile = "bench_stats.csv"
	}

	samples := defaultSamples

	var records []benchRecord

	for _, impl := range benchImpls {
		t.Logf("Benchmarking %s...", impl.name)
		collectAllStats(t, &records, impl.name, "int", impl.newInt, genInts, benchSizes, samples)
		collectAllStats(t, &records, impl.name, "string", impl.newStr, genStrings, benchSizes, samples)
	}

	if err := writeStatsCSV(outFile, records); err != nil {
		t.Fatal(err)
	}
	t.Logf("Results written to %s (%d records)", outFile, len(records))
}
