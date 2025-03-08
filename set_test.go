package sets

import (
	"iter"
	"maps"
	"slices"
	"sync"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"pgregory.net/rapid"
)

func sortInts() cmp.Option {
	return cmpopts.SortSlices(
		func(a, b int) bool { return a < b },
	)
}

type SetStateMachine struct {
	set       Set[int]
	stateI    map[int]int
	stateO    []int
	newFn     func() Set[int]
	newFromFn func(seq iter.Seq[int]) Set[int]
}

func TestMap(t *testing.T) {
	t.Parallel()

	setStateMachine := &SetStateMachine{
		set:       New[int](),
		newFn:     func() Set[int] { return New[int]() },
		newFromFn: func(seq iter.Seq[int]) Set[int] { return NewFrom[int](seq) },
		stateI:    make(map[int]int),
	}
	rapid.Check(t, func(t *rapid.T) {
		t.Repeat(rapid.StateMachineActions(setStateMachine))
	})
}

func TestSyncMap(t *testing.T) {
	t.Parallel()

	setStateMachine := &SetStateMachine{
		set:       NewSync[int](),
		newFn:     func() Set[int] { return NewSync[int]() },
		newFromFn: func(seq iter.Seq[int]) Set[int] { return NewSyncFrom[int](seq) },
		stateI:    make(map[int]int),
	}
	rapid.Check(t, func(t *rapid.T) {
		t.Repeat(rapid.StateMachineActions(setStateMachine))
	})
}

func TestLockedMap(t *testing.T) {
	t.Parallel()

	setStateMachine := &SetStateMachine{
		set:       NewLocked[int](),
		newFn:     func() Set[int] { return NewLocked[int]() },
		newFromFn: func(seq iter.Seq[int]) Set[int] { return NewLockedFrom(seq) },
		stateI:    make(map[int]int),
	}
	rapid.Check(t, func(t *rapid.T) {
		t.Repeat(rapid.StateMachineActions(setStateMachine))
	})
}

func TestOrderedMap(t *testing.T) {
	t.Parallel()

	setStateMachine := &SetStateMachine{
		set:       NewOrdered[int](),
		newFn:     func() Set[int] { return NewOrdered[int]() },
		newFromFn: func(seq iter.Seq[int]) Set[int] { return NewOrderedFrom(seq) },
		stateI:    make(map[int]int),
	}
	rapid.Check(t, func(t *rapid.T) {
		t.Repeat(rapid.StateMachineActions(setStateMachine))
	})
}

func TestLockedOrdered(t *testing.T) {
	t.Parallel()

	setStateMachine := &SetStateMachine{
		set:       NewLockedOrdered[int](),
		newFn:     func() Set[int] { return NewLockedOrdered[int]() },
		newFromFn: func(seq iter.Seq[int]) Set[int] { return NewLockedOrderedFrom(seq) },
		stateI:    make(map[int]int),
	}
	rapid.Check(t, func(t *rapid.T) {
		t.Repeat(rapid.StateMachineActions(setStateMachine))
	})
}

func (sm *SetStateMachine) Add(t *rapid.T) {
	i := rapid.Int().Draw(t, "Int")
	_, exists := sm.stateI[i]
	if sm.set.Add(i) == exists {
		t.Fatalf("expected %d to exist: %v", i, exists)
	}
	sm.add(t, i)
}

func (sm *SetStateMachine) Remove(t *rapid.T) {
	i := rapid.Int().Draw(t, "Int")
	_, exists := sm.stateI[i]
	if sm.set.Remove(i) != exists {
		t.Fatalf("expected %v to exist: %v", i, exists)
	}
	sm.remove(t, i)
}

func (sm *SetStateMachine) Contains(t *rapid.T) {
	i := rapid.Int().Draw(t, "Int")
	_, exists := sm.stateI[i]
	if got := exists != sm.set.Contains(i); got {
		t.Fatalf("expected %v to exist: %v", i, got)
	}
}
func (sm *SetStateMachine) Clone(t *rapid.T) {
	sm.set = sm.set.Clone()
}

func (sm *SetStateMachine) Intersection(t *rapid.T) {
	if len(sm.stateI) == 0 {
		t.Skip("no elements to intersect")
	}

	other := sm.newFromFn(slices.Values(
		rapid.SliceOfNDistinct(
			rapid.SampledFrom(slices.Collect(sm.set.Iterator)), 0, sm.set.Cardinality(), func(i int) int { return i },
		).Draw(t, "Intersecting Sets"),
	))

	got := Intersection(sm.set, other)
	expected := sm.newFn()
	for _, i := range sm.stateO {
		if other.Contains(i) && sm.set.Contains(i) {
			expected.Add(i)
		}
	}

	if !Equal(expected, got) {
		t.Fatalf("expected %v, got %v", Elements(expected), Elements(got))
	}
}

func (sm *SetStateMachine) add(t *rapid.T, i int) {
	if _, exist := sm.stateI[i]; exist {
		return
	}
	sm.stateO = append(sm.stateO, i)
	sm.stateI[i] = len(sm.stateO) - 1
}

func (sm *SetStateMachine) remove(t *rapid.T, i int) {
	d, exist := sm.stateI[i]
	if !exist {
		return
	}
	t.Logf("remove - d: %d, i: %d", d, i)
	sm.stateO = append(sm.stateO[:d], sm.stateO[d+1:]...)
	for i, v := range sm.stateO[d:] {
		sm.stateI[v] = d + i
	}
	delete(sm.stateI, i)
}

func (sm *SetStateMachine) AddSeq(t *rapid.T) {
	values := rapid.SliceOfNDistinct(rapid.Int().Filter(func(i int) bool { return !sm.set.Contains(i) }), 1, 20, func(i int) int { return i }).Draw(t, "Seq Values")
	n := AppendSeq(sm.set, slices.Values(values))
	if n != len(values) {
		t.Fatalf("expected %d elements to be added, got %d", len(values), n)
	}
	for _, i := range values {
		sm.add(t, i)
	}
}

func (sm *SetStateMachine) RemoveSeq(t *rapid.T) {
	if len(sm.stateI) == 0 {
		t.Skip("no elements to remove")
	}
	values := rapid.SliceOfNDistinct(rapid.SampledFrom(slices.Collect(sm.set.Iterator)), 1, len(sm.stateI), func(i int) int { return i }).Draw(t, "Seq Values")
	n := RemoveSeq(sm.set, slices.Values(values))
	if n != len(values) {
		t.Fatalf("expected %d elements to be removed, got %d", len(values), n)
	}
	for _, i := range values {
		sm.remove(t, i)
	}
}

func (sm *SetStateMachine) Union(t *rapid.T) {
	other := sm.newFromFn(slices.Values(rapid.SliceOfN(rapid.Int(), 0, 20).Draw(t, "Union Values")))
	for i := range other.Iterator {
		sm.add(t, i)
	}
	sm.set = Union(sm.set, other)
}

func (sm *SetStateMachine) Difference(t *rapid.T) {
	other := sm.newFromFn(slices.Values(rapid.SliceOfN(rapid.Int(), 0, 20).Draw(t, "Difference Values")))
	for i := range slices.Values(slices.Collect(maps.Keys(sm.stateI))) {
		if other.Contains(i) {
			sm.remove(t, i)
		}
	}
	sm.set = Difference(sm.set, other)
}

func (sm *SetStateMachine) SymmetricDifference(t *rapid.T) {
	other := sm.newFromFn(slices.Values(rapid.SliceOfN(rapid.Int(), 0, 20).Draw(t, "Symmetric Difference Values")))
	for i := range other.Iterator {
		if _, exists := sm.stateI[i]; exists {
			sm.remove(t, i)
		} else {
			sm.add(t, i)
		}
	}
	sm.set = SymmetricDifference(sm.set, other)
}

func (sm *SetStateMachine) Subset(t *rapid.T) {
	if len(sm.stateI) == 0 {
		t.Skip("no elements to check for subset")
	}
	other := sm.newFromFn(slices.Values(rapid.SliceOfNDistinct(rapid.SampledFrom(slices.Collect(sm.set.Iterator)), 1, len(sm.stateI), func(i int) int { return i }).Draw(t, "Subset Values")))
	if !Subset(other, sm.set) {
		t.Fatalf("expected %v to be a subset of %v", Elements(other), Elements(sm.set))
	}
}

func (sm *SetStateMachine) Superset(t *rapid.T) {
	if len(sm.stateI) == 0 {
		t.Skip("no elements to check for subset")
	}
	other := sm.newFromFn(slices.Values(rapid.SliceOfNDistinct(rapid.SampledFrom(slices.Collect(sm.set.Iterator)), 1, len(sm.stateI), func(i int) int { return i }).Draw(t, "Superset Values")))
	if !Superset(sm.set, other) {
		t.Fatalf("expected %v to be a superset of %v", Elements(sm.set), Elements(other))
	}
}

func (sm *SetStateMachine) Equal(t *rapid.T) {
	other := sm.set.Clone()
	if !Equal(sm.set, other) {
		t.Fatalf("expected %v to be equal to %v", Elements(sm.set), Elements(other))
	}
	other.Add(rapid.Int().Filter(func(i int) bool { return !sm.set.Contains(i) }).Draw(t, "Extra Value"))
	if Equal(sm.set, other) {
		t.Fatalf("expected %v to be different from %v", Elements(sm.set), Elements(other))
	}
}

func (sm *SetStateMachine) Elements(t *rapid.T) {
	elem := Elements(sm.set)
	if len(elem) != len(sm.stateI) {
		t.Fatalf("expected %d elements, got %d", len(sm.stateI), len(elem))
	}
	if len(elem) != sm.set.Cardinality() {
		t.Fatalf("expected %d elements, got %d", len(sm.stateI), sm.set.Cardinality())
	}
	for _, i := range elem {
		if _, exists := sm.stateI[i]; !exists {
			t.Fatalf("expected %d to exist", i)
		}
		if !sm.set.Contains(i) {
			t.Fatalf("expected %d to exist", i)
		}
	}
}

func (sm *SetStateMachine) ContainsSeq(t *rapid.T) {
	var values []int
	if len(sm.stateI) > 0 {
		// items in the set
		values = rapid.SliceOfNDistinct(rapid.SampledFrom(Elements(sm.set)), 1, len(sm.stateI), func(i int) int { return i }).Draw(t, "Seq Values")
	}
	if !ContainsSeq(sm.set, slices.Values(values)) {
		t.Fatalf("expected %v to be a subset of %v", values, Elements(sm.set))
	}
	// items not in the set
	values = rapid.SliceOfNDistinct(rapid.Int().Filter(func(i int) bool { return !sm.set.Contains(i) }), 1, 20, func(i int) int { return i }).Draw(t, "Seq Values")
	if ContainsSeq(sm.set, slices.Values(values)) {
		t.Fatalf("expected %v to not be a subset of %v", values, Elements(sm.set))
	}
}

func (sm *SetStateMachine) Disjoint(t *rapid.T) {
	if len(sm.stateI) == 0 {
		t.Skip("no elements to check for disjoint")
	}
	other := sm.newFromFn(slices.Values(rapid.SliceOfNDistinct(rapid.Int().Filter(func(i int) bool { return !sm.set.Contains(i) }), 1, 20, func(i int) int { return i }).Draw(t, "Disjoint Values")))
	if !Disjoint(sm.set, other) {
		t.Fatalf("expected %v and %v to be disjoint", Elements(sm.set), Elements(other))
	}
	other.Add(rapid.SampledFrom(Elements(sm.set)).Draw(t, "Common Value"))
	if Disjoint(sm.set, other) {
		t.Fatalf("expected %v and %v to not be disjoint", Elements(sm.set), Elements(other))
	}
}

func (sm *SetStateMachine) Clear(t *rapid.T) {
	if n := sm.set.Clear(); n != len(sm.stateI) {
		t.Fatalf("expected %d elements to be removed, got %d", len(sm.stateI), n)
	}
	sm.stateI = make(map[int]int)
	sm.stateO = nil
}

func (sm *SetStateMachine) Check(t *rapid.T) {
	if len(sm.stateI) != sm.set.Cardinality() {
		t.Fatalf("expected %d elements, got %d", len(sm.stateI), sm.set.Cardinality())
	}
	if len(sm.stateO) != sm.set.Cardinality() {
		t.Fatalf("expected %d elements, got %d", len(sm.stateO), sm.set.Cardinality())
	}

	if diff := cmp.Diff(slices.Collect(maps.Keys(sm.stateI)), Elements(sm.set), sortInts()); diff != "" {
		t.Fatalf("unexpected elements (-want +got):\n%s", diff)
	}
	t.Logf("set: %#v\n", sm.set)
	t.Logf("stateI: %#v\n", sm.stateI)
	t.Logf("stateO: %#v\n", sm.stateO)
}

func testSetConcurrency(t *testing.T, set Set[int]) {
	var started, finished sync.WaitGroup
	changes := make(chan int, 100)

	for i := range 20 {
		started.Add(5)
		finished.Add(5)
		go func(base int) {
			started.Done()
			started.Wait()
			for i := range (base + 1) * 100 {
				set.Add(i)
			}
			finished.Done()
		}(i)

		go func(base int) {
			started.Done()
			started.Wait()
			var x int
			for range (base + 1) * 100 {
				x = set.Cardinality()
			}
			_ = x
			finished.Done()
		}(i)

		go func(base int) {
			started.Done()
			started.Wait()
			var x bool
			for i := range (base + 1) * 100 {
				x = set.Contains(i)
			}
			_ = x
			finished.Done()
		}(i)

		go func(base int) {
			started.Done()
			started.Wait()
			for i := range (base + 1) * 100 {
				set.Remove(i)
			}
			finished.Done()
		}(i)

		go func(base int) {
			other := New[int]()
			for i := range (base + 1) * 100 {
				other.Add(i)
			}
			started.Done()
			started.Wait()
			AppendSeq(other, set.Iterator)
			RemoveSeq(set, other.Iterator)
			finished.Done()
		}(i)
	}

	go func() {
		for i := range changes {
			set.Add(i)
		}
	}()

	finished.Wait()
	close(changes)
}
func TestLocked_Concurrency(t *testing.T) {
	t.Parallel()
	testSetConcurrency(t,
		NewLockedFrom(slices.Values([]int{9, 8, 7, 6, 5, 4, 3, 2, 1})),
	)
}

func TestLockedOrdered_Concurrency(t *testing.T) {
	t.Parallel()
	testSetConcurrency(t,
		NewLockedOrderedFrom(slices.Values([]int{9, 8, 7, 6, 5, 4, 3, 2, 1})),
	)
}

func TestSync_Concurrency(t *testing.T) {
	t.Parallel()
	testSetConcurrency(t,
		NewSyncFrom(slices.Values([]int{9, 8, 7, 6, 5, 4, 3, 2, 1})),
	)
}
