package sets

import (
	"hash/maphash"
	"iter"
	"slices"
)

// SetHasher hashes and compares sets by their contents, so a set can be used as an element of a
// hash-based data structure. Set values are otherwise compared by identity: two *Map values
// holding the same elements are different keys of a built-in map. SetHasher's zero value is ready
// to use and it has no state.
//
// It implements the maphash.Hasher interface added in Go 1.27 for Set[M] values. The interface is
// structural, so SetHasher itself does not require Go 1.27.
//
// Equality is that of the Equal function: sets are equal when they hold the same elements, whatever
// their implementation or iteration order. Ordered sets that differ only in order therefore hash
// alike; use EqualOrdered if order matters to you.
//
// SetHasher relies on the elements' == relation, as every implementation in this package does. A
// custom Set implementation whose Contains uses some other equivalence (case-insensitive strings,
// say) can report two sets as Equal that hash differently, which breaks the maphash.Hasher contract.
//
// A set's hash changes when its elements do, so a set must not be modified while it is held in a
// hash-based data structure: it would no longer be found. Clone it first if it needs to change.
type SetHasher[M comparable] struct{}

// Hash adds the contents of s to the data hashed by h, in O(n). It reads s only through Iterator, so
// it is as safe for concurrent use as the set's own iteration.
func (SetHasher[M]) Hash(h *maphash.Hash, s Set[M]) {
	var sub maphash.Hash
	sub.SetSeed(h.Seed())

	// Each element is hashed separately and the results combined with XOR, which is commutative, so
	// iteration order doesn't affect the result. Elements are unique, so none can cancel another out.
	var accum uint64
	for m := range s.Iterator {
		sub.Reset()
		maphash.WriteComparable(&sub, m)
		accum ^= sub.Sum64()
	}
	maphash.WriteComparable(h, accum)
}

// Equal reports whether x and y contain the same elements. See the package-level Equal function.
func (SetHasher[M]) Equal(x, y Set[M]) bool {
	return Equal(x, y)
}

// Distinct returns an iterator over the sets in seq, skipping any set Equal to one already yielded.
// The first occurrence of each is kept, in the order of seq. Sets are compared as they are by
// SetHasher, so its limits apply: the sets must not be modified during iteration.
func Distinct[K comparable](seq iter.Seq[Set[K]]) iter.Seq[Set[K]] {
	return func(yield func(Set[K]) bool) {
		var hasher SetHasher[K]
		seed := maphash.MakeSeed()
		seen := make(map[uint64][]Set[K])

		for s := range seq {
			var h maphash.Hash
			h.SetSeed(seed)
			hasher.Hash(&h, s)
			key := h.Sum64()

			// Sets with the same hash aren't necessarily equal.
			isS := func(other Set[K]) bool { return hasher.Equal(other, s) }
			if slices.ContainsFunc(seen[key], isS) {
				continue
			}
			seen[key] = append(seen[key], s)
			if !yield(s) {
				return
			}
		}
	}
}
