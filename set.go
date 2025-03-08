// A set package with various set helper functions and set types.
// Supports tha latest Go versions and features including generics
// and iterators. The package is designed to be simple and easy to use
// alongside the standard library.
package set

import (
	"iter"
	"slices"
)

// Set is a collection of unique elements. The elements must be comparable. Each set implementation must implement this
// interface as the set functions within this package build on top of this interface.
type Set[M comparable] interface {
	// Add an element to the set. Returns true if the element was not already in the set.
	Add(M) bool

	// Cardinality of the set (number of elements in the set).
	Cardinality() int

	// Clear removes all elements from the set and returns the number of elements removed.
	Clear() int

	// Clone returns a deep copy of the set.
	Clone() Set[M]

	// Contains returns true if the set contains the element.
	Contains(M) bool

	// Iterator for the set elements. The yield function is called for each element in the set. If the yield function
	// returns false, the iteration is stopped. The order of iteration is not guaranteed unless the set is ordered.
	// Changing the set while iterating over it is undefined. It may or may not be safe to change the set or allowed
	// based on the implementation in use. Implementations must document their iteration safety.
	Iterator(yield func(M) bool)

	// Remove an element from the set. Returns true if the element was in the set.
	Remove(M) bool
}

// Elements returns a slice of all elements in the set. This is a convenience wrapper around slices.Collect(s.Iterator)
func Elements[K comparable](s Set[K]) []K {
	return slices.Collect(s.Iterator)
}

// AppendSeq appends all elements from the sequence to the set.
func AppendSeq[K comparable](s Set[K], seq iter.Seq[K]) int {
	var n int
	for k := range seq {
		if s.Add(k) {
			n++
		}
	}
	return n
}

// RemoveSeq removes all elements from the set that are in the sequence.
func RemoveSeq[K comparable](s Set[K], seq iter.Seq[K]) int {
	var n int
	for k := range seq {
		if s.Remove(k) {
			n++
		}
	}
	return n
}

// Union of the two sets. Returns a new set with all elements from both sets.
func Union[K comparable](a, b Set[K]) Set[K] {
	c := a.Clone()
	AppendSeq(c, b.Iterator)
	return c
}

// Intersection of the two sets. Returns a new set with elements that are in both sets.
func Intersection[K comparable](a, b Set[K]) Set[K] {
	c := a.Clone()
	for k := range a.Iterator {
		if !b.Contains(k) {
			c.Remove(k)
		}
	}
	return c
}

// Difference of the two sets. Returns a new set with elements that are in the first set but not in the second set.
func Difference[K comparable](a, b Set[K]) Set[K] {
	c := a.Clone()
	for k := range a.Iterator {
		if b.Contains(k) {
			c.Remove(k)
		}
	}
	return c
}

// SymmetricDifference of the two sets. Returns a new set with elements that are not in both sets.
func SymmetricDifference[K comparable](a, b Set[K]) Set[K] {
	c := a.Clone()
	for k := range b.Iterator {
		if a.Contains(k) {
			c.Remove(k)
		} else {
			c.Add(k)
		}
	}

	return c
}

// Subset returns true if all elements in the first set are also in the second set.
func Subset[K comparable](a, b Set[K]) bool {
	for k := range a.Iterator {
		if !b.Contains(k) {
			return false
		}
	}
	return true
}

// Superset returns true if all elements in the second set are also in the first set.
func Superset[K comparable](a, b Set[K]) bool {
	return Subset(b, a)
}

// Equal returns true if the two sets contain the same elements.
func Equal[K comparable](a, b Set[K]) bool {
	return Subset(a, b) && Subset(b, a)
}

// ContainsSeq returns true if the set contains all elements in the sequence. Empty sets are considered to contain only empty sequences.
func ContainsSeq[K comparable](s Set[K], seq iter.Seq[K]) bool {
	noitems := true
	for k := range seq {
		noitems = false
		if !s.Contains(k) {
			return false
		}
	}
	return (s.Cardinality() == 0 && noitems) || !noitems
}

// Disjoint returns true if the two sets have no elements in common.
func Disjoint[K comparable](a, b Set[K]) bool {
	for k := range a.Iterator {
		if b.Contains(k) {
			return false
		}
	}
	return true
}
