package sets

import (
	"cmp"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"testing"
)

// TODO: Add parallel benchmarks (b.RunParallel) for concurrent-safe types
// (SyncMap, Locked, LockedOrdered) to measure contention under concurrent access.

var benchSizes = []int{10, 100, 1_000, 10_000, 100_000, 1_000_000}

func init() {
	if os.Getenv("BENCH_LARGE") != "" {
		benchSizes = append(benchSizes, 10_000_000)
	}
}

// --- Element generators ---

func genInts(n int) []int {
	e := make([]int, n)
	for i := range e {
		e[i] = i
	}
	return e
}

func genStrings(n int) []string {
	e := make([]string, n)
	for i := range e {
		e[i] = strconv.Itoa(i)
	}
	return e
}

// --- Implementation table ---

type benchImpl struct {
	name   string
	newInt func() Set[int]
	newStr func() Set[string]
}

var benchImpls = []benchImpl{
	{"Map", func() Set[int] { return New[int]() }, func() Set[string] { return New[string]() }},
	{"SyncMap", func() Set[int] { return NewSyncMap[int]() }, func() Set[string] { return NewSyncMap[string]() }},
	{"Locked", func() Set[int] { return NewLocked[int]() }, func() Set[string] { return NewLocked[string]() }},
	{"Ordered", func() Set[int] { return NewOrdered[int]() }, func() Set[string] { return NewOrdered[string]() }},
	{"LockedOrdered", func() Set[int] { return NewLockedOrdered[int]() }, func() Set[string] { return NewLockedOrdered[string]() }},
}

// --- Generic runners ---

// benchEach runs fn for every implementation × element type × size combination.
func benchEach(
	b *testing.B,
	intFn func(*testing.B, func() Set[int], []int),
	strFn func(*testing.B, func() Set[string], []string),
) {
	for _, impl := range benchImpls {
		b.Run(impl.name, func(b *testing.B) {
			forEachSize(b, "int", impl.newInt, genInts, intFn)
			forEachSize(b, "string", impl.newStr, genStrings, strFn)
		})
	}
}

func forEachSize[M cmp.Ordered](
	b *testing.B,
	typeName string,
	newSet func() Set[M],
	genElems func(int) []M,
	fn func(*testing.B, func() Set[M], []M),
) {
	for _, size := range benchSizes {
		b.Run(fmt.Sprintf("%s/%d", typeName, size), func(b *testing.B) {
			elems := genElems(size)
			fn(b, newSet, elems)
		})
	}
}

// benchEachTwoSet runs a two-set operation for every implementation × element type × size.
// Both sets have size N elements with 50% overlap.
func benchEachTwoSet(
	b *testing.B,
	intOp func(Set[int], Set[int]) Set[int],
	strOp func(Set[string], Set[string]) Set[string],
) {
	for _, impl := range benchImpls {
		b.Run(impl.name, func(b *testing.B) {
			forEachSizeTwoSet(b, "int", impl.newInt, genInts, intOp)
			forEachSizeTwoSet(b, "string", impl.newStr, genStrings, strOp)
		})
	}
}

func forEachSizeTwoSet[M cmp.Ordered](
	b *testing.B,
	typeName string,
	newSet func() Set[M],
	genElems func(int) []M,
	op func(Set[M], Set[M]) Set[M],
) {
	for _, size := range benchSizes {
		b.Run(fmt.Sprintf("%s/%d", typeName, size), func(b *testing.B) {
			// Generate 1.5N elements so both sets can have N elements with 50% overlap.
			all := genElems(size + size/2)
			a := newSet()
			for _, e := range all[:size] {
				a.Add(e)
			}
			bSet := newSet()
			for _, e := range all[size/2 : size+size/2] {
				bSet.Add(e)
			}
			for b.Loop() {
				op(a, bSet)
			}
		})
	}
}

// --- Benchmark kernels ---

func benchAdd[M cmp.Ordered](b *testing.B, newSet func() Set[M], elems []M) {
	for b.Loop() {
		s := newSet()
		for _, e := range elems {
			s.Add(e)
		}
	}
}

func benchContains[M cmp.Ordered](b *testing.B, newSet func() Set[M], elems []M) {
	s := newSet()
	for _, e := range elems {
		s.Add(e)
	}
	for b.Loop() {
		for _, e := range elems {
			s.Contains(e)
		}
	}
}

func benchRemove[M cmp.Ordered](b *testing.B, newSet func() Set[M], elems []M) {
	for b.Loop() {
		b.StopTimer()
		s := newSet()
		for _, e := range elems {
			s.Add(e)
		}
		b.StartTimer()
		for _, e := range elems {
			s.Remove(e)
		}
	}
}

func benchClone[M cmp.Ordered](b *testing.B, newSet func() Set[M], elems []M) {
	s := newSet()
	for _, e := range elems {
		s.Add(e)
	}
	for b.Loop() {
		s.Clone()
	}
}

func benchFilter[M cmp.Ordered](b *testing.B, newSet func() Set[M], elems []M) {
	s := newSet()
	for _, e := range elems {
		s.Add(e)
	}
	keep := true
	pred := func(M) bool {
		keep = !keep
		return keep
	}
	for b.Loop() {
		keep = true
		Filter(s, pred)
	}
}

func benchMapBy[M cmp.Ordered](b *testing.B, newSet func() Set[M], elems []M) {
	s := newSet()
	for _, e := range elems {
		s.Add(e)
	}
	fn := func(m M) M { return m }
	for b.Loop() {
		MapBy(s, fn)
	}
}

func benchChunk[M cmp.Ordered](b *testing.B, newSet func() Set[M], elems []M) {
	s := newSet()
	for _, e := range elems {
		s.Add(e)
	}
	chunkSize := max(len(elems)/10, 1)
	for b.Loop() {
		for range Chunk(s, chunkSize) {
		}
	}
}

// --- Top-level benchmarks: core operations ---

func BenchmarkAdd(b *testing.B) {
	benchEach(b, benchAdd[int], benchAdd[string])
}

func BenchmarkContains(b *testing.B) {
	benchEach(b, benchContains[int], benchContains[string])
}

func BenchmarkRemove(b *testing.B) {
	benchEach(b, benchRemove[int], benchRemove[string])
}

// --- Top-level benchmarks: operations returning new sets ---

func BenchmarkClone(b *testing.B) {
	benchEach(b, benchClone[int], benchClone[string])
}

func BenchmarkFilter(b *testing.B) {
	benchEach(b, benchFilter[int], benchFilter[string])
}

func BenchmarkMapBy(b *testing.B) {
	benchEach(b, benchMapBy[int], benchMapBy[string])
}

func BenchmarkChunk(b *testing.B) {
	benchEach(b, benchChunk[int], benchChunk[string])
}

func benchElements[M cmp.Ordered](b *testing.B, newSet func() Set[M], elems []M) {
	s := newSet()
	for _, e := range elems {
		s.Add(e)
	}
	for b.Loop() {
		Elements(s)
	}
}

func BenchmarkElements(b *testing.B) {
	benchEach(b, benchElements[int], benchElements[string])
}

func benchMarshalJSON[M cmp.Ordered](b *testing.B, newSet func() Set[M], elems []M) {
	s := newSet()
	for _, e := range elems {
		s.Add(e)
	}
	jm, ok := s.(json.Marshaler)
	if !ok {
		b.Fatalf("%T is not a json.Marshaler", s)
	}
	for b.Loop() {
		if _, err := jm.MarshalJSON(); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMarshalJSON(b *testing.B) {
	benchEach(b, benchMarshalJSON[int], benchMarshalJSON[string])
}

func BenchmarkNewWith(b *testing.B) {
	for _, size := range benchSizes {
		b.Run(fmt.Sprintf("int/%d", size), func(b *testing.B) {
			elems := genInts(size)
			for b.Loop() {
				NewWith(elems...)
			}
		})
	}
}

func BenchmarkUnion(b *testing.B) {
	benchEachTwoSet(b, Union[int], Union[string])
}

func BenchmarkIntersection(b *testing.B) {
	benchEachTwoSet(b, Intersection[int], Intersection[string])
}

func BenchmarkDifference(b *testing.B) {
	benchEachTwoSet(b, Difference[int], Difference[string])
}

func BenchmarkSymmetricDifference(b *testing.B) {
	benchEachTwoSet(b, SymmetricDifference[int], SymmetricDifference[string])
}
