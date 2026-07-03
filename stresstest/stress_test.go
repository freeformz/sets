// Package stresstest holds heavier randomized stress checks that complement the
// property-based state machine tests in the root package. It compares the
// Ordered implementation against a simple slice-backed reference model across
// many randomized operation sequences, and hammers the concurrent types to
// catch regressions in their concurrency guarantees.
package stresstest

import (
	"math/rand/v2"
	"slices"
	"sync"
	"testing"

	"github.com/freeformz/sets"
)

// model is a slice-backed reference implementation of an insertion-ordered set.
type model struct{ el []int }

func (r *model) add(v int) bool {
	if slices.Contains(r.el, v) {
		return false
	}
	r.el = append(r.el, v)
	return true
}

func (r *model) remove(v int) bool {
	i := slices.Index(r.el, v)
	if i < 0 {
		return false
	}
	r.el = slices.Delete(r.el, i, i+1)
	return true
}

// TestOrderedDifferential runs randomized Add/Remove/Sort/Pop/Clear sequences
// against Ordered and the reference model, verifying Cardinality, Iterator,
// At, Index, Ordered, and Backwards after every operation. The element domain
// (50) is deliberately small relative to the operation count so sequences
// repeatedly grow, shrink, and empty the set, exercising the gap buffer's
// compaction and Fenwick tree rebuild paths.
func TestOrderedDifferential(t *testing.T) {
	t.Parallel()

	trials := 250
	if testing.Short() {
		trials = 25
	}

	for trial := range trials {
		rng := rand.New(rand.NewPCG(uint64(trial), 0xda7aba5e))
		s := sets.NewOrdered[int]()
		var r model
		for op := range 300 {
			step(t, rng, s, &r, trial, op)
			check(t, s, &r, trial, op)
		}
	}
}

// step applies one random operation to both the set and the reference model.
func step(t *testing.T, rng *rand.Rand, s *sets.Ordered[int], r *model, trial, op int) {
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
		s.Sort()
		slices.Sort(r.el)
	case 8:
		v, ok := s.Pop()
		if ok != (len(r.el) > 0) {
			t.Fatalf("trial %d op %d: Pop() ok = %t with %d elements", trial, op, ok, len(r.el))
		}
		if ok && !r.remove(v) {
			t.Fatalf("trial %d op %d: Pop() returned %d, not in the set", trial, op, v)
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

// check verifies every read-side invariant of s against the reference model.
func check(t *testing.T, s *sets.Ordered[int], r *model, trial, op int) {
	t.Helper()
	want := r.el

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
	if idx := s.Index(999); idx != -1 { // 999 is outside the generated element domain
		t.Fatalf("trial %d op %d: Index(999) = %d, want -1", trial, op, idx)
	}

	next := 0
	s.Ordered(func(i, v int) bool {
		if i != next || want[i] != v {
			t.Fatalf("trial %d op %d: Ordered yielded (%d, %d), want (%d, %d)", trial, op, i, v, next, want[next])
		}
		next++
		return true
	})

	next = len(want) - 1
	s.Backwards(func(i, v int) bool {
		if i != next || want[i] != v {
			t.Fatalf("trial %d op %d: Backwards yielded (%d, %d), want (%d, %d)", trial, op, i, v, next, want[next])
		}
		next--
		return true
	})
}

// TestSyncMapConcurrentClearCount verifies that concurrent Clear calls on the
// same SyncMap count each removed element exactly once: the per-call counts
// must sum to the number of elements that were in the set.
func TestSyncMapConcurrentClearCount(t *testing.T) {
	t.Parallel()

	const n = 1000
	trials := 100
	if testing.Short() {
		trials = 10
	}

	for trial := range trials {
		s := sets.NewSyncMap[int]()
		for i := range n {
			s.Add(i)
		}

		var wg sync.WaitGroup
		counts := make([]int, 4)
		for g := range counts {
			wg.Go(func() {
				counts[g] = s.Clear()
			})
		}
		wg.Wait()

		var total int
		for _, c := range counts {
			total += c
		}
		if total != n {
			t.Fatalf("trial %d: concurrent Clear() counts sum to %d, want %d", trial, total, n)
		}
		if got := s.Cardinality(); got != 0 {
			t.Fatalf("trial %d: Cardinality() = %d after Clear, want 0", trial, got)
		}
	}
}

// TestLockedWrappingPreservesOrder verifies that a Locked set wrapping an
// ordered set keeps the wrapped set's insertion-order semantics through Clone
// and NewEmpty.
func TestLockedWrappingPreservesOrder(t *testing.T) {
	t.Parallel()

	wrapped := sets.NewLockedWrapping(sets.NewOrderedWith(3, 1, 2))

	clone := wrapped.Clone()
	if got, want := slices.Collect(clone.Iterator), []int{3, 1, 2}; !slices.Equal(got, want) {
		t.Fatalf("Clone().Iterator yielded %v, want %v", got, want)
	}

	empty := wrapped.NewEmpty()
	for _, v := range []int{9, 4, 7} {
		empty.Add(v)
	}
	if got, want := slices.Collect(empty.Iterator), []int{9, 4, 7}; !slices.Equal(got, want) {
		t.Fatalf("NewEmpty() set yielded %v, want %v", got, want)
	}
}

// TestLockedOrderedClonePreservesOrder verifies that cloning a LockedOrdered
// set returns an ordered set with the same insertion order.
func TestLockedOrderedClonePreservesOrder(t *testing.T) {
	t.Parallel()

	s := sets.NewLockedOrderedWith(5, 4, 6)

	clone := s.Clone()
	if got, want := slices.Collect(clone.Iterator), []int{5, 4, 6}; !slices.Equal(got, want) {
		t.Fatalf("Clone().Iterator yielded %v, want %v", got, want)
	}
	if _, ok := clone.(sets.OrderedSet[int]); !ok {
		t.Fatalf("Clone() returned %T, want an OrderedSet", clone)
	}
}
