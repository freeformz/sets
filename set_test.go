package sets

import (
	"database/sql/driver"
	"encoding/json"
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
	set    Set[int]
	stateI map[int]int
	stateO []int
}

func TestMap(t *testing.T) {
	t.Parallel()

	setStateMachine := &SetStateMachine{
		set:    New[int](),
		stateI: make(map[int]int),
	}
	rapid.Check(t, func(t *rapid.T) {
		t.Repeat(rapid.StateMachineActions(setStateMachine))
	})
}

func TestSyncMap(t *testing.T) {
	t.Parallel()

	setStateMachine := &SetStateMachine{
		set:    NewSyncMap[int](),
		stateI: make(map[int]int),
	}
	rapid.Check(t, func(t *rapid.T) {
		t.Repeat(rapid.StateMachineActions(setStateMachine))
	})
}

func TestLockedMap(t *testing.T) {
	t.Parallel()

	setStateMachine := &SetStateMachine{
		set:    NewLocked[int](),
		stateI: make(map[int]int),
	}
	rapid.Check(t, func(t *rapid.T) {
		t.Repeat(rapid.StateMachineActions(setStateMachine))
	})
}

func TestOrderedMap(t *testing.T) {
	t.Parallel()

	setStateMachine := &SetStateMachine{
		set:    NewOrdered[int](),
		stateI: make(map[int]int),
	}
	rapid.Check(t, func(t *rapid.T) {
		t.Repeat(rapid.StateMachineActions(setStateMachine))
	})
}

func TestLockedOrdered(t *testing.T) {
	t.Parallel()

	setStateMachine := &SetStateMachine{
		set:    NewLockedOrdered[int](),
		stateI: make(map[int]int),
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

type Embed[M comparable] struct {
	ASet Set[M]
}

func (sm *SetStateMachine) Pop(t *rapid.T) {
	c := sm.set.Clone()
	i, ok := sm.set.Pop()
	t.Logf("Pop: %d, %v", i, ok)
	if ok != !IsEmpty(c) {
		t.Fatalf("expected Pop() to not be %v, got %v", !IsEmpty(c), ok)
	}
	if ok {
		if !c.Contains(i) {
			t.Fatalf("expected %v to exist in copy", i)
		}
		sm.remove(t, i)
	}
}

func (sm *SetStateMachine) JSON(t *rapid.T) {
	d, err := json.Marshal(sm.set)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	t.Log("JSON:", string(d))
	if err = json.Unmarshal(d, &sm.set); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	v := Embed[int]{ASet: sm.set}
	d, err = json.Marshal(v)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	t.Log("JSON:", string(d))
	v = Embed[int]{ASet: sm.set.NewEmpty()}
	if err := json.Unmarshal(d, &v); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !Equal(sm.set, v.ASet) {
		t.Fatalf("expected %v, got %v", Elements(sm.set), Elements(v.ASet))
	}
}

func (sm *SetStateMachine) Intersection(t *rapid.T) {
	if len(sm.stateI) == 0 {
		t.Skip("no elements to intersect")
	}

	other := sm.set.NewEmpty()
	AppendSeq(other, slices.Values(
		rapid.SliceOfNDistinct(
			rapid.SampledFrom(slices.Collect(sm.set.Iterator)), 0, sm.set.Cardinality(), func(i int) int { return i },
		).Draw(t, "Intersecting Sets"),
	))

	got := Intersection(sm.set, other)
	expected := sm.set.NewEmpty()
	for _, i := range sm.stateO {
		if other.Contains(i) && sm.set.Contains(i) {
			expected.Add(i)
		}
	}

	if !Equal(expected, got) {
		t.Fatalf("expected %v, got %v", Elements(expected), Elements(got))
	}
}

func (sm *SetStateMachine) add(_ *rapid.T, i int) {
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
	other := sm.set.NewEmpty()
	AppendSeq(other, slices.Values(rapid.SliceOfN(rapid.Int(), 0, 20).Draw(t, "Union Values")))
	for i := range other.Iterator {
		sm.add(t, i)
	}
	sm.set = Union(sm.set, other)
}

func (sm *SetStateMachine) Difference(t *rapid.T) {
	other := sm.set.NewEmpty()
	AppendSeq(other, slices.Values(rapid.SliceOfN(rapid.Int(), 0, 20).Draw(t, "Difference Values")))
	for i := range slices.Values(slices.Collect(maps.Keys(sm.stateI))) {
		if other.Contains(i) {
			sm.remove(t, i)
		}
	}
	sm.set = Difference(sm.set, other)
}

func (sm *SetStateMachine) SymmetricDifference(t *rapid.T) {
	other := sm.set.NewEmpty()
	AppendSeq(other, slices.Values(rapid.SliceOfN(rapid.Int(), 0, 20).Draw(t, "Symmetric Difference Values")))
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
	other := sm.set.NewEmpty()
	AppendSeq(other, slices.Values(rapid.SliceOfNDistinct(rapid.SampledFrom(slices.Collect(sm.set.Iterator)), 1, len(sm.stateI), func(i int) int { return i }).Draw(t, "Subset Values")))
	if !Subset(other, sm.set) {
		t.Fatalf("expected %v to be a subset of %v", Elements(other), Elements(sm.set))
	}
}

func (sm *SetStateMachine) Superset(t *rapid.T) {
	if len(sm.stateI) == 0 {
		t.Skip("no elements to check for subset")
	}
	other := sm.set.NewEmpty()
	AppendSeq(other, slices.Values(rapid.SliceOfNDistinct(rapid.SampledFrom(slices.Collect(sm.set.Iterator)), 1, len(sm.stateI), func(i int) int { return i }).Draw(t, "Superset Values")))
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
	other := sm.set.NewEmpty()
	AppendSeq(other, slices.Values(rapid.SliceOfNDistinct(rapid.Int().Filter(func(i int) bool { return !sm.set.Contains(i) }), 1, 20, func(i int) int { return i }).Draw(t, "Disjoint Values")))
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
	t.Logf("set: %#v\n", sm.set)
	t.Logf("stateI: %#v\n", sm.stateI)
	t.Logf("stateO: %#v\n", sm.stateO)

	if len(sm.stateI) != sm.set.Cardinality() {
		t.Fatalf("expected %d elements, got %d", len(sm.stateI), sm.set.Cardinality())
	}
	if len(sm.stateO) != sm.set.Cardinality() {
		t.Fatalf("expected %d elements, got %d", len(sm.stateO), sm.set.Cardinality())
	}

	cmps := sm.set.NewEmpty()
	AppendSeq(cmps, slices.Values(sm.stateO))
	if !Equal(sm.set, cmps) {
		t.Fatalf("expected %v to be equal to %v", Elements(sm.set), Elements(cmps))
	}

	// ensure the set is comparable
	if diff := cmp.Diff(sm.set, cmps, cmp.Comparer(Equal[int])); diff != "" {
		t.Fatalf("unexpected elements (-want +got):\n%s", diff)
	}

	// ensure the set is comparable when embedded
	type Test struct {
		S Set[int]
	}

	if diff := cmp.Diff(Test{S: sm.set}, Test{S: cmps}, cmp.Comparer(Equal[int])); diff != "" {
		t.Fatalf("unexpected elements (-want +got):\n%s", diff)
	}

	if diff := cmp.Diff(slices.Collect(maps.Keys(sm.stateI)), Elements(sm.set), sortInts()); diff != "" {
		t.Fatalf("unexpected elements (-want +got):\n%s", diff)
	}
}

func (sm *SetStateMachine) Any(t *rapid.T) {
	threshold := rapid.Int().Draw(t, "Threshold")
	expected := false
	for k := range sm.stateI {
		if k > threshold {
			expected = true
			break
		}
	}
	got := Any(sm.set, func(k int) bool { return k > threshold })
	if got != expected {
		t.Fatalf("Any(> %d): expected %v, got %v", threshold, expected, got)
	}
}

func (sm *SetStateMachine) All(t *rapid.T) {
	threshold := rapid.Int().Draw(t, "Threshold")
	expected := true
	for k := range sm.stateI {
		if k <= threshold {
			expected = false
			break
		}
	}
	got := All(sm.set, func(k int) bool { return k > threshold })
	if got != expected {
		t.Fatalf("All(> %d): expected %v, got %v", threshold, expected, got)
	}
}

func (sm *SetStateMachine) ContainsAllAction(t *rapid.T) {
	if len(sm.stateI) == 0 {
		// test empty elements on empty set
		if !ContainsAll(sm.set) {
			t.Fatalf("ContainsAll with no args should return true")
		}
		return
	}
	// elements that are in the set
	values := rapid.SliceOfNDistinct(rapid.SampledFrom(slices.Collect(sm.set.Iterator)), 1, len(sm.stateI), func(i int) int { return i }).Draw(t, "ContainsAll Values")
	if !ContainsAll(sm.set, values...) {
		t.Fatalf("expected ContainsAll to be true for %v", values)
	}
	// add an element not in the set
	extra := rapid.Int().Filter(func(i int) bool { return !sm.set.Contains(i) }).Draw(t, "Extra Value")
	values = append(values, extra)
	if ContainsAll(sm.set, values...) {
		t.Fatalf("expected ContainsAll to be false for %v", values)
	}
}

func (sm *SetStateMachine) ContainsAnyAction(t *rapid.T) {
	if len(sm.stateI) == 0 {
		if ContainsAny(sm.set) {
			t.Fatalf("ContainsAny with no args should return false")
		}
		return
	}
	// at least one element in the set
	elem := rapid.SampledFrom(slices.Collect(sm.set.Iterator)).Draw(t, "ContainsAny Value")
	extra := rapid.Int().Filter(func(i int) bool { return !sm.set.Contains(i) }).Draw(t, "Extra Value")
	if !ContainsAny(sm.set, elem, extra) {
		t.Fatalf("expected ContainsAny to be true for %v", []int{elem, extra})
	}
	// all elements not in the set
	notIn := rapid.SliceOfNDistinct(rapid.Int().Filter(func(i int) bool { return !sm.set.Contains(i) }), 1, 5, func(i int) int { return i }).Draw(t, "NotIn Values")
	if ContainsAny(sm.set, notIn...) {
		t.Fatalf("expected ContainsAny to be false for %v", notIn)
	}
}

func (sm *SetStateMachine) Random(t *rapid.T) {
	cardBefore := sm.set.Cardinality()
	v, ok := Random(sm.set)
	if cardBefore == 0 {
		if ok {
			t.Fatalf("Random on empty set should return false")
		}
		return
	}
	if !ok {
		t.Fatalf("Random on non-empty set should return true")
	}
	if !sm.set.Contains(v) {
		t.Fatalf("Random returned %v which is not in the set", v)
	}
	if sm.set.Cardinality() != cardBefore {
		t.Fatalf("Random should not modify the set, cardinality changed from %d to %d", cardBefore, sm.set.Cardinality())
	}
}

func (sm *SetStateMachine) Valuer(t *rapid.T) {
	valuer, ok := any(sm.set).(driver.Valuer)
	if !ok {
		t.Fatalf("set does not implement driver.Valuer")
	}
	v, err := valuer.Value()
	if err != nil {
		t.Fatalf("Value() error: %v", err)
	}
	b, ok := v.([]byte)
	if !ok {
		t.Fatalf("Value() did not return []byte, got %T", v)
	}
	// round-trip: scan it back
	clone := sm.set.NewEmpty()
	scanner, ok := any(clone).(interface{ Scan(any) error })
	if !ok {
		t.Fatalf("set does not implement sql.Scanner")
	}
	if err := scanner.Scan(b); err != nil {
		t.Fatalf("Scan() error: %v", err)
	}
	if !Equal(sm.set, clone) {
		t.Fatalf("Value/Scan round-trip failed: expected %v, got %v", Elements(sm.set), Elements(clone))
	}
}

func testSetConcurrency(t *testing.T, set Set[int]) {
	t.Helper()

	started := make(chan struct{})
	start := make(chan struct{})
	var finished sync.WaitGroup

	goroutines := []func(int){
		func(base int) {
			started <- struct{}{}
			<-start
			for i := range (base + 1) * 100 {
				set.Add(i)
			}
			finished.Done()
		},
		func(base int) {
			var x int
			started <- struct{}{}
			<-start
			for range (base + 1) * 100 {
				x = set.Cardinality()
			}
			_ = x
			finished.Done()
		},
		func(base int) {
			var x bool
			started <- struct{}{}
			<-start
			for i := range (base + 1) * 100 {
				x = set.Contains(i)
			}
			_ = x
			finished.Done()
		},
		func(base int) {
			started <- struct{}{}
			<-start
			for i := range (base + 1) * 100 {
				set.Remove(i)
			}
			finished.Done()
		},
		func(base int) {
			other := New[int]()
			for i := range (base + 1) * 100 {
				other.Add(i)
			}
			started <- struct{}{}
			<-start
			AppendSeq(other, set.Iterator)
			RemoveSeq(set, other.Iterator)
			finished.Done()
		},
		func(base int) {
			var x int
			started <- struct{}{}
			<-start
			for j := range base {
				for i := range set.Iterator {
					x += i + j
				}
			}
			_ = x
			finished.Done()
		},
	}
	count := 20
	for i := range count {
		finished.Add(len(goroutines))
		for j := range len(goroutines) {
			go goroutines[j](i)
		}
	}

	for range count * len(goroutines) {
		<-started
	}
	close(start)

	finished.Wait()
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
		NewSyncMapFrom(slices.Values([]int{9, 8, 7, 6, 5, 4, 3, 2, 1})),
	)
}

func TestOrdered_Remove(t *testing.T) {
	t.Parallel()
	s := NewOrdered[int]()
	for i := range 5 {
		s.Add(i)
	}
	if s.Cardinality() != 5 {
		t.Fatalf("expected 5 elements, got %d", s.Cardinality())
	}
	s.Remove(2)
	if s.Cardinality() != 4 {
		t.Fatalf("expected 9 elements, got %d", s.Cardinality())
	}
	values := slices.Collect(s.Iterator)

	if values[0] != 0 {
		t.Fatalf("expected 0, got %d", values[0])
	}
	if values[1] != 1 {
		t.Fatalf("expected 1, got %d", values[1])
	}
	if values[2] != 3 {
		t.Fatalf("expected 3, got %d", values[2])
	}
	if values[3] != 4 {
		t.Fatalf("expected 4, got %d", values[3])
	}
}

func TestEqualOrdered(t *testing.T) {
	t.Parallel()
	s := NewOrdered[int]()
	for i := range 5 {
		s.Add(i)
	}
	if !EqualOrdered(s, s.Clone().(OrderedSet[int])) {
		t.Fatalf("expected s to be equal to itself")
	}
	s2 := NewOrdered[int]()
	for i := 4; i >= 0; i-- {
		s2.Add(i)
	}
	if EqualOrdered(s, s2) {
		t.Fatalf("expected s to be different from s2")
	}
	if !Equal(s, s2) {
		t.Fatalf("expected s to be equal to s2")
	}
}

func TestChunk_Ordered(t *testing.T) {
	t.Parallel()
	s := NewOrdered[int]()
	for i := range 21 {
		s.Add(i)
	}

	var i int
	for chunk := range Chunk(s, 5) {
		switch i {
		case 4: // deal with the odd chunk
			if chunk.Cardinality() != 1 {
				t.Fatalf("expected 1 elements, got %d", chunk.Cardinality())
			}
			if !Equal(chunk, NewOrderedFrom(slices.Values([]int{20}))) {
				t.Fatalf("expected 20, got %v", Elements(chunk))
			}
		default:
			if chunk.Cardinality() != 5 {
				t.Fatalf("expected 5 elements, got %d", chunk.Cardinality())
			}
			values := slices.Collect(chunk.Iterator)
			for j := range 5 {
				if values[j] != i*5+j {
					t.Fatalf("expected %d, got %d", i*5+j, values[j])
				}
			}
		}
		i++
	}
}

func TestChunk(t *testing.T) {
	t.Parallel()
	s := New[int]()
	for i := range 22 {
		s.Add(i)
	}

	var i int
	for chunk := range Chunk(s, 5) {
		switch i {
		case 4: // deal with the odd chunk
			if chunk.Cardinality() != 2 {
				t.Fatalf("expected 2 elements, got %d", chunk.Cardinality())
			}
		default:
			if chunk.Cardinality() != 5 {
				t.Fatalf("expected 5 elements, got %d", chunk.Cardinality())
			}
		}
		if !Subset(chunk, s) {
			t.Fatalf("expected %v to be a subset of %v", Elements(chunk), Elements(s))
		}
		i++
	}
}

type Foo interface {
	Foo()
}

type bar struct {
	Bar string
}

func (b *bar) Foo() {}

type foo struct {
	Baz string
}

func (f *foo) Foo() {}

func TestLocked_JSON(t *testing.T) {
	t.Parallel()
	set := NewLocked[Foo]()
	set.Add(&foo{})
	set.Add(&bar{})
	d, err := json.Marshal(set)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	t.Log("JSON:", string(d))
	// can't unmarshal into a set of interfaces
	if err := json.Unmarshal(d, &set); err == nil {
		t.Fatalf("expected error: %v", err)
	}

	set2 := NewLocked[foo]()
	set2.Add(foo{Baz: "bar"})
	set2.Add(foo{Baz: "foo"})

	d, err = json.Marshal(set2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	t.Log("JSON:", string(d))
	if err = json.Unmarshal(d, &set2); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	set3 := NewLocked[*foo]()
	set3.Add(&foo{Baz: "bar"})
	set3.Add(&foo{Baz: "foo"})
	d, err = json.Marshal(set3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	t.Log("JSON:", string(d))
	if err = json.Unmarshal(d, &set3); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	set4 := NewLocked[chan foo]()
	set4.Add(make(chan foo))
	set4.Add(make(chan foo))
	// see comparison rules for channels
	if set.Cardinality() != 2 {
		t.Fatalf("expected 2 elements, got %d", set.Cardinality())
	}
	_, err = json.Marshal(set4)
	// can't marshal a set of channels
	if err == nil {
		t.Fatalf("expected error: %v", err)
	}

	type Bar struct {
		Set *Locked[int]
	}

	b := Bar{Set: NewLocked[int]()}
	b.Set.Add(1)
	b.Set.Add(2)

	d, err = json.Marshal(b)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	t.Log("JSON:", string(d))
	var c Bar
	if err = json.Unmarshal(d, &c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !Equal(b.Set, c.Set) {
		t.Fatalf("expected %v, got %v", Elements(b.Set), Elements(c.Set))
	}
}

func TestMap_JSON(t *testing.T) {
	t.Parallel()
	set := New[Foo]()
	set.Add(&foo{})
	set.Add(&bar{})
	d, err := json.Marshal(set)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	t.Log("JSON:", string(d))
	// can't unmarshal into a set of interfaces
	if err := json.Unmarshal(d, &set); err == nil {
		t.Fatalf("expected error: %v", err)
	}

	set2 := New[foo]()
	set2.Add(foo{Baz: "bar"})
	set2.Add(foo{Baz: "foo"})

	d, err = json.Marshal(set2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	t.Log("JSON:", string(d))
	if err = json.Unmarshal(d, &set2); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	set3 := New[*foo]()
	set3.Add(&foo{Baz: "bar"})
	set3.Add(&foo{Baz: "foo"})
	d, err = json.Marshal(set3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	t.Log("JSON:", string(d))
	if err = json.Unmarshal(d, &set3); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	set4 := New[chan foo]()
	set4.Add(make(chan foo))
	set4.Add(make(chan foo))
	// see comparison rules for channels
	if set.Cardinality() != 2 {
		t.Fatalf("expected 2 elements, got %d", set.Cardinality())
	}
	_, err = json.Marshal(set4)
	// can't marshal a set of channels
	if err == nil {
		t.Fatalf("expected error: %v", err)
	}

	type Bar struct {
		Set *Map[int]
	}

	b := Bar{Set: New[int]()}
	b.Set.Add(1)
	b.Set.Add(2)

	d, err = json.Marshal(b)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	t.Log("JSON:", string(d))
	var c Bar
	if err = json.Unmarshal(d, &c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !Equal(b.Set, c.Set) {
		t.Fatalf("expected %v, got %v", Elements(b.Set), Elements(c.Set))
	}
}

func TestOrdered_JSON(t *testing.T) {
	t.Parallel()

	a := NewOrdered[int]()
	a.Add(1)
	a.Add(2)

	j, err := json.Marshal(a)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	t.Log("JSON:", string(j))
	var b *Ordered[int]
	if err = json.Unmarshal(j, &b); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !Equal(a, b) {
		t.Fatalf("expected %v, got %v", Elements(a), Elements(b))
	}

	type Bar struct {
		Set *Ordered[int]
	}

	c := Bar{Set: NewOrdered[int]()}
	c.Set.Add(1)
	c.Set.Add(2)

	j, err = json.Marshal(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	t.Log("JSON:", string(j))
	var d Bar
	if err = json.Unmarshal(j, &d); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !Equal(c.Set, d.Set) {
		t.Fatalf("expected %v, got %v", Elements(c.Set), Elements(d.Set))
	}
}

func TestLockedOrdered_JSON(t *testing.T) {
	t.Parallel()

	a := NewLockedOrdered[int]()
	a.Add(1)
	a.Add(2)

	j, err := json.Marshal(a)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	t.Log("JSON:", string(j))
	var b *LockedOrdered[int]
	if err = json.Unmarshal(j, &b); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !Equal(a, b) {
		t.Fatalf("expected %v, got %v", Elements(a), Elements(b))
	}

	type Bar struct {
		Set *LockedOrdered[int]
	}

	c := Bar{Set: NewLockedOrdered[int]()}
	c.Set.Add(1)
	c.Set.Add(2)

	j, err = json.Marshal(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	t.Log("JSON:", string(j))
	var d Bar
	if err = json.Unmarshal(j, &d); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !Equal(c.Set, d.Set) {
		t.Fatalf("expected %v, got %v", Elements(c.Set), Elements(d.Set))
	}
}

func TestSyncMap_JSON(t *testing.T) {
	t.Parallel()
	set := NewSyncMap[Foo]()
	set.Add(&foo{})
	set.Add(&bar{})
	d, err := json.Marshal(set)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	t.Log("JSON:", string(d))
	// can't unmarshal into a set of interfaces
	if err := json.Unmarshal(d, &set); err == nil {
		t.Fatalf("expected error: %v", err)
	}

	set2 := NewSyncMap[foo]()
	set2.Add(foo{Baz: "bar"})
	set2.Add(foo{Baz: "foo"})

	d, err = json.Marshal(set2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	t.Log("JSON:", string(d))
	if err = json.Unmarshal(d, &set2); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	set3 := NewSyncMap[*foo]()
	set3.Add(&foo{Baz: "bar"})
	set3.Add(&foo{Baz: "foo"})
	d, err = json.Marshal(set3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	t.Log("JSON:", string(d))
	if err = json.Unmarshal(d, &set3); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	set4 := NewSyncMap[chan foo]()
	set4.Add(make(chan foo))
	set4.Add(make(chan foo))
	// see comparison rules for channels
	if set.Cardinality() != 2 {
		t.Fatalf("expected 2 elements, got %d", set.Cardinality())
	}
	_, err = json.Marshal(set4)
	// can't marshal a set of channels
	if err == nil {
		t.Fatalf("expected error: %v", err)
	}

	type Bar struct {
		Set *SyncMap[int]
	}

	b := Bar{Set: NewSyncMap[int]()}
	b.Set.Add(1)
	b.Set.Add(2)

	d, err = json.Marshal(b)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	t.Log("JSON:", string(d))
	var c Bar
	if err = json.Unmarshal(d, &c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !Equal(b.Set, c.Set) {
		t.Fatalf("expected %v, got %v", Elements(b.Set), Elements(c.Set))
	}
}

func TestFirst_Last_Ordered(t *testing.T) {
	t.Parallel()

	s := NewOrderedWith(5, 3, 1)
	if v, ok := First(s); !ok || v != 5 {
		t.Fatalf("expected First to be 5, got %d (ok=%v)", v, ok)
	}
	if v, ok := Last(s); !ok || v != 1 {
		t.Fatalf("expected Last to be 1, got %d (ok=%v)", v, ok)
	}

	empty := NewOrdered[int]()
	if _, ok := First(empty); ok {
		t.Fatalf("expected First on empty set to return false")
	}
	if _, ok := Last(empty); ok {
		t.Fatalf("expected Last on empty set to return false")
	}
}

func TestFirst_Last_LockedOrdered(t *testing.T) {
	t.Parallel()

	s := NewLockedOrderedWith(5, 3, 1)
	if v, ok := First(s); !ok || v != 5 {
		t.Fatalf("expected First to be 5, got %d (ok=%v)", v, ok)
	}
	if v, ok := Last(s); !ok || v != 1 {
		t.Fatalf("expected Last to be 1, got %d (ok=%v)", v, ok)
	}

	empty := NewLockedOrdered[int]()
	if _, ok := First(empty); ok {
		t.Fatalf("expected First on empty set to return false")
	}
	if _, ok := Last(empty); ok {
		t.Fatalf("expected Last on empty set to return false")
	}
}
