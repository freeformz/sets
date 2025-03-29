// A set package with various set helper functions and set types.
// Supports tha latest Go versions and features including generics
// and iterators. The package is designed to be simple and easy to use
// alongside the standard library.
package sets

import (
	"cmp"
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

	// Clone returns a copy of the set. Ordered sets maintain their order.
	Clone() Set[M]

	// Contains returns true if the set contains the element.
	Contains(M) bool

	// Iterator for the set elements. The yield function is called for each element in the set. If the yield function
	// returns false, the iteration is stopped. The order of iteration is not guaranteed unless the set is ordered.
	// Changing the set while iterating over it is undefined. It may or may not be safe to change the set or allowed
	// based on the implementation in use. Implementations must document their iteration safety.
	Iterator(yield func(M) bool)

	// NewEmpty returns a new empty set of the same underlying type. If the type is ordered, the new set will also be ordered.
	NewEmpty() Set[M]

	// Pop returns and removes a random element from the set. The second return value is false if nothing was removed.
	Pop() (M, bool)

	// Remove an element from the set. Returns true if the element was in the set.
	Remove(M) bool

	// String representation of the set.
	String() string
}

// Elements of the set as a slice. This is a convenience wrapper around slices.Collect(s.Iterator)
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

// Union of the two sets. Returns a new set (of the same underling type as a) with all elements from both sets.
func Union[K comparable](a, b Set[K]) Set[K] {
	c := a.Clone()
	AppendSeq(c, b.Iterator)
	return c
}

// Intersection of the two sets. Returns a new set (of the same underlying type as a) with elements that are in both sets.
func Intersection[K comparable](a, b Set[K]) Set[K] {
	c := a.Clone()
	for k := range a.Iterator {
		if !b.Contains(k) {
			c.Remove(k)
		}
	}
	return c
}

// Difference of the two sets. Returns a new set (of the same underlying type as a) with elements that are in the first set but not in the second set.
func Difference[K comparable](a, b Set[K]) Set[K] {
	c := a.Clone()
	for k := range a.Iterator {
		if b.Contains(k) {
			c.Remove(k)
		}
	}
	return c
}

// SymmetricDifference of the two sets. Returns a new set (of the same underlying type as a) with elements that are not in both sets.
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
	if a.Cardinality() > b.Cardinality() {
		return false
	}
	for k := range a.Iterator {
		if !b.Contains(k) {
			return false
		}
	}
	return true
}

// Superset returns true if all elements in the second set are also in the first set.
func Superset[K comparable](a, b Set[K]) bool {
	if a.Cardinality() < b.Cardinality() {
		return false
	}
	return Subset(b, a)
}

// Equal returns true if the two sets contain the same elements.
func Equal[K comparable](a, b Set[K]) bool {
	// can't be equal if they don't have the same cardinality
	if a.Cardinality() != b.Cardinality() {
		return false
	}
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

// Iter2 is a helper function that simplifies iterating over a set when an "index" is needed, by providing a pseudo-index
// to the yield function. The index is not stable across iterations. The yield function is called for each element in the
// set. If the yield function returns false, the iteration is stopped.
func Iter2[K comparable](iter iter.Seq[K]) func(func(i int, k K) bool) {
	var i int
	return func(yield func(i int, k K) bool) {
		for k := range iter {
			if !yield(i, k) {
				return
			}
			i++
		}
	}
}

// Max element in the set. Set must be a set of cmp.Ordered elements. Panics if the set is empty.
func Max[K cmp.Ordered](s Set[K]) K {
	if s.Cardinality() == 0 {
		panic("empty set")
	}

	var mx K
	for i, k := range Iter2(s.Iterator) {
		if i == 0 {
			mx = k
			continue
		}
		mx = max(mx, k)
	}
	return mx
}

// Min element in the set. Set must be a set of cmp.Ordered elements. Panics if the set is empty.
func Min[K cmp.Ordered](s Set[K]) K {
	if s.Cardinality() == 0 {
		panic("empty set")
	}

	var mn K
	for i, k := range Iter2(s.Iterator) {
		if i == 0 {
			mn = k
			continue
		}
		mn = min(mn, k)
	}
	return mn
}

// Chunk the set into n sets of equal size. The last set will have fewer elements if the cardinality of the set is not a multiple of n.
func Chunk[K comparable](s Set[K], n int) iter.Seq[Set[K]] {
	return func(yield func(Set[K]) bool) {
		chunk := s.NewEmpty()
		for i, v := range Iter2(s.Iterator) {
			if i%n == 0 {
				if chunk.Cardinality() > 0 {
					if !yield(chunk) {
						return
					}
				}
				chunk = s.NewEmpty()
				chunk.Add(v)
			} else {
				chunk.Add(v)
			}
		}
		if chunk.Cardinality() > 0 {
			yield(chunk)
		}
	}
}

// IsEmpty returns true if the set is empty.
func IsEmpty[K comparable](s Set[K]) bool {
	return s.Cardinality() == 0
}

// MapBy applies the function to each element in the set and returns a new set with the results.
func MapBy[K comparable, V comparable](s Set[K], f func(K) V) Set[V] {
	m := New[V]()
	for k := range s.Iterator {
		m.Add(f(k))
	}
	return m
}

// MapTo applies the function to each element in the set and adds the results to the destination set.
func MapTo[K comparable, V comparable](s Set[K], d Set[V], f func(K) V) {
	for k := range s.Iterator {
		d.Add(f(k))
	}
}

// MapToSlice applies the function to each element in the set and returns a slice with the results.
func MapToSlice[K comparable, V any](s Set[K], f func(K) V) []V {
	o := make([]V, 0, s.Cardinality())
	for v := range s.Iterator {
		o = append(o, f(v))
	}
	return o
}

// Filter applies the function to each element in the set and returns a new set with the elements for which the function
// returns true.
func Filter[K comparable](s Set[K], f func(K) bool) Set[K] {
	m := s.NewEmpty()
	for k := range s.Iterator {
		if f(k) {
			m.Add(k)
		}
	}
	return m
}

// FilterTo applies the function to each element in the set and adds the elements for which the function returns true to
// the destination set.
func FilterTo[K comparable](s Set[K], d Set[K], f func(K) bool) {
	s.Iterator(func(k K) bool {
		if f(k) {
			d.Add(k)
		}
		return true
	})
}

// Reduce applies the function to each element in the set and returns the accumulated value. "initial" is the initial
// value of the accumulator. The function is called with the accumulator and each element in turn. The result of the
// function is the new accumulator value. The final accumulator value is returned.
func Reduce[K comparable, O any](s Set[K], initial O, f func(agg O, k K) O) O {
	v := initial
	for k := range s.Iterator {
		v = f(v, k)
	}
	return v
}

// ForEach calls the function with each element in the set.
func ForEach[K comparable](s Set[K], f func(K)) {
	for k := range s.Iterator {
		f(k)
	}
}
