package sets

import (
	"slices"
	"testing"

	"pgregory.net/rapid"
)

// TestLockedDelegation differentially tests the locked wrappers' optional-interface delegation
// against the generic Map-based results: every combination of wrapped/bare operands must agree
// with the reference, algebra results must stay concurrency-safe wrappers, and inner sets that
// cannot optimize must decline into the generic path.
func TestLockedDelegation(t *testing.T) {
	t.Parallel()

	algebra := []struct {
		name string
		f    func(a, b Set[int]) Set[int]
	}{
		{"Union", Union[int]},
		{"Intersection", Intersection[int]},
		{"Difference", Difference[int]},
		{"SymmetricDifference", SymmetricDifference[int]},
	}
	preds := []struct {
		name string
		f    func(a, b Set[int]) bool
	}{
		{"Equal", Equal[int]},
		{"Disjoint", Disjoint[int]},
		{"Subset", Subset[int]},
		{"Superset", Superset[int]},
	}

	rapid.Check(t, func(t *rapid.T) {
		av := rapid.SliceOfN(rapid.IntRange(-64, 64), 0, 24).Draw(t, "A")
		var bv []int
		switch rapid.SampledFrom([]string{"independent", "equal", "subset", "disjoint"}).Draw(t, "shape") {
		case "independent":
			bv = rapid.SliceOfN(rapid.IntRange(-64, 64), 0, 24).Draw(t, "B")
		case "equal":
			bv = slices.Clone(av)
		case "subset":
			for _, v := range av {
				if rapid.Bool().Draw(t, "keep") {
					bv = append(bv, v)
				}
			}
		case "disjoint":
			bv = rapid.SliceOfN(rapid.IntRange(65, 128), 0, 24).Draw(t, "B")
		}
		am, bm := NewWith(av...), NewWith(bv...)

		// operand pairs covering wrapped×wrapped, wrapped×bare, cross-wrapper, and an inner type
		// mismatch that must decline into the generic path
		pairs := []struct {
			name string
			a, b Set[int]
		}{
			{"lockedBitSet×lockedBitSet", NewLockedWrapping(Set[int](NewBitSetWith(av...))), NewLockedWrapping(Set[int](NewBitSetWith(bv...)))},
			{"lockedSorted×lockedSorted", NewLockedWrapping(Set[int](NewSortedSetWith(av...))), NewLockedWrapping(Set[int](NewSortedSetWith(bv...)))},
			{"lockedBitSet×bareBitSet", NewLockedWrapping(Set[int](NewBitSetWith(av...))), NewBitSetWith(bv...)},
			{"lockedBitSet×lockedOrderedBitSet", NewLockedWrapping(Set[int](NewBitSetWith(av...))), NewLockedOrderedWrapping(OrderedSet[int](NewBitSetWith(bv...)))},
			{"lockedOrderedSorted×lockedOrderedSorted", NewLockedOrderedWrapping(OrderedSet[int](NewSortedSetWith(av...))), NewLockedOrderedWrapping(OrderedSet[int](NewSortedSetWith(bv...)))},
			{"lockedBitSet×lockedSorted (declines)", NewLockedWrapping(Set[int](NewBitSetWith(av...))), NewLockedWrapping(Set[int](NewSortedSetWith(bv...)))},
			{"lockedMap×lockedMap (declines)", NewLockedWrapping(Set[int](NewWith(av...))), NewLockedWrapping(Set[int](NewWith(bv...)))},
		}

		for _, pair := range pairs {
			for _, op := range algebra {
				want := op.f(am, bm)
				got := op.f(pair.a, pair.b)
				if !Equal(got, want) {
					t.Fatalf("%s %s = %v, want %v (A=%v B=%v)", pair.name, op.name, Elements(got), Elements(want), av, bv)
				}
				if _, ok := got.(Locker); !ok {
					t.Fatalf("%s %s returned %T, which is not concurrency-safe", pair.name, op.name, got)
				}
			}
			for _, p := range preds {
				want := p.f(am, bm)
				if got := p.f(pair.a, pair.b); got != want {
					t.Fatalf("%s %s = %v, want %v (A=%v B=%v)", pair.name, p.name, got, want, av, bv)
				}
			}
			if len(av) > 0 {
				if got, want := Max(pair.a), Max(am); got != want {
					t.Fatalf("%s Max = %d, want %d", pair.name, got, want)
				}
				if got, want := Min(pair.a), Min(am); got != want {
					t.Fatalf("%s Min = %d, want %d", pair.name, got, want)
				}
			}
		}
	})
}

// TestLockedDelegationResultTypes pins what the delegated fast paths produce: a new wrapper of
// the receiver's kind around the inner result type, computed via the inner fast path.
func TestLockedDelegationResultTypes(t *testing.T) {
	t.Parallel()

	a := NewLockedWrapping(Set[int](NewBitSetWith(1, 2)))
	b := NewLockedWrapping(Set[int](NewBitSetWith(2, 3)))
	got, ok := a.(*Locked[int]).Union(b)
	if !ok {
		t.Fatal("Locked(BitSet).Union(Locked(BitSet)) declined")
	}
	l, ok := got.(*Locked[int])
	if !ok {
		t.Fatalf("delegated Union returned %T, want *Locked[int]", got)
	}
	if _, ok := l.set.(*BitSet[int]); !ok {
		t.Fatalf("delegated Union inner is %T, want *BitSet[int]", l.set)
	}

	oa := NewLockedOrderedWrapping(OrderedSet[int](NewSortedSetWith(1, 2)))
	ogot, ok := oa.(*LockedOrdered[int]).Union(NewSortedSetWith(2, 3))
	if !ok {
		t.Fatal("LockedOrdered(SortedSet).Union(SortedSet) declined")
	}
	lo, ok := ogot.(*LockedOrdered[int])
	if !ok {
		t.Fatalf("delegated Union returned %T, want *LockedOrdered[int]", ogot)
	}
	if _, ok := lo.set.(*SortedSet[int]); !ok {
		t.Fatalf("delegated Union inner is %T, want *SortedSet[int]", lo.set)
	}

	// method-level declines: inner set without the interface, and mismatched inner types
	m := NewLockedWrapping(Set[int](NewWith(1)))
	if _, ok := m.(*Locked[int]).Union(b); ok {
		t.Fatal("Locked(Map).Union must decline")
	}
	if _, ok := a.(*Locked[int]).Union(NewLockedWrapping(Set[int](NewSortedSetWith(1)))); ok {
		t.Fatal("Locked(BitSet).Union(Locked(SortedSet)) must decline")
	}
}

// TestLockedDelegationContention pins the deadlock-avoidance rule: when the operand wrapper's
// lock is write-held, delegation declines immediately instead of blocking.
func TestLockedDelegationContention(t *testing.T) {
	t.Parallel()

	a := NewLockedWrapping(Set[int](NewBitSetWith(1, 2))).(*Locked[int])
	b := NewLockedWrapping(Set[int](NewBitSetWith(2, 3))).(*Locked[int])

	b.Lock()
	if _, ok := a.Union(b); ok {
		t.Fatal("Union with a write-locked operand must decline")
	}
	if _, ok := a.Equal(b); ok {
		t.Fatal("Equal with a write-locked operand must decline")
	}
	if _, ok := a.Subset(b); ok {
		t.Fatal("Subset with a write-locked operand must decline")
	}
	// the receiver's own lock is unaffected: bare operands still delegate
	if _, ok := a.Union(NewBitSetWith(9)); !ok {
		t.Fatal("Union with a bare operand declined during unrelated contention")
	}
	b.Unlock()

	if _, ok := a.Union(b); !ok {
		t.Fatal("Union declined after the contention was released")
	}

	// self-operand: the try-acquired second read lock on the same wrapper succeeds
	if c, ok := a.Union(a); !ok || !Equal(c, a) {
		t.Fatal("Union with the receiver itself as operand must delegate and equal the receiver")
	}
}

// TestLockedDelegationNilAndZero pins nil/zero-value safety: typed-nil wrappers and zero-value
// wrappers (nil inner set) decline both as receivers and as unwrapped operands.
func TestLockedDelegationNilAndZero(t *testing.T) {
	t.Parallel()

	var nl *Locked[int]
	var nlo *LockedOrdered[int]
	if _, ok := nl.Union(NewWith(1)); ok {
		t.Fatal("nil Locked.Union reported handled")
	}
	if _, ok := nl.Equal(NewWith(1)); ok {
		t.Fatal("nil Locked.Equal reported handled")
	}
	if _, ok := nl.Max(); ok {
		t.Fatal("nil Locked.Max reported ok")
	}
	if _, ok := nlo.Union(NewSortedSetWith(1)); ok {
		t.Fatal("nil LockedOrdered.Union reported handled")
	}
	if _, ok := nlo.Max(); ok {
		t.Fatal("nil LockedOrdered.Max reported ok")
	}

	var zl Locked[int]
	if _, ok := zl.Union(NewBitSetWith(1)); ok {
		t.Fatal("zero-value Locked.Union reported handled")
	}
	if _, ok := zl.Max(); ok {
		t.Fatal("zero-value Locked.Max reported ok")
	}
	// zero-value wrappers as operands decline the unwrap, so the receiver declines too
	a := NewLockedWrapping(Set[int](NewBitSetWith(1))).(*Locked[int])
	if _, ok := a.Union(&zl); ok {
		t.Fatal("Union with a zero-value locked operand must decline")
	}
	if _, ok := a.Union(nl); ok {
		t.Fatal("Union with a typed-nil locked operand must decline")
	}
}
