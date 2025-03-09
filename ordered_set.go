package set

import (
	"cmp"
	"slices"
)

// OrderedSet is an extended set interface that implementations can implement to indicate that the set is ordered.
// OrderedSet sets have a guaranteed order of elements when iterating over them and are restricted to types that are in
// cmp.Ordered. The order of elements is not guaranteed for sets that do not implement this interface. OrderedSets will
// always return an ordered set when cloning.
type OrderedSet[M cmp.Ordered] interface {
	Set[M]

	// Ordered index, value iterator. The yield function is called for each element in the set in order. If the yield
	// function returns false, the iteration is stopped. Changing the set while iterating over it is undefined. It may
	// or may not be safe to change the set or allowed based on the implementation in use. Implementations must document
	// their iteration safety.
	Ordered(yield func(int, M) bool)

	// Backwards index, value iterator. The yield function is called for each element in the set in reverse order. If the
	// yield function returns false, the iteration is stopped. Changing the set while iterating over it is undefined. It
	// may or may not be safe to change the set or allowed based on the implementation in use. Implementations must
	// document their iteration safety.
	Backwards(yield func(int, M) bool)

	// At returns the element at the index. If the index is out of bounds, the second return value is false.
	At(i int) (M, bool)

	// Index returns the index of the element in the set, or -1 if not present.
	Index(M) int

	// Sort the set in ascending order in place.
	Sort()

	// NewEmptyOrdered returns a new empty ordered set of the same underlying type.
	NewEmptyOrdered() OrderedSet[M]
}

// EqualOrdered returns true if the two OrderedSets contain the same elements in the same order.
func EqualOrdered[K cmp.Ordered](a, b OrderedSet[K]) bool {
	// can't be equal if they don't have the same cardinality
	if a.Cardinality() != b.Cardinality() {
		return false
	}
	bv := slices.Collect(b.Iterator)
	for ai, ak := range a.Ordered {
		if ak != bv[ai] {
			return false
		}
	}
	return true
}

// IsSorted returns true if the OrderedSet is sorted in ascending order.
func IsSorted[K cmp.Ordered](s OrderedSet[K]) bool {
	var prev K
	for i, k := range s.Ordered {
		if i != 0 && cmp.Less(k, prev) {
			return false
		}
		prev = k
	}
	return true
}

// Reverse returns a new OrderedSet with the elements in the reverse order of the original OrderedSet.
func Reverse[K cmp.Ordered](s OrderedSet[K]) OrderedSet[K] {
	out := s.NewEmptyOrdered()
	AppendSeq(out, func(yield func(k K) bool) {
		s.Backwards(func(_ int, k K) bool {
			return yield(k)
		})
	})
	return out
}
