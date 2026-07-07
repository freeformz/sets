package sets

import (
	"encoding/json"
	"math"
	"slices"
	"testing"

	"pgregory.net/rapid"
)

func TestSortedSet(t *testing.T) {
	t.Parallel()

	setStateMachine := &SetStateMachine{
		set:    NewSortedSet[int](),
		stateI: make(map[int]int),
	}
	rapid.Check(t, func(t *rapid.T) {
		t.Repeat(rapid.StateMachineActions(setStateMachine))
	})
}

// TestSortedSet_Invariant verifies that the sorted invariant (strictly ascending, no duplicates)
// holds after every mutation, checked against a map-based model.
func TestSortedSet_Invariant(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(t *rapid.T) {
		s := NewSortedSet[int]()
		model := make(map[int]struct{})

		steps := rapid.IntRange(1, 200).Draw(t, "Steps")
		for range steps {
			switch rapid.IntRange(0, 3).Draw(t, "Op") {
			case 0:
				v := rapid.IntRange(-50, 50).Draw(t, "Add")
				_, exists := model[v]
				if s.Add(v) == exists {
					t.Fatalf("Add(%d): expected added=%v", v, !exists)
				}
				model[v] = struct{}{}
			case 1:
				v := rapid.IntRange(-50, 50).Draw(t, "Remove")
				_, exists := model[v]
				if s.Remove(v) != exists {
					t.Fatalf("Remove(%d): expected removed=%v", v, exists)
				}
				delete(model, v)
			case 2:
				v, ok := s.Pop()
				if ok != (len(model) > 0) {
					t.Fatalf("Pop(): expected ok=%v", len(model) > 0)
				}
				if ok {
					if _, exists := model[v]; !exists {
						t.Fatalf("Pop(): returned %d, not in the model", v)
					}
					delete(model, v)
				}
			case 3:
				s.Sort() // no-op, must not disturb anything
			}

			if s.Cardinality() != len(model) {
				t.Fatalf("Cardinality() = %d, want %d", s.Cardinality(), len(model))
			}
			if !IsSorted(s) {
				t.Fatalf("set is not sorted: %v", Elements(s))
			}
			var prev int
			for i, v := range s.Ordered {
				if i > 0 && v <= prev {
					t.Fatalf("elements not strictly ascending at index %d: %v", i, Elements(s))
				}
				prev = v
				if _, exists := model[v]; !exists {
					t.Fatalf("set contains %d, model does not", v)
				}
			}
		}
	})
}

func TestSortedSet_Order(t *testing.T) {
	t.Parallel()

	s := NewSortedSetWith(5, 3, 9, 1, 7, 3) // includes a duplicate
	want := []int{1, 3, 5, 7, 9}

	if got := slices.Collect(s.Iterator); !slices.Equal(got, want) {
		t.Fatalf("Iterator yielded %v, want %v", got, want)
	}

	// adding out of order keeps things sorted
	s.Add(4)
	s.Add(0)
	want = []int{0, 1, 3, 4, 5, 7, 9}
	if got := slices.Collect(s.Iterator); !slices.Equal(got, want) {
		t.Fatalf("after Add: Iterator yielded %v, want %v", got, want)
	}

	for i, v := range want {
		got, ok := s.At(i)
		if !ok || got != v {
			t.Fatalf("At(%d) = %d, %v, want %d, true", i, got, ok, v)
		}
		if idx := s.Index(v); idx != i {
			t.Fatalf("Index(%d) = %d, want %d", v, idx, i)
		}
	}

	if _, ok := s.At(-1); ok {
		t.Fatal("At(-1): expected ok=false")
	}
	if _, ok := s.At(len(want)); ok {
		t.Fatalf("At(%d): expected ok=false", len(want))
	}
	if idx := s.Index(2); idx != -1 {
		t.Fatalf("Index(2) = %d, want -1", idx)
	}

	if v, ok := First(s); !ok || v != 0 {
		t.Fatalf("First = %d, %v, want 0, true", v, ok)
	}
	if v, ok := Last(s); !ok || v != 9 {
		t.Fatalf("Last = %d, %v, want 9, true", v, ok)
	}

	// Backwards yields descending values with their ascending indexes
	var gotIdx, gotVals []int
	s.Backwards(func(i, v int) bool {
		gotIdx = append(gotIdx, i)
		gotVals = append(gotVals, v)
		return true
	})
	if !slices.Equal(gotIdx, []int{6, 5, 4, 3, 2, 1, 0}) {
		t.Fatalf("Backwards indexes = %v", gotIdx)
	}
	if !slices.Equal(gotVals, []int{9, 7, 5, 4, 3, 1, 0}) {
		t.Fatalf("Backwards values = %v", gotVals)
	}
}

func TestSortedSet_IterationEarlyStop(t *testing.T) {
	t.Parallel()

	s := NewSortedSetWith(1, 2, 3, 4, 5)

	var got []int
	s.Iterator(func(v int) bool {
		got = append(got, v)
		return len(got) < 2
	})
	if !slices.Equal(got, []int{1, 2}) {
		t.Fatalf("Iterator early stop yielded %v", got)
	}

	got = nil
	s.Ordered(func(_, v int) bool {
		got = append(got, v)
		return false
	})
	if !slices.Equal(got, []int{1}) {
		t.Fatalf("Ordered early stop yielded %v", got)
	}

	got = nil
	s.Backwards(func(_, v int) bool {
		got = append(got, v)
		return false
	})
	if !slices.Equal(got, []int{5}) {
		t.Fatalf("Backwards early stop yielded %v", got)
	}

	got = nil
	for v := range s.Range(2, 5) {
		got = append(got, v)
		if len(got) == 2 {
			break
		}
	}
	if !slices.Equal(got, []int{2, 3}) {
		t.Fatalf("Range early stop yielded %v", got)
	}
}

func TestSortedSet_Range(t *testing.T) {
	t.Parallel()

	s := NewSortedSetWith(10, 20, 30, 40, 50)

	cases := []struct {
		name   string
		lo, hi int
		want   []int
	}{
		{"inclusive both ends", 20, 40, []int{20, 30, 40}},
		{"bounds between elements", 15, 45, []int{20, 30, 40}},
		{"covers everything", -100, 100, []int{10, 20, 30, 40, 50}},
		{"single element", 30, 30, []int{30}},
		{"below all", -10, 5, nil},
		{"above all", 60, 100, nil},
		{"lo > hi", 40, 20, nil},
	}
	for _, tc := range cases {
		if got := slices.Collect(s.Range(tc.lo, tc.hi)); !slices.Equal(got, tc.want) {
			t.Fatalf("%s: Range(%d, %d) yielded %v, want %v", tc.name, tc.lo, tc.hi, got, tc.want)
		}
	}

	empty := NewSortedSet[int]()
	if got := slices.Collect(empty.Range(0, 100)); got != nil {
		t.Fatalf("Range on empty set yielded %v", got)
	}
}

func TestSortedSet_Pop(t *testing.T) {
	t.Parallel()

	s := NewSortedSetWith(3, 1, 2)
	remaining := map[int]bool{1: true, 2: true, 3: true}
	for range 3 {
		v, ok := s.Pop()
		if !ok {
			t.Fatalf("Pop() on non-empty set returned ok=false")
		}
		if !remaining[v] {
			t.Fatalf("Pop() = %d, which was already popped or never present", v)
		}
		delete(remaining, v)
		if s.Contains(v) {
			t.Fatalf("Pop() returned %d but it is still in the set", v)
		}
		if !IsSorted(s) {
			t.Fatalf("set not sorted after Pop: %v", Elements(s))
		}
	}
	if v, ok := s.Pop(); ok {
		t.Fatalf("Pop() on empty set = %d, true, want ok=false", v)
	}
}

func TestSortedSet_ZeroValue(t *testing.T) {
	t.Parallel()

	var s SortedSet[int]
	if s.Cardinality() != 0 {
		t.Fatalf("zero value Cardinality() = %d", s.Cardinality())
	}
	if s.Contains(1) {
		t.Fatal("zero value Contains(1) = true")
	}
	if n := s.Clear(); n != 0 {
		t.Fatalf("zero value Clear() = %d", n)
	}
	if !s.Add(2) || !s.Add(1) {
		t.Fatal("zero value Add failed")
	}
	if got := slices.Collect(s.Iterator); !slices.Equal(got, []int{1, 2}) {
		t.Fatalf("zero value Iterator yielded %v", got)
	}
	if !s.Remove(1) {
		t.Fatal("zero value Remove(1) failed")
	}

	var nilSafe *SortedSet[int]
	if nilSafe.Cardinality() != 0 {
		t.Fatal("nil receiver Cardinality() != 0")
	}
}

func TestSortedSet_CloneAndNewEmpty(t *testing.T) {
	t.Parallel()

	s := NewSortedSetWith(1, 2, 3)

	c := s.Clone()
	if _, ok := c.(*SortedSet[int]); !ok {
		t.Fatalf("Clone() returned %T, want *SortedSet[int]", c)
	}
	if !Equal(s, c) {
		t.Fatalf("Clone() = %v, want %v", Elements(c), Elements(s))
	}
	// clone is independent of the original
	c.Add(4)
	if s.Contains(4) {
		t.Fatal("mutating the clone changed the original")
	}
	s.Remove(1)
	if !c.Contains(1) {
		t.Fatal("mutating the original changed the clone")
	}

	if _, ok := s.NewEmpty().(*SortedSet[int]); !ok {
		t.Fatalf("NewEmpty() returned %T, want *SortedSet[int]", s.NewEmpty())
	}
	if _, ok := s.NewEmptyOrdered().(*SortedSet[int]); !ok {
		t.Fatalf("NewEmptyOrdered() returned %T, want *SortedSet[int]", s.NewEmptyOrdered())
	}
}

func TestSortedSet_EqualOrdered(t *testing.T) {
	t.Parallel()

	s := NewSortedSetWith(3, 1, 2)

	o := NewOrderedWith(3, 1, 2)
	o.Sort()
	if !EqualOrdered(s, o) {
		t.Fatalf("expected %v to equal sorted Ordered %v", Elements(s), Elements(o))
	}

	unsorted := NewOrderedWith(3, 1, 2)
	if EqualOrdered(s, unsorted) {
		t.Fatalf("expected %v to differ from insertion-ordered %v", Elements(s), Elements(unsorted))
	}
	if !Equal(s, unsorted) {
		t.Fatal("expected sets to be Equal regardless of order")
	}
}

func TestSortedSet_String(t *testing.T) {
	t.Parallel()

	s := NewSortedSetWith(2, 1)
	if got, want := s.String(), "SortedSet[int]([1 2])"; got != want {
		t.Fatalf("String() = %q, want %q", got, want)
	}
}

func TestSortedSet_JSON(t *testing.T) {
	t.Parallel()

	a := NewSortedSet[int]()
	a.Add(2)
	a.Add(1)

	j, err := json.Marshal(a)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// marshals in ascending order
	if string(j) != "[1,2]" {
		t.Fatalf("MarshalJSON = %s, want [1,2]", j)
	}
	var b *SortedSet[int]
	if err = json.Unmarshal(j, &b); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !Equal(a, b) {
		t.Fatalf("expected %v, got %v", Elements(a), Elements(b))
	}

	// unsorted input with duplicates is sorted and de-duplicated
	var c SortedSet[int]
	if err := json.Unmarshal([]byte("[3,1,2,3,1]"), &c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := slices.Collect(c.Iterator); !slices.Equal(got, []int{1, 2, 3}) {
		t.Fatalf("unmarshal of unsorted input yielded %v", got)
	}

	// invalid JSON errors
	if err := json.Unmarshal([]byte("{"), &c); err == nil {
		t.Fatal("expected error unmarshaling invalid JSON")
	}
	// element type mismatch errors and leaves the set unchanged
	if err := c.UnmarshalJSON([]byte(`["a","b"]`)); err == nil {
		t.Fatal("expected error unmarshaling mismatched element type")
	}
	if got := slices.Collect(c.Iterator); !slices.Equal(got, []int{1, 2, 3}) {
		t.Fatalf("set changed after failed unmarshal: %v", got)
	}

	// empty set marshals to an empty array
	j, err = json.Marshal(NewSortedSet[int]())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(j) != "[]" {
		t.Fatalf("empty MarshalJSON = %s, want []", j)
	}

	// values JSON can't represent error
	if _, err := NewSortedSetWith(math.NaN()).MarshalJSON(); err == nil {
		t.Fatal("expected error marshaling NaN")
	}

	type Bar struct {
		Set *SortedSet[int]
	}

	d := Bar{Set: NewSortedSetWith(2, 1)}
	j, err = json.Marshal(d)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var e Bar
	if err = json.Unmarshal(j, &e); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !Equal(d.Set, e.Set) {
		t.Fatalf("expected %v, got %v", Elements(d.Set), Elements(e.Set))
	}
}

func TestSortedSet_ValueScan(t *testing.T) {
	t.Parallel()

	s := NewSortedSetWith(3, 1, 2)
	v, err := s.Value()
	if err != nil {
		t.Fatalf("Value() error: %v", err)
	}
	b, ok := v.([]byte)
	if !ok {
		t.Fatalf("Value() returned %T, want []byte", v)
	}

	got := NewSortedSet[int]()
	if err := got.Scan(b); err != nil {
		t.Fatalf("Scan([]byte) error: %v", err)
	}
	if !Equal(s, got) {
		t.Fatalf("Scan([]byte) = %v, want %v", Elements(got), Elements(s))
	}

	got = NewSortedSet[int]()
	if err := got.Scan(string(b)); err != nil {
		t.Fatalf("Scan(string) error: %v", err)
	}
	if !Equal(s, got) {
		t.Fatalf("Scan(string) = %v, want %v", Elements(got), Elements(s))
	}

	// nil clears the set
	if err := got.Scan(nil); err != nil {
		t.Fatalf("Scan(nil) error: %v", err)
	}
	if got.Cardinality() != 0 {
		t.Fatalf("Scan(nil) left %d elements", got.Cardinality())
	}

	if err := got.Scan(42); err == nil {
		t.Fatal("expected error scanning unsupported type")
	}
}

func TestSortedSet_Clear(t *testing.T) {
	t.Parallel()

	s := NewSortedSetWith(1, 2, 3)
	if n := s.Clear(); n != 3 {
		t.Fatalf("Clear() = %d, want 3", n)
	}
	if s.Cardinality() != 0 {
		t.Fatalf("Cardinality() after Clear = %d", s.Cardinality())
	}
	// reusable after Clear
	s.Add(9)
	s.Add(4)
	if got := slices.Collect(s.Iterator); !slices.Equal(got, []int{4, 9}) {
		t.Fatalf("Iterator after Clear yielded %v", got)
	}
}

func TestSortedSet_From(t *testing.T) {
	t.Parallel()

	s := NewSortedSetFrom(slices.Values([]int{5, 5, 2, 8, 2}))
	if got := slices.Collect(s.Iterator); !slices.Equal(got, []int{2, 5, 8}) {
		t.Fatalf("NewSortedSetFrom yielded %v", got)
	}

	empty := NewSortedSetFrom(slices.Values([]int(nil)))
	if empty.Cardinality() != 0 {
		t.Fatalf("NewSortedSetFrom(empty) has %d elements", empty.Cardinality())
	}
}

// TestSortedSet_MergeOps differentially tests the merge-based set operations against the generic
// Map-based results, for both pure-SortedSet operands (fast path) and mixed operands (generic
// fallback).
func TestSortedSet_MergeOps(t *testing.T) {
	t.Parallel()

	ops := []struct {
		name string
		f    func(a, b Set[int]) Set[int]
	}{
		{"Union", Union[int]},
		{"Intersection", Intersection[int]},
		{"Difference", Difference[int]},
		{"SymmetricDifference", SymmetricDifference[int]},
	}

	rapid.Check(t, func(t *rapid.T) {
		av := rapid.SliceOfN(rapid.IntRange(-512, 512), 0, 100).Draw(t, "A")
		bv := rapid.SliceOfN(rapid.IntRange(-512, 512), 0, 100).Draw(t, "B")
		as, bs := NewSortedSetWith(av...), NewSortedSetWith(bv...)
		am, bm := NewWith(av...), NewWith(bv...)

		for _, op := range ops {
			want := op.f(am, bm)

			got := op.f(as, bs)
			ss, ok := got.(*SortedSet[int])
			if !ok {
				t.Fatalf("%s(sorted, sorted) returned %T, want *SortedSet[int]", op.name, got)
			}
			if !slices.IsSorted(ss.el) {
				t.Fatalf("%s(sorted, sorted) result is not sorted: %v", op.name, ss.el)
			}
			if !Equal(got, want) {
				t.Fatalf("%s(sorted, sorted) = %v, want %v", op.name, Elements(got), Elements(want))
			}

			// mixed operands fall back to the generic path
			if got := op.f(as, bm); !Equal(got, want) {
				t.Fatalf("%s(sorted, map) = %v, want %v", op.name, Elements(got), Elements(want))
			}
			if got := op.f(am, bs); !Equal(got, want) {
				t.Fatalf("%s(map, sorted) = %v, want %v", op.name, Elements(got), Elements(want))
			}
		}
	})

	// SortedSet's optimization methods report false for non-SortedSet operands
	if _, ok := NewSortedSetWith(1).Union(NewWith(2)); ok {
		t.Fatal("SortedSet.Union(non-SortedSet) must report false")
	}
}

func TestSortedSet_MaxMin(t *testing.T) {
	t.Parallel()

	s := NewSortedSetWith(5, -3, 12, 7)
	if v, ok := s.Max(); !ok || v != 12 {
		t.Fatalf("Max() = %d, %v; want 12, true", v, ok)
	}
	if v, ok := s.Min(); !ok || v != -3 {
		t.Fatalf("Min() = %d, %v; want -3, true", v, ok)
	}

	// the package-level functions use the fast paths
	if got := Max(Set[int](s)); got != 12 {
		t.Fatalf("Max = %d, want 12", got)
	}
	if got := Min(Set[int](s)); got != -3 {
		t.Fatalf("Min = %d, want -3", got)
	}

	empty := NewSortedSet[int]()
	if _, ok := empty.Max(); ok {
		t.Fatal("Max on an empty set: expected ok=false")
	}
	if _, ok := empty.Min(); ok {
		t.Fatal("Min on an empty set: expected ok=false")
	}
}
