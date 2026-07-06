package sets

import (
	"encoding/json"
	"math"
	"slices"
	"testing"

	"pgregory.net/rapid"
)

func TestBitSet(t *testing.T) {
	t.Parallel()

	setStateMachine := &SetStateMachine{
		set:    NewBitSet[int](),
		stateI: make(map[int]int),
		// BitSet memory is proportional to the element span, so the state
		// machine draws from a bounded universe instead of the full int range.
		intGen: rapid.IntRange(-1024, 1024),
	}
	rapid.Check(t, func(t *rapid.T) {
		t.Repeat(rapid.StateMachineActions(setStateMachine))
	})
}

// TestBitSet_Invariant verifies the sorted invariant (strictly ascending, no
// duplicates), cardinality bookkeeping, and At/Index consistency after every
// mutation, checked against a map-based model.
func TestBitSet_Invariant(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(t *rapid.T) {
		s := NewBitSet[int]()
		model := make(map[int]struct{})

		steps := rapid.IntRange(1, 200).Draw(t, "Steps")
		for range steps {
			switch rapid.IntRange(0, 4).Draw(t, "Op") {
			case 0:
				v := rapid.IntRange(-200, 200).Draw(t, "Add")
				_, exists := model[v]
				if s.Add(v) == exists {
					t.Fatalf("Add(%d): expected added=%v", v, !exists)
				}
				model[v] = struct{}{}
			case 1:
				v := rapid.IntRange(-200, 200).Draw(t, "Remove")
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
			case 4:
				s.Compact() // memory management, must not disturb contents
			}

			if s.Cardinality() != len(model) {
				t.Fatalf("Cardinality() = %d, want %d", s.Cardinality(), len(model))
			}
			if !IsSorted(s) {
				t.Fatalf("set is not sorted: %v", Elements(s))
			}
			prev := 0
			for i, v := range s.Ordered {
				if i > 0 && v <= prev {
					t.Fatalf("elements not strictly ascending at index %d: %v", i, Elements(s))
				}
				prev = v
				if _, exists := model[v]; !exists {
					t.Fatalf("set contains %d, model does not", v)
				}
				if got, ok := s.At(i); !ok || got != v {
					t.Fatalf("At(%d) = %d, %v; want %d, true", i, got, ok, v)
				}
				if got := s.Index(v); got != i {
					t.Fatalf("Index(%d) = %d, want %d", v, got, i)
				}
			}
			// Backwards must yield the exact reverse of Ordered
			var fwd, bwd []int
			s.Ordered(func(_ int, v int) bool { fwd = append(fwd, v); return true })
			s.Backwards(func(_ int, v int) bool { bwd = append(bwd, v); return true })
			slices.Reverse(bwd)
			if !slices.Equal(fwd, bwd) {
				t.Fatalf("Backwards is not the reverse of Ordered: %v vs %v", fwd, bwd)
			}
		}
	})
}

// TestBitSet_UniverseEdges exercises the signed/unsigned universe mapping at the
// extreme values of several element types.
func TestBitSet_UniverseEdges(t *testing.T) {
	t.Parallel()

	t.Run("int mixed signs sort correctly", func(t *testing.T) {
		t.Parallel()
		s := NewBitSetWith(-5, 3, -1, 0)
		want := []int{-5, -1, 0, 3}
		if got := Elements(s); !slices.Equal(got, want) {
			t.Fatalf("Elements = %v, want %v", got, want)
		}
		if i := s.Index(-5); i != 0 {
			t.Fatalf("Index(-5) = %d, want 0", i)
		}
		if v, ok := s.At(3); !ok || v != 3 {
			t.Fatalf("At(3) = %d, %v; want 3, true", v, ok)
		}
	})

	t.Run("int8 full range", func(t *testing.T) {
		t.Parallel()
		s := NewBitSet[int8]()
		for i := math.MinInt8; i <= math.MaxInt8; i++ {
			if !s.Add(int8(i)) {
				t.Fatalf("Add(%d) = false", i)
			}
		}
		if s.Cardinality() != 256 {
			t.Fatalf("Cardinality = %d, want 256", s.Cardinality())
		}
		if v, ok := s.At(0); !ok || v != math.MinInt8 {
			t.Fatalf("At(0) = %d, %v; want %d, true", v, ok, math.MinInt8)
		}
		if v, ok := s.At(255); !ok || v != math.MaxInt8 {
			t.Fatalf("At(255) = %d, %v; want %d, true", v, ok, math.MaxInt8)
		}
	})

	t.Run("uint8 boundaries", func(t *testing.T) {
		t.Parallel()
		s := NewBitSetWith[uint8](255, 0, 128, 127)
		want := []uint8{0, 127, 128, 255}
		if got := Elements(s); !slices.Equal(got, want) {
			t.Fatalf("Elements = %v, want %v", got, want)
		}
	})

	t.Run("uint64 above MaxInt64", func(t *testing.T) {
		t.Parallel()
		s := NewBitSetWith[uint64](math.MaxUint64, math.MaxUint64-3, math.MaxUint64-200)
		want := []uint64{math.MaxUint64 - 200, math.MaxUint64 - 3, math.MaxUint64}
		if got := Elements(s); !slices.Equal(got, want) {
			t.Fatalf("Elements = %v, want %v", got, want)
		}
		for _, v := range want {
			if !s.Contains(v) {
				t.Fatalf("Contains(%d) = false", v)
			}
			if !s.Remove(v) {
				t.Fatalf("Remove(%d) = false", v)
			}
		}
		if s.Cardinality() != 0 {
			t.Fatalf("Cardinality = %d, want 0", s.Cardinality())
		}
	})

	t.Run("int64 near MinInt64", func(t *testing.T) {
		t.Parallel()
		s := NewBitSetWith[int64](math.MinInt64+7, math.MinInt64)
		want := []int64{math.MinInt64, math.MinInt64 + 7}
		if got := Elements(s); !slices.Equal(got, want) {
			t.Fatalf("Elements = %v, want %v", got, want)
		}
	})

	t.Run("named integer type", func(t *testing.T) {
		t.Parallel()
		type port uint16
		s := NewBitSetWith[port](443, 80, 8080)
		if got := Elements(Set[port](s)); !slices.Equal(got, []port{80, 443, 8080}) {
			t.Fatalf("Elements = %v", got)
		}
	})
}

// TestBitSet_WordwiseOps differentially tests the word-wise set operations
// against the generic Map-based results, for both pure-BitSet operands (fast
// path) and mixed operands (generic fallback).
func TestBitSet_WordwiseOps(t *testing.T) {
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
		ab, bb := NewBitSetWith(av...), NewBitSetWith(bv...)
		am, bm := NewWith(av...), NewWith(bv...)

		for _, op := range ops {
			want := op.f(am, bm)

			got := op.f(ab, bb)
			if _, ok := got.(*BitSet[int]); !ok {
				t.Fatalf("%s(bitset, bitset) returned %T, want *BitSet[int]", op.name, got)
			}
			if !Equal(got, want) {
				t.Fatalf("%s(bitset, bitset) = %v, want %v", op.name, Elements(got), Elements(want))
			}

			// mixed operands fall back to the generic path
			if got := op.f(ab, bm); !Equal(got, want) {
				t.Fatalf("%s(bitset, map) = %v, want %v", op.name, Elements(got), Elements(want))
			}
			if got := op.f(am, bb); !Equal(got, want) {
				t.Fatalf("%s(map, bitset) = %v, want %v", op.name, Elements(got), Elements(want))
			}
		}
	})
}

func TestBitSet_ReserveCompact(t *testing.T) {
	t.Parallel()

	var s BitSet[int] // zero value is ready to use
	s.Reserve(0, 1023)
	if got := len(s.words); got != 16 {
		t.Fatalf("after Reserve(0, 1023): len(words) = %d, want 16", got)
	}
	if s.Cardinality() != 0 {
		t.Fatalf("Reserve must not add elements, Cardinality = %d", s.Cardinality())
	}

	s.Add(0)
	s.Add(500)
	s.Add(1023)
	if got := len(s.words); got != 16 {
		t.Fatalf("adds within the reserved span must not grow: len(words) = %d, want 16", got)
	}

	// Removing the extremes does not release memory until Compact
	s.Remove(0)
	s.Remove(1023)
	if got := len(s.words); got != 16 {
		t.Fatalf("Remove must not shrink: len(words) = %d, want 16", got)
	}
	s.Compact()
	if got := len(s.words); got != 1 {
		t.Fatalf("after Compact: len(words) = %d, want 1", got)
	}
	if !s.Contains(500) || s.Cardinality() != 1 {
		t.Fatalf("Compact must preserve contents: %v", Elements(&s))
	}

	// growing again after a Compact-induced trim must not resurrect stale bits
	s.Add(1023)
	if got := Elements(&s); !slices.Equal(got, []int{500, 1023}) {
		t.Fatalf("Elements = %v, want [500 1023]", got)
	}

	s.Remove(500)
	s.Remove(1023)
	s.Compact()
	if s.words != nil {
		t.Fatalf("Compact on an empty set must release the backing array")
	}

	s.Reserve(10, 5) // lo > hi is a no-op
	if s.words != nil {
		t.Fatalf("Reserve(10, 5) must be a no-op")
	}
}

func TestBitSet_ClearReleasesMemory(t *testing.T) {
	t.Parallel()

	s := NewBitSetWith(0, 100_000)
	if len(s.words) == 0 {
		t.Fatal("expected a backing array")
	}
	if n := s.Clear(); n != 2 {
		t.Fatalf("Clear() = %d, want 2", n)
	}
	if s.words != nil {
		t.Fatal("Clear must release the backing array")
	}
	if !s.Add(42) || !s.Contains(42) {
		t.Fatal("set must be usable after Clear")
	}
}

func TestBitSet_Range(t *testing.T) {
	t.Parallel()

	s := NewBitSetWith(50, 10, 40, 20, 30, -10)

	collect := func(lo, hi int) []int {
		return slices.Collect(s.Range(lo, hi))
	}

	if got := collect(15, 40); !slices.Equal(got, []int{20, 30, 40}) {
		t.Fatalf("Range(15, 40) = %v", got)
	}
	if got := collect(-100, 100); !slices.Equal(got, []int{-10, 10, 20, 30, 40, 50}) {
		t.Fatalf("Range(-100, 100) = %v", got)
	}
	if got := collect(40, 10); got != nil {
		t.Fatalf("Range(40, 10) = %v, want nothing", got)
	}
	if got := collect(51, 1000); got != nil {
		t.Fatalf("Range(51, 1000) = %v, want nothing", got)
	}
	if got := collect(-100, -11); got != nil {
		t.Fatalf("Range(-100, -11) = %v, want nothing", got)
	}
	if got := collect(30, 30); !slices.Equal(got, []int{30}) {
		t.Fatalf("Range(30, 30) = %v", got)
	}

	// early stop
	var first []int
	for v := range s.Range(0, 100) {
		first = append(first, v)
		break
	}
	if !slices.Equal(first, []int{10}) {
		t.Fatalf("early-stopped Range = %v", first)
	}

	if got := slices.Collect(NewBitSet[int]().Range(0, 10)); got != nil {
		t.Fatalf("Range on empty set = %v, want nothing", got)
	}
}

func TestBitSet_AtIndexOutOfBounds(t *testing.T) {
	t.Parallel()

	s := NewBitSetWith(1, 2, 3)
	if _, ok := s.At(-1); ok {
		t.Fatal("At(-1): expected ok=false")
	}
	if _, ok := s.At(3); ok {
		t.Fatal("At(3): expected ok=false")
	}
	if got := s.Index(4); got != -1 {
		t.Fatalf("Index(4) = %d, want -1", got)
	}
	if got := s.Index(0); got != -1 {
		t.Fatalf("Index(0) = %d, want -1", got)
	}
}

func TestBitSet_JSON(t *testing.T) {
	t.Parallel()

	a := NewBitSetWith(-3, 1, 2)
	d, err := json.Marshal(a)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(d) != "[-3,1,2]" {
		t.Fatalf("JSON = %s, want [-3,1,2]", d)
	}

	var b *BitSet[int]
	if err = json.Unmarshal(d, &b); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !Equal(a, b) {
		t.Fatalf("expected %v, got %v", Elements(a), Elements(b))
	}

	type Bar struct {
		Set *BitSet[int]
	}
	c := Bar{Set: NewBitSetWith(1, 2)}
	d, err = json.Marshal(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var e Bar
	if err = json.Unmarshal(d, &e); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !Equal(c.Set, e.Set) {
		t.Fatalf("expected %v, got %v", Elements(c.Set), Elements(e.Set))
	}

	if d, err := json.Marshal(NewBitSet[int]()); err != nil || string(d) != "[]" {
		t.Fatalf("empty set JSON = %s, %v; want []", d, err)
	}

	// invalid JSON leaves the set unchanged
	f := NewBitSetWith(7)
	if err := json.Unmarshal([]byte(`"nope"`), f); err == nil {
		t.Fatal("expected an error")
	}
	if !f.Contains(7) || f.Cardinality() != 1 {
		t.Fatalf("set changed by failed unmarshal: %v", Elements(f))
	}
}

func TestBitSet_Scan(t *testing.T) {
	t.Parallel()

	s := NewBitSetWith(9)
	if err := s.Scan([]byte("[1,2,3]")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := Elements(s); !slices.Equal(got, []int{1, 2, 3}) {
		t.Fatalf("Elements = %v", got)
	}
	if err := s.Scan("[4]"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := Elements(s); !slices.Equal(got, []int{4}) {
		t.Fatalf("Elements = %v", got)
	}
	if err := s.Scan(nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.Cardinality() != 0 {
		t.Fatalf("Cardinality = %d, want 0", s.Cardinality())
	}
	if err := s.Scan(42); err == nil {
		t.Fatal("expected an error for an unsupported source type")
	}
}

func TestBitSet_String(t *testing.T) {
	t.Parallel()

	if got := NewBitSetWith(2, 1).String(); got != "BitSet[int]([1 2])" {
		t.Fatalf("String() = %q", got)
	}
	if got := NewBitSet[uint8]().String(); got != "BitSet[uint8]([])" {
		t.Fatalf("String() = %q", got)
	}
}

func TestBitSet_NilReceiverCardinality(t *testing.T) {
	t.Parallel()

	var s *BitSet[int]
	if got := s.Cardinality(); got != 0 {
		t.Fatalf("nil receiver Cardinality = %d, want 0", got)
	}
}

// TestBitSet_HugeSpanPanics documents the memory tradeoff: adding two elements
// separated by an unallocatable span must panic rather than silently misbehave.
func TestBitSet_HugeSpanPanics(t *testing.T) {
	t.Parallel()

	defer func() {
		if recover() == nil {
			t.Fatal("expected a panic for an unallocatable span")
		}
	}()
	s := NewBitSetWith[uint64](0)
	s.Add(math.MaxUint64)
}

// algebraStub is a minimal third-party-style Set whose Algebra methods return a sentinel,
// proving the package-level functions honor Algebra implementations outside this package's
// concrete types. The embedded Set provides the rest of the interface.
type algebraStub struct {
	Set[int]
	sentinel Set[int]
	optimize bool
}

func (a *algebraStub) Union(Set[int]) (Set[int], bool)               { return a.sentinel, a.optimize }
func (a *algebraStub) Intersection(Set[int]) (Set[int], bool)        { return a.sentinel, a.optimize }
func (a *algebraStub) Difference(Set[int]) (Set[int], bool)          { return a.sentinel, a.optimize }
func (a *algebraStub) SymmetricDifference(Set[int]) (Set[int], bool) { return a.sentinel, a.optimize }

// TestAlgebraOptionalInterface verifies that Union/Intersection/Difference/SymmetricDifference
// use any first operand's Algebra implementation when it reports true, and fall back to the
// generic element-wise path when it reports false.
func TestAlgebraOptionalInterface(t *testing.T) {
	t.Parallel()

	sentinel := NewWith(42)
	inner := NewWith(1, 2)
	other := NewWith(2, 3)

	ops := []struct {
		name string
		f    func(a, b Set[int]) Set[int]
		want []int // generic-path result for inner vs other
	}{
		{"Union", Union[int], []int{1, 2, 3}},
		{"Intersection", Intersection[int], []int{2}},
		{"Difference", Difference[int], []int{1}},
		{"SymmetricDifference", SymmetricDifference[int], []int{1, 3}},
	}

	for _, op := range ops {
		optimized := op.f(&algebraStub{Set: inner, sentinel: sentinel, optimize: true}, other)
		if optimized != sentinel {
			t.Fatalf("%s: expected the Algebra result to be returned, got %v", op.name, Elements(optimized))
		}

		fallback := op.f(&algebraStub{Set: inner, sentinel: sentinel, optimize: false}, other)
		if fallback == sentinel {
			t.Fatalf("%s: Algebra reported false but its result was used", op.name)
		}
		got := Elements(fallback)
		slices.Sort(got)
		if !slices.Equal(got, op.want) {
			t.Fatalf("%s fallback = %v, want %v", op.name, got, op.want)
		}
	}

	// BitSet's Algebra methods report false for non-BitSet operands
	if _, ok := NewBitSetWith(1).Union(NewWith(2)); ok {
		t.Fatal("BitSet.Union(non-BitSet) must report false")
	}
}
