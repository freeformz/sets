package stresstest

import (
	"math/rand/v2"
	"slices"
	"testing"

	"github.com/freeformz/sets"
)

// unsortedModel is a slice-backed reference implementation of a set. Membership is tracked in
// insertion order and sorted on demand, so it shares no logic with SortedSet's
// binary-search-and-shift approach.
type unsortedModel struct{ el []int }

func (r *unsortedModel) add(v int) bool {
	if slices.Contains(r.el, v) {
		return false
	}
	r.el = append(r.el, v)
	return true
}

func (r *unsortedModel) remove(v int) bool {
	i := slices.Index(r.el, v)
	if i < 0 {
		return false
	}
	r.el = slices.Delete(r.el, i, i+1)
	return true
}

// sorted returns the expected element view: ascending order.
func (r *unsortedModel) sorted() []int {
	out := slices.Clone(r.el)
	slices.Sort(out)
	return out
}

// TestSortedSetDifferential runs randomized Add/Remove/Sort/Pop/Clear sequences against SortedSet
// and an unsorted reference model, verifying Cardinality, Iterator, At, Index, Ordered, Backwards,
// and Range after every operation. The element domain (50) is deliberately small relative to the
// operation count so sequences repeatedly grow, shrink, and empty the set, exercising the insertion
// and deletion shift paths at every position.
func TestSortedSetDifferential(t *testing.T) {
	t.Parallel()

	trials := 250
	if testing.Short() {
		trials = 25
	}

	for trial := range trials {
		rng := rand.New(rand.NewPCG(uint64(trial), 0x50e7ed5e7))
		s := sets.NewSortedSet[int]()
		var r unsortedModel
		for op := range 300 {
			sortedStep(t, rng, s, &r, trial, op)
			sortedCheck(t, rng, s, &r, trial, op)
		}
	}
}

// sortedStep applies one random operation to both the set and the reference model.
func sortedStep(t *testing.T, rng *rand.Rand, s *sets.SortedSet[int], r *unsortedModel, trial, op int) {
	t.Helper()
	switch rng.IntN(10) {
	case 0, 1, 2, 3:
		v := rng.IntN(50)
		if got, want := s.Add(v), r.add(v); got != want {
			t.Fatalf("trial %d op %d: Add(%d) = %t, want %t", trial, op, v, got, want)
		}
	case 4, 5, 6:
		v := rng.IntN(50)
		if got, want := s.Remove(v), r.remove(v); got != want {
			t.Fatalf("trial %d op %d: Remove(%d) = %t, want %t", trial, op, v, got, want)
		}
	case 7:
		s.Sort() // no-op for SortedSet; the model needs no change either
	case 8:
		v, ok := s.Pop()
		if ok != (len(r.el) > 0) {
			t.Fatalf("trial %d op %d: Pop() ok = %t with %d elements", trial, op, ok, len(r.el))
		}
		if ok {
			if !r.remove(v) {
				t.Fatalf("trial %d op %d: Pop() returned %d, not in the set", trial, op, v)
			}
		}
	case 9:
		if rng.IntN(10) == 0 {
			if got := s.Clear(); got != len(r.el) {
				t.Fatalf("trial %d op %d: Clear() = %d, want %d", trial, op, got, len(r.el))
			}
			r.el = nil
		}
	}
}

// sortedCheck verifies every read-side invariant of s against the reference model.
func sortedCheck(t *testing.T, rng *rand.Rand, s *sets.SortedSet[int], r *unsortedModel, trial, op int) {
	t.Helper()
	want := r.sorted()

	if got := s.Cardinality(); got != len(want) {
		t.Fatalf("trial %d op %d: Cardinality() = %d, want %d", trial, op, got, len(want))
	}
	if got := slices.Collect(s.Iterator); !slices.Equal(got, want) {
		t.Fatalf("trial %d op %d: Iterator yielded %v, want %v", trial, op, got, want)
	}

	for i, v := range want {
		got, ok := s.At(i)
		if !ok || got != v {
			t.Fatalf("trial %d op %d: At(%d) = %d, %t, want %d, true", trial, op, i, got, ok, v)
		}
		if idx := s.Index(v); idx != i {
			t.Fatalf("trial %d op %d: Index(%d) = %d, want %d", trial, op, v, idx, i)
		}
	}
	if _, ok := s.At(-1); ok {
		t.Fatalf("trial %d op %d: At(-1) ok = true", trial, op)
	}
	if _, ok := s.At(len(want)); ok {
		t.Fatalf("trial %d op %d: At(%d) ok = true", trial, op, len(want))
	}

	var ordered []int
	s.Ordered(func(i, v int) bool {
		if i != len(ordered) {
			t.Fatalf("trial %d op %d: Ordered index = %d, want %d", trial, op, i, len(ordered))
		}
		ordered = append(ordered, v)
		return true
	})
	if !slices.Equal(ordered, want) {
		t.Fatalf("trial %d op %d: Ordered yielded %v, want %v", trial, op, ordered, want)
	}

	var backwards []int
	s.Backwards(func(i, v int) bool {
		if i != len(want)-1-len(backwards) {
			t.Fatalf("trial %d op %d: Backwards index = %d, want %d", trial, op, i, len(want)-1-len(backwards))
		}
		backwards = append(backwards, v)
		return true
	})
	slices.Reverse(backwards)
	if !slices.Equal(backwards, want) {
		t.Fatalf("trial %d op %d: Backwards yielded %v, want %v", trial, op, backwards, want)
	}

	// Range with random bounds (possibly inverted) against a filtered model view.
	lo, hi := rng.IntN(60)-5, rng.IntN(60)-5
	var wantRange []int
	for _, v := range want {
		if v >= lo && v <= hi {
			wantRange = append(wantRange, v)
		}
	}
	if got := slices.Collect(s.Range(lo, hi)); !slices.Equal(got, wantRange) {
		t.Fatalf("trial %d op %d: Range(%d, %d) yielded %v, want %v", trial, op, lo, hi, got, wantRange)
	}
}
