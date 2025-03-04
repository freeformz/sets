package sets

import (
	"fmt"
	"iter"
	"maps"
	"slices"
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
	state     map[int]struct{}
	newFn     func() Set[int]
	newFromFn func(seq iter.Seq[int]) Set[int]
}

func TestMapSets(t *testing.T) {
	setStateMachine := &SetStateMachine{
		set:       New[int](),
		newFn:     func() Set[int] { return New[int]() },
		newFromFn: func(seq iter.Seq[int]) Set[int] { return NewFrom[int](seq) },
		state:     make(map[int]struct{}),
	}
	rapid.Check(t, func(t *rapid.T) {
		t.Repeat(rapid.StateMachineActions(setStateMachine))
	})
}

func TestSyncMapSets(t *testing.T) {
	setStateMachine := &SetStateMachine{
		set:       NewSync[int](),
		newFn:     func() Set[int] { return NewSync[int]() },
		newFromFn: func(seq iter.Seq[int]) Set[int] { return NewSyncFrom[int](seq) },
		state:     make(map[int]struct{}),
	}
	rapid.Check(t, func(t *rapid.T) {
		t.Repeat(rapid.StateMachineActions(setStateMachine))
	})
}

func TestLockedMapSets(t *testing.T) {
	setStateMachine := &SetStateMachine{
		set:       NewSync[int](),
		newFn:     func() Set[int] { return NewLocked[int]() },
		newFromFn: func(seq iter.Seq[int]) Set[int] { return NewLockedFrom(seq) },
		state:     make(map[int]struct{}),
	}
	rapid.Check(t, func(t *rapid.T) {
		t.Repeat(rapid.StateMachineActions(setStateMachine))
	})
}

func TestOrderedMapSets(t *testing.T) {
	setStateMachine := &SetStateMachine{
		set:       NewSync[int](),
		newFn:     func() Set[int] { return NewOrdered[int]() },
		newFromFn: func(seq iter.Seq[int]) Set[int] { return NewOrderedFrom(seq) },
		state:     make(map[int]struct{}),
	}
	rapid.Check(t, func(t *rapid.T) {
		t.Repeat(rapid.StateMachineActions(setStateMachine))
	})
}

func (sm *SetStateMachine) Add(t *rapid.T) {
	i := rapid.Int().Draw(t, "Int")
	_, exists := sm.state[i]
	if sm.set.Add(i) == exists {
		t.Fatalf("expected %d to exist: %v", i, exists)
	}
	sm.state[i] = struct{}{}
}

func (sm *SetStateMachine) Remove(t *rapid.T) {
	i := rapid.Int().Draw(t, "Int")
	_, exists := sm.state[i]
	if sm.set.Remove(i) != exists {
		t.Fatalf("expected %v to exist: %v", i, exists)
	}
	delete(sm.state, i)
}

func (sm *SetStateMachine) Contains(t *rapid.T) {
	i := rapid.Int().Draw(t, "Int")
	_, exists := sm.state[i]
	if got := exists != sm.set.Contains(i); got {
		t.Fatalf("expected %v to exist: %v", i, got)
	}
}
func (sm *SetStateMachine) Clone(t *rapid.T) {
	sm.set = sm.set.Clone()
}

func (sm *SetStateMachine) Intersection(t *rapid.T) {
	if len(sm.state) == 0 {
		t.Skip("no elements to intersect")
	}

	other := sm.newFromFn(slices.Values(
		rapid.SliceOfNDistinct(
			rapid.SampledFrom(slices.Collect(sm.set.Iterator)), 0, sm.set.Cardinality(), func(i int) int { return i },
		).Draw(t, "Intersecting Sets"),
	))

	got := Intersection(sm.set, other)
	expected := sm.newFn()
	for i := range sm.state {
		if other.Contains(i) {
			expected.Add(i)
		}
	}

	if !Equal(got, expected) {
		t.Fatalf("expected %v, got %v", Elements(expected), Elements(got))
	}
}

func (sm *SetStateMachine) AddSeq(t *rapid.T) {
	values := rapid.SliceOfNDistinct(rapid.Int().Filter(func(i int) bool { return !sm.set.Contains(i) }), 1, 20, func(i int) int { return i }).Draw(t, "Seq Values")
	n := AppendSeq(sm.set, slices.Values(values))
	if n != len(values) {
		t.Fatalf("expected %d elements to be added, got %d", len(values), n)
	}
	for _, i := range values {
		sm.state[i] = struct{}{}
	}
}

func (sm *SetStateMachine) RemoveSeq(t *rapid.T) {
	if len(sm.state) == 0 {
		t.Skip("no elements to remove")
	}
	values := rapid.SliceOfNDistinct(rapid.SampledFrom(slices.Collect(sm.set.Iterator)), 1, len(sm.state), func(i int) int { return i }).Draw(t, "Seq Values")
	n := RemoveSeq(sm.set, slices.Values(values))
	if n != len(values) {
		t.Fatalf("expected %d elements to be removed, got %d", len(values), n)
	}
	for _, i := range values {
		delete(sm.state, i)
	}
}

func (sm *SetStateMachine) Union(t *rapid.T) {
	other := sm.newFromFn(slices.Values(rapid.SliceOfN(rapid.Int(), 0, 20).Draw(t, "Union Values")))
	for i := range other.Iterator {
		sm.state[i] = struct{}{}
	}
	sm.set = Union(sm.set, other)
}

func (sm *SetStateMachine) Difference(t *rapid.T) {
	other := sm.newFromFn(slices.Values(rapid.SliceOfN(rapid.Int(), 0, 20).Draw(t, "Difference Values")))
	for i := range slices.Values(slices.Collect(maps.Keys(sm.state))) {
		if other.Contains(i) {
			delete(sm.state, i)
		}
	}
	sm.set = Difference(sm.set, other)
}

func (sm *SetStateMachine) SymmetricDifference(t *rapid.T) {
	other := sm.newFromFn(slices.Values(rapid.SliceOfN(rapid.Int(), 0, 20).Draw(t, "Symmetric Difference Values")))
	for i := range other.Iterator {
		if _, exists := sm.state[i]; exists {
			delete(sm.state, i)
		} else {
			sm.state[i] = struct{}{}
		}
	}
	sm.set = SymmetricDifference(sm.set, other)
}

func (sm *SetStateMachine) Subset(t *rapid.T) {
	if len(sm.state) == 0 {
		t.Skip("no elements to check for subset")
	}
	other := sm.newFromFn(slices.Values(rapid.SliceOfNDistinct(rapid.SampledFrom(slices.Collect(sm.set.Iterator)), 1, len(sm.state), func(i int) int { return i }).Draw(t, "Subset Values")))
	if !Subset(other, sm.set) {
		t.Fatalf("expected %v to be a subset of %v", Elements(other), Elements(sm.set))
	}
}

func (sm *SetStateMachine) Superset(t *rapid.T) {
	if len(sm.state) == 0 {
		t.Skip("no elements to check for subset")
	}
	other := sm.newFromFn(slices.Values(rapid.SliceOfNDistinct(rapid.SampledFrom(slices.Collect(sm.set.Iterator)), 1, len(sm.state), func(i int) int { return i }).Draw(t, "Superset Values")))
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
	if len(elem) != len(sm.state) {
		t.Fatalf("expected %d elements, got %d", len(sm.state), len(elem))
	}
	if len(elem) != sm.set.Cardinality() {
		t.Fatalf("expected %d elements, got %d", len(sm.state), sm.set.Cardinality())
	}
	for _, i := range elem {
		if _, exists := sm.state[i]; !exists {
			t.Fatalf("expected %d to exist", i)
		}
		if !sm.set.Contains(i) {
			t.Fatalf("expected %d to exist", i)
		}
	}
}

func (sm *SetStateMachine) ContainsSeq(t *rapid.T) {
	var values []int
	if len(sm.state) > 0 {
		// items in the set
		values = rapid.SliceOfNDistinct(rapid.SampledFrom(Elements(sm.set)), 1, len(sm.state), func(i int) int { return i }).Draw(t, "Seq Values")
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
	if len(sm.state) == 0 {
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
	if n := Clear(sm.set); n != len(sm.state) {
		t.Fatalf("expected %d elements to be removed, got %d", len(sm.state), n)
	}
	sm.state = make(map[int]struct{})
}

func (sm *SetStateMachine) Check(t *rapid.T) {
	if len(sm.state) != sm.set.Cardinality() {
		t.Fatalf("expected %d elements, got %d", len(sm.state), sm.set.Cardinality())
	}

	if diff := cmp.Diff(slices.Collect(maps.Keys(sm.state)), Elements(sm.set), sortInts()); diff != "" {
		t.Fatalf("unexpected elements (-want +got):\n%s", diff)
	}
}

func ExampleMapSets_Iterator() {
	ints := New[int]()
	ints.Add(5)
	ints.Add(3)
	ints.Add(2)
	ints.Add(4)
	ints.Add(1)

	out := make([]int, 0, ints.Cardinality())
	for i := range ints.Iterator {
		out = append(out, i)
	}

	// sort the values for consistent output
	slices.Sort(out)
	for _, i := range out {
		fmt.Println(i)
	}
	// Output:
	// 1
	// 2
	// 3
	// 4
	// 5
}

func ExampleOrderedSets() {
	ints := NewOrdered[int]()
	ints.Add(5)
	ints.Add(3)

	AppendSeq(ints, slices.Values([]int{2, 4, 1}))
	AppendSeq(ints, slices.Values([]int{5, 6, 1}))

	out := make([]int, 0, ints.Cardinality())
	for i := range ints.Iterator {
		out = append(out, i)
	}

	for _, i := range out {
		fmt.Println(i)
	}
	// Output:
	// 5
	// 3
	// 2
	// 4
	// 1
	// 6
}
