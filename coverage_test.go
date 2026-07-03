package sets

import (
	"cmp"
	"encoding/json"
	"math"
	"slices"
	"testing"
)

// plainSet delegates to an embedded Set but does not implement json.Marshaler,
// json.Unmarshaler, or Locker, exercising the wrapper types' error branches.
type plainSet[M comparable] struct {
	Set[M]
}

// plainOrdered is the OrderedSet analog of plainSet.
type plainOrdered[M cmp.Ordered] struct {
	OrderedSet[M]
}

// lyingSet reports a non-zero cardinality while iterating nothing, exercising
// Random's fallback return for misbehaving Set implementations.
type lyingSet struct {
	Set[int]
}

func (lyingSet) Cardinality() int { return 1 }

func TestFilterTo(t *testing.T) {
	t.Parallel()

	src := NewWith(1, 2, 3, 4)
	dst := New[int]()
	FilterTo(src, dst, func(i int) bool { return i%2 == 0 })
	if !Equal(dst, NewWith(2, 4)) {
		t.Fatalf("FilterTo yielded %v, want [2 4]", Elements(dst))
	}
}

func TestString(t *testing.T) {
	t.Parallel()

	if got, want := NewLockedWith(1).String(), "LockedSet[int]([1])"; got != want {
		t.Fatalf("Locked.String() = %q, want %q", got, want)
	}
	if got, want := NewLockedOrderedWith(1).String(), "LockedOrderedSet[int]([1])"; got != want {
		t.Fatalf("LockedOrdered.String() = %q, want %q", got, want)
	}
	if got, want := NewSyncMapWith(1).String(), "SyncSet[int]([1])"; got != want {
		t.Fatalf("SyncMap.String() = %q, want %q", got, want)
	}
}

func TestLockedOrdered_OrderedAccessors(t *testing.T) {
	t.Parallel()

	s := NewLockedOrderedWith(3, 1, 2)

	var idx, vals []int
	s.Ordered(func(i, v int) bool {
		idx = append(idx, i)
		vals = append(vals, v)
		return true
	})
	if !slices.Equal(idx, []int{0, 1, 2}) || !slices.Equal(vals, []int{3, 1, 2}) {
		t.Fatalf("Ordered yielded idx=%v vals=%v", idx, vals)
	}
	vals = nil
	s.Ordered(func(_, v int) bool {
		vals = append(vals, v)
		return false
	})
	if !slices.Equal(vals, []int{3}) {
		t.Fatalf("Ordered early stop yielded %v", vals)
	}

	idx, vals = nil, nil
	s.Backwards(func(i, v int) bool {
		idx = append(idx, i)
		vals = append(vals, v)
		return true
	})
	if !slices.Equal(idx, []int{2, 1, 0}) || !slices.Equal(vals, []int{2, 1, 3}) {
		t.Fatalf("Backwards yielded idx=%v vals=%v", idx, vals)
	}
	vals = nil
	s.Backwards(func(_, v int) bool {
		vals = append(vals, v)
		return false
	})
	if !slices.Equal(vals, []int{2}) {
		t.Fatalf("Backwards early stop yielded %v", vals)
	}

	if got := s.Index(1); got != 1 {
		t.Fatalf("Index(1) = %d, want 1", got)
	}
	if got := s.Index(99); got != -1 {
		t.Fatalf("Index(99) = %d, want -1", got)
	}

	s.Sort()
	if got := slices.Collect(s.Iterator); !slices.Equal(got, []int{1, 2, 3}) {
		t.Fatalf("after Sort: Iterator yielded %v", got)
	}
}

func TestWrappingAlreadyLocked(t *testing.T) {
	t.Parallel()

	l := NewLocked[int]()
	if got := NewLockedWrapping(l); got != Set[int](l) {
		t.Fatalf("NewLockedWrapping returned %T, want the original locked set", got)
	}
	lo := NewLockedOrdered[int]()
	if got := NewLockedOrderedWrapping(lo); got != OrderedSet[int](lo) {
		t.Fatalf("NewLockedOrderedWrapping returned %T, want the original locked set", got)
	}
}

func TestNilReceiverCardinality(t *testing.T) {
	t.Parallel()

	var m *Map[int]
	var o *Ordered[int]
	var sm *SyncMap[int]
	var l *Locked[int]
	var lo *LockedOrdered[int]
	var ss *SortedSet[int]
	for i, c := range []int{
		m.Cardinality(), o.Cardinality(), sm.Cardinality(),
		l.Cardinality(), lo.Cardinality(), ss.Cardinality(),
	} {
		if c != 0 {
			t.Fatalf("nil receiver Cardinality() #%d = %d, want 0", i, c)
		}
	}
}

func TestZeroValueWrappers(t *testing.T) {
	t.Parallel()

	var m Map[int]
	if c := m.Clone(); c.Cardinality() != 0 || !c.Add(1) {
		t.Fatal("zero value Map.Clone() is not an empty usable set")
	}

	var l Locked[int]
	if c := l.Clone(); c.Cardinality() != 0 {
		t.Fatalf("zero value Locked.Clone() has %d elements", c.Cardinality())
	}
	if e := l.NewEmpty(); e.Cardinality() != 0 {
		t.Fatalf("zero value Locked.NewEmpty() has %d elements", e.Cardinality())
	}
	var l2 Locked[int]
	if err := l2.UnmarshalJSON([]byte(`[1,2]`)); err != nil {
		t.Fatalf("zero value Locked.UnmarshalJSON error: %v", err)
	}
	if l2.Cardinality() != 2 {
		t.Fatalf("zero value Locked after UnmarshalJSON has %d elements", l2.Cardinality())
	}
	var l3 Locked[int]
	if err := l3.Scan([]byte(`[1]`)); err != nil {
		t.Fatalf("zero value Locked.Scan error: %v", err)
	}

	var lo LockedOrdered[int]
	if c := lo.Clone(); c.Cardinality() != 0 {
		t.Fatalf("zero value LockedOrdered.Clone() has %d elements", c.Cardinality())
	}
	if e := lo.NewEmptyOrdered(); e.Cardinality() != 0 {
		t.Fatalf("zero value LockedOrdered.NewEmptyOrdered() has %d elements", e.Cardinality())
	}
	var lo2 LockedOrdered[int]
	if err := lo2.UnmarshalJSON([]byte(`[1,2]`)); err != nil {
		t.Fatalf("zero value LockedOrdered.UnmarshalJSON error: %v", err)
	}
	var lo3 LockedOrdered[int]
	if err := lo3.Scan([]byte(`[1]`)); err != nil {
		t.Fatalf("zero value LockedOrdered.Scan error: %v", err)
	}
}

func TestEqualSubsetSameCardinality(t *testing.T) {
	t.Parallel()

	if Equal(NewWith(1, 2), NewWith(1, 3)) {
		t.Fatal("Equal on same-cardinality different sets returned true")
	}
	if Subset(NewWith(1, 5), NewWith(1, 2, 3)) {
		t.Fatal("Subset returned true for a non-subset of larger cardinality")
	}
}

func TestIter2EarlyStop(t *testing.T) {
	t.Parallel()

	var seen []int
	for i := range Iter2(NewOrderedWith(1, 2, 3).Iterator) {
		seen = append(seen, i)
		break
	}
	if !slices.Equal(seen, []int{0}) {
		t.Fatalf("Iter2 early stop yielded %v", seen)
	}
}

func TestMinMaxPanicOnEmpty(t *testing.T) {
	t.Parallel()

	for name, fn := range map[string]func(){
		"Max": func() { Max(New[int]()) },
		"Min": func() { Min(New[int]()) },
	} {
		func() {
			defer func() {
				if recover() == nil {
					t.Fatalf("%s on empty set did not panic", name)
				}
			}()
			fn()
		}()
	}
}

func TestChunkPanicAndEarlyStop(t *testing.T) {
	t.Parallel()

	func() {
		defer func() {
			if recover() == nil {
				t.Fatal("Chunk with n=0 did not panic")
			}
		}()
		Chunk(New[int](), 0)
	}()

	var chunks int
	for range Chunk(NewOrderedWith(1, 2, 3, 4, 5), 2) {
		chunks++
		break
	}
	if chunks != 1 {
		t.Fatalf("Chunk early stop yielded %d chunks", chunks)
	}
}

func TestOrderedBackwardsEarlyStop(t *testing.T) {
	t.Parallel()

	var vals []int
	NewOrderedWith(1, 2, 3).Backwards(func(_, v int) bool {
		vals = append(vals, v)
		return false
	})
	if !slices.Equal(vals, []int{3}) {
		t.Fatalf("Backwards early stop yielded %v", vals)
	}
}

func TestMarshalJSONErrors(t *testing.T) {
	t.Parallel()

	if _, err := NewOrderedWith(math.NaN()).MarshalJSON(); err == nil {
		t.Fatal("Ordered.MarshalJSON(NaN) did not error")
	}
	if _, err := NewLockedWith(math.NaN()).MarshalJSON(); err == nil {
		t.Fatal("Locked.MarshalJSON(NaN) did not error")
	}
	if _, err := NewLockedOrderedWith(math.NaN()).MarshalJSON(); err == nil {
		t.Fatal("LockedOrdered.MarshalJSON(NaN) did not error")
	}
}

func TestLockedWrappingNonJSONSet(t *testing.T) {
	t.Parallel()

	w := NewLockedWrapping(plainSet[int]{New[int]()})
	if _, err := w.(json.Marshaler).MarshalJSON(); err == nil {
		t.Fatal("MarshalJSON of a wrapped non-marshaler set did not error")
	}
	if err := w.(json.Unmarshaler).UnmarshalJSON([]byte(`[1]`)); err == nil {
		t.Fatal("UnmarshalJSON of a wrapped non-unmarshaler set did not error")
	}
	if err := w.(interface{ Scan(any) error }).Scan([]byte(`[1]`)); err == nil {
		t.Fatal("Scan of a wrapped non-unmarshaler set did not error")
	}

	wo := NewLockedOrderedWrapping(plainOrdered[int]{NewOrdered[int]()})
	if _, err := wo.(json.Marshaler).MarshalJSON(); err == nil {
		t.Fatal("MarshalJSON of a wrapped non-marshaler ordered set did not error")
	}
	if err := wo.(json.Unmarshaler).UnmarshalJSON([]byte(`[1]`)); err == nil {
		t.Fatal("UnmarshalJSON of a wrapped non-unmarshaler ordered set did not error")
	}
	if err := wo.(interface{ Scan(any) error }).Scan([]byte(`[1]`)); err == nil {
		t.Fatal("Scan of a wrapped non-unmarshaler ordered set did not error")
	}
}

func TestRandomMisbehavingSet(t *testing.T) {
	t.Parallel()

	if _, ok := Random(lyingSet{New[int]()}); ok {
		t.Fatal("Random on a set that iterates nothing returned ok=true")
	}
}
