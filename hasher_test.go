package sets

import (
	"fmt"
	"hash/maphash"
	"math"
	"slices"
	"testing"

	"pgregory.net/rapid"
)

// hashSet returns the SetHasher hash of s under seed.
func hashSet[M comparable](seed maphash.Seed, s Set[M]) uint64 {
	var h maphash.Hash
	h.SetSeed(seed)
	var hasher SetHasher[M]
	hasher.Hash(&h, s)
	return h.Sum64()
}

// hasherImpls builds a set of each implementation from the same elements.
func hasherImpls(elems []int) map[string]Set[int] {
	return map[string]Set[int]{
		"Map":           NewWith(elems...),
		"SyncMap":       NewSyncMapWith(elems...),
		"Locked":        NewLockedWith(elems...),
		"Ordered":       NewOrderedWith(elems...),
		"LockedOrdered": NewLockedOrderedWith(elems...),
		"SortedSet":     NewSortedSetWith(elems...),
		"BitSet":        NewBitSetWith(elems...),
	}
}

func TestSetHasherAgreesWithEqual(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(t *rapid.T) {
		elems := rapid.SliceOf(rapid.IntRange(-64, 64)).Draw(t, "elems")
		reversed := slices.Clone(elems)
		slices.Reverse(reversed)
		seed := maphash.MakeSeed()
		want := hashSet[int](seed, NewWith(elems...))

		var hasher SetHasher[int]
		for name, s := range hasherImpls(elems) {
			if got := hashSet(seed, s); got != want {
				t.Errorf("%s: hash = %x, want %x", name, got, want)
			}
			for otherName, other := range hasherImpls(reversed) {
				if !hasher.Equal(s, other) {
					t.Errorf("Equal(%s, %s) = false for the same elements", name, otherName)
				}
				if got := hashSet(seed, other); got != want {
					t.Errorf("%s (reversed insertion): hash = %x, want %x", otherName, got, want)
				}
			}
		}
	})
}

func TestSetHasherDistinguishesSets(t *testing.T) {
	t.Parallel()

	// Distinct sets can collide in principle, but at 64 bits these won't.
	distinct := []Set[int]{
		New[int](),
		NewWith(0),
		NewWith(1),
		NewWith(1, 2),
		NewWith(2, 3),
		NewWith(1, 2, 3),
	}
	seed := maphash.MakeSeed()
	var hasher SetHasher[int]
	for i, a := range distinct {
		for _, b := range distinct[i+1:] {
			if hashSet(seed, a) == hashSet(seed, b) {
				t.Errorf("%v and %v hash alike", a, b)
			}
			if hasher.Equal(a, b) {
				t.Errorf("Equal(%v, %v) = true", a, b)
			}
		}
	}
}

func TestSetHasherUsesSeed(t *testing.T) {
	t.Parallel()

	s := NewWith(1, 2, 3)
	seedA, seedB := maphash.MakeSeed(), maphash.MakeSeed()
	if hashSet[int](seedA, s) == hashSet[int](seedB, s) {
		t.Error("different seeds produced the same hash")
	}

	seed := maphash.MakeSeed()
	if hashSet[int](seed, s) != hashSet[int](seed, s.Clone()) {
		t.Error("the same seed produced different hashes for a set and its clone")
	}
}

// TestSetHasherComposes verifies Hash adds to h rather than replacing it, as it must for a set to be
// hashed as part of a larger value.
func TestSetHasherComposes(t *testing.T) {
	t.Parallel()

	seed := maphash.MakeSeed()
	var hasher SetHasher[int]
	sum := func(prefix string, s Set[int]) uint64 {
		var h maphash.Hash
		h.SetSeed(seed)
		h.WriteString(prefix)
		hasher.Hash(&h, s)
		return h.Sum64()
	}

	s := NewWith(1, 2)
	if sum("a", s) == sum("b", s) {
		t.Error("data written before Hash did not affect the result")
	}
	if sum("a", s) != sum("a", NewOrderedWith(2, 1)) {
		t.Error("equal sets with the same prefix hash differently")
	}
}

// TestSetHasherFloats covers the elements for which == is unusual: Equal must imply the same hash,
// so -0 and +0 have to agree, while NaN never equals itself.
func TestSetHasherFloats(t *testing.T) {
	t.Parallel()

	seed := maphash.MakeSeed()
	var hasher SetHasher[float64]

	pos, neg := NewWith(0.0), NewWith(math.Copysign(0, -1))
	if !hasher.Equal(pos, neg) {
		t.Fatal("sets of +0 and -0 are not Equal")
	}
	if hashSet[float64](seed, pos) != hashSet[float64](seed, neg) {
		t.Error("sets of +0 and -0 are Equal but hash differently")
	}

	nan := NewWith(math.NaN())
	if hasher.Equal(nan, nan.Clone()) {
		t.Error("sets of NaN are Equal")
	}
}

func TestSetHasherElementTypes(t *testing.T) {
	t.Parallel()

	type point struct {
		X, Y int
		Name string
	}

	seed := maphash.MakeSeed()
	if hashSet[string](seed, NewWith("a", "b")) != hashSet[string](seed, NewSortedSetWith("b", "a")) {
		t.Error("equal string sets hash differently")
	}
	a := NewWith(point{1, 2, "a"}, point{3, 4, "b"})
	b := NewLockedWith(point{3, 4, "b"}, point{1, 2, "a"})
	if hashSet[point](seed, a) != hashSet[point](seed, b) {
		t.Error("equal struct sets hash differently")
	}
}

func TestDistinct(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(t *rapid.T) {
		lists := rapid.SliceOf(rapid.SliceOfN(rapid.IntRange(0, 4), 0, 4)).Draw(t, "lists")
		names := []string{"Map", "SyncMap", "Locked", "Ordered", "LockedOrdered", "SortedSet", "BitSet"}

		// The input mixes implementations, and want holds the first set seen with each contents.
		var input, want []Set[int]
		seen := make(map[string]bool)
		for i, list := range lists {
			s := hasherImpls(list)[names[i%len(names)]]
			input = append(input, s)

			key := fmt.Sprint(slices.Sorted(s.Iterator))
			if !seen[key] {
				seen[key] = true
				want = append(want, s)
			}
		}

		got := slices.Collect(Distinct(slices.Values(input)))
		if !slices.Equal(got, want) {
			t.Fatalf("Distinct(%v) = %v, want %v", input, got, want)
		}
	})
}

func TestDistinctIteratesRepeatedly(t *testing.T) {
	t.Parallel()

	seq := Distinct(slices.Values([]Set[int]{NewWith(1), NewWith(1), NewWith(2)}))
	for range 2 {
		if got := len(slices.Collect(seq)); got != 2 {
			t.Errorf("got %d sets, want 2", got)
		}
	}
}

func TestDistinctEmpty(t *testing.T) {
	t.Parallel()

	if got := slices.Collect(Distinct(slices.Values[[]Set[int]](nil))); got != nil {
		t.Errorf("got %v, want nil", got)
	}
}

// TestDistinctIsLazy verifies Distinct reads no further than it needs to and stops with its consumer.
func TestDistinctIsLazy(t *testing.T) {
	t.Parallel()

	var pulled int
	input := func(yield func(Set[int]) bool) {
		for i := range 10 {
			pulled++
			if !yield(NewWith(i)) {
				return
			}
		}
	}

	for range Distinct(input) {
		break
	}
	if pulled != 1 {
		t.Errorf("pulled %d sets, want 1", pulled)
	}
}
