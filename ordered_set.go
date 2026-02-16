package sets

import (
	"cmp"
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

	// Sort the set in ascending order.
	Sort()

	// NewEmptyOrdered returns a new empty ordered set of the same underlying type.
	NewEmptyOrdered() OrderedSet[M]
}

// EqualOrdered returns true if the two OrderedSets contain the same elements in the same order. [cmp.Compare] is used
// to compare elements.
func EqualOrdered[K cmp.Ordered](a, b OrderedSet[K]) bool {
	// can't be equal if they don't have the same cardinality
	if a.Cardinality() != b.Cardinality() {
		return false
	}
	for i, ak := range a.Ordered {
		bv, ok := b.At(i)
		if !ok || cmp.Compare(ak, bv) != 0 {
			return false
		}
	}
	return true
}

// IsSorted returns true if the OrderedSet is sorted in ascending order. [cmp.Less] is used to compare elements.
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

// Sorted copy of the set. The original set is not modified.
func Sorted[K cmp.Ordered](s OrderedSet[K]) OrderedSet[K] {
	out := s.Clone().(OrderedSet[K])
	out.Sort()
	return out
}

// ReduceRight reduces the set from right to left using the given function. "initial" is the initial value of the
// accumulator. The function is called with the accumulator and the element backwards. The result of the function is the
// new accumulator value. The final accumulator value is returned.
func ReduceRight[K cmp.Ordered, O any](s OrderedSet[K], initial O, fn func(agg O, k K) O) O {
	out := initial
	for _, k := range s.Backwards {
		out = fn(out, k)
	}
	return out
}

func ForEachRight[K cmp.Ordered](s OrderedSet[K], fn func(k K)) {
	for _, k := range s.Backwards {
		fn(k)
	}
}

// First returns the first element of the ordered set. If the set is empty, the second return value is false.
func First[K cmp.Ordered](s OrderedSet[K]) (K, bool) {
	return s.At(0)
}

// Last returns the last element of the ordered set. If the set is empty, the second return value is false.
func Last[K cmp.Ordered](s OrderedSet[K]) (K, bool) {
	return s.At(s.Cardinality() - 1)
}
