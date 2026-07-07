// A set package with various set helper functions and set types.
// Supports the latest Go versions and features including generics
// and iterators. The package is designed to be simple and easy to use
// alongside the standard library.
package sets

import (
	"cmp"
	"iter"
	"math/rand/v2"
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

// Elements of the set as a slice. Returns nil if the set is empty.
func Elements[K comparable](s Set[K]) []K {
	n := s.Cardinality()
	if n == 0 {
		return nil
	}
	out := make([]K, 0, n)
	for k := range s.Iterator {
		out = append(out, k)
	}
	return out
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

// Unioner is an optional interface that Set implementations can implement to provide an optimized
// implementation of the package-level Union function, which checks whether its first operand
// implements it. The Intersectioner, Differencer, and SymmetricDifferencer interfaces work the
// same way for the other set-algebra functions; implement whichever subset you can optimize.
// BitSet implements all four with word-wise operations; SortedSet implements all four with a
// single linear merge of the two sorted backing slices.
//
// The boolean return is deliberate: implementing one of these interfaces opts a type into an
// operation, and the boolean additionally lets each call decline a specific operand — typically
// anything but the implementation's own concrete type (e.g. BitSet.Union reports false unless
// other is also a *BitSet of the same element type). On false the caller runs the generic
// element-wise algorithm, which therefore lives in exactly one place: implementations never
// reproduce it, never risk getting it wrong, and a conservative implementation's worst outcome is
// generic speed, not an incorrect result. Declining is always safe.
//
// Contract, shared by all four interfaces: the method must not modify the receiver or the operand,
// and when reporting true it must return a new set, of the same underlying type as the receiver,
// holding the correct result.
type Unioner[M comparable] interface {
	// Union returns a new set with all elements from the receiver and other, or false if other
	// cannot be handled more efficiently than the generic element-wise fallback.
	Union(other Set[M]) (Set[M], bool)
}

// Intersectioner is an optional interface providing an optimized Intersection; see Unioner for the
// contract shared by the set-algebra optimization interfaces.
type Intersectioner[M comparable] interface {
	// Intersection returns a new set with the elements common to the receiver and other, or false
	// if other cannot be handled more efficiently than the generic element-wise fallback.
	Intersection(other Set[M]) (Set[M], bool)
}

// Differencer is an optional interface providing an optimized Difference; see Unioner for the
// contract shared by the set-algebra optimization interfaces.
type Differencer[M comparable] interface {
	// Difference returns a new set with the elements of the receiver that are not in other, or
	// false if other cannot be handled more efficiently than the generic element-wise fallback.
	Difference(other Set[M]) (Set[M], bool)
}

// SymmetricDifferencer is an optional interface providing an optimized SymmetricDifference; see
// Unioner for the contract shared by the set-algebra optimization interfaces.
type SymmetricDifferencer[M comparable] interface {
	// SymmetricDifference returns a new set with the elements that are in exactly one of the
	// receiver and other, or false if other cannot be handled more efficiently than the generic
	// element-wise fallback.
	SymmetricDifference(other Set[M]) (Set[M], bool)
}

// Union of the two sets. Returns a new set (of the same underlying type as a) with all elements from both sets.
// If a implements Unioner, its optimized Union is used when it can handle b (e.g. two BitSets combine word-wise).
func Union[K comparable](a, b Set[K]) Set[K] {
	if u, ok := a.(Unioner[K]); ok {
		if c, ok := u.Union(b); ok {
			return c
		}
	}
	c := a.Clone()
	AppendSeq(c, b.Iterator)
	return c
}

// Intersection of the two sets. Returns a new set (of the same underlying type as a) with elements that are in both sets.
// If a implements Intersectioner, its optimized Intersection is used when it can handle b (e.g. two BitSets combine word-wise).
func Intersection[K comparable](a, b Set[K]) Set[K] {
	if i, ok := a.(Intersectioner[K]); ok {
		if c, ok := i.Intersection(b); ok {
			return c
		}
	}
	c := a.NewEmpty()
	for k := range a.Iterator {
		if b.Contains(k) {
			c.Add(k)
		}
	}
	return c
}

// Difference of the two sets. Returns a new set (of the same underlying type as a) with elements that are in the first set but not in the second set.
// If a implements Differencer, its optimized Difference is used when it can handle b (e.g. two BitSets combine word-wise).
func Difference[K comparable](a, b Set[K]) Set[K] {
	if d, ok := a.(Differencer[K]); ok {
		if c, ok := d.Difference(b); ok {
			return c
		}
	}
	c := a.NewEmpty()
	for k := range a.Iterator {
		if !b.Contains(k) {
			c.Add(k)
		}
	}
	return c
}

// SymmetricDifference of the two sets. Returns a new set (of the same underlying type as a) with elements that are not in both sets.
// If a implements SymmetricDifferencer, its optimized SymmetricDifference is used when it can handle b (e.g. two BitSets combine word-wise).
func SymmetricDifference[K comparable](a, b Set[K]) Set[K] {
	if sd, ok := a.(SymmetricDifferencer[K]); ok {
		if c, ok := sd.SymmetricDifference(b); ok {
			return c
		}
	}
	c := a.NewEmpty()
	for k := range a.Iterator {
		if !b.Contains(k) {
			c.Add(k)
		}
	}
	for k := range b.Iterator {
		if !a.Contains(k) {
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
	for k := range a.Iterator {
		if !b.Contains(k) {
			return false
		}
	}
	return true
}

// ContainsSeq returns true if the set contains all elements in the sequence. Returns true for an empty sequence (vacuous truth).
func ContainsSeq[K comparable](s Set[K], seq iter.Seq[K]) bool {
	for k := range seq {
		if !s.Contains(k) {
			return false
		}
	}
	return true
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
func Iter2[K comparable](seq iter.Seq[K]) func(func(i int, k K) bool) {
	return func(yield func(i int, k K) bool) {
		var i int
		for k := range seq {
			if !yield(i, k) {
				return
			}
			i++
		}
	}
}

// Maxer is an optional interface that Set implementations can implement to provide an optimized
// implementation of the package-level Max function, which checks whether its argument implements
// it; Minner works the same way for Min. Implement them when an extreme element can be found
// without iterating every element — always-sorted implementations answer from the ends of their
// storage (SortedSet in O(1); BitSet from the outermost set bit of its span).
//
// The boolean return is deliberate: reporting false — whether because the set is empty or because
// the receiver cannot answer efficiently — makes the caller fall back to the generic O(N)
// iteration, so declining is always safe. Contract, shared by both interfaces: the method must not
// modify the receiver, and when reporting true it must return the correct extreme element.
type Maxer[M cmp.Ordered] interface {
	// Max returns the largest element in the set, or false if the set is empty or the maximum
	// cannot be determined more efficiently than the generic iteration fallback.
	Max() (M, bool)
}

// Minner is an optional interface providing an optimized Min; see Maxer for the contract shared by
// the two extreme-element optimization interfaces.
type Minner[M cmp.Ordered] interface {
	// Min returns the smallest element in the set, or false if the set is empty or the minimum
	// cannot be determined more efficiently than the generic iteration fallback.
	Min() (M, bool)
}

// Max element in the set. Set must be a set of cmp.Ordered elements. Panics if the set is empty.
// If the set implements Maxer, its optimized Max is used when it can answer (e.g. SortedSet
// answers in O(1) from the end of its sorted storage).
func Max[K cmp.Ordered](s Set[K]) K {
	if s.Cardinality() == 0 {
		panic("empty set")
	}
	if mx, ok := s.(Maxer[K]); ok {
		if m, ok := mx.Max(); ok {
			return m
		}
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
// If the set implements Minner, its optimized Min is used when it can answer (e.g. SortedSet
// answers in O(1) from the start of its sorted storage).
func Min[K cmp.Ordered](s Set[K]) K {
	if s.Cardinality() == 0 {
		panic("empty set")
	}
	if mn, ok := s.(Minner[K]); ok {
		if m, ok := mn.Min(); ok {
			return m
		}
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

// Chunk the set into sets of n elements each. The last set will have fewer elements if the cardinality of the set is not a multiple of n.
// Panics if n <= 0.
func Chunk[K comparable](s Set[K], n int) iter.Seq[Set[K]] {
	if n <= 0 {
		panic("sets.Chunk: n must be > 0")
	}
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

// Any returns true if any element in the set satisfies the predicate. Returns false for an empty set.
func Any[K comparable](s Set[K], f func(K) bool) bool {
	for k := range s.Iterator {
		if f(k) {
			return true
		}
	}
	return false
}

// All returns true if all elements in the set satisfy the predicate. Returns true for an empty set (vacuous truth).
func All[K comparable](s Set[K], f func(K) bool) bool {
	for k := range s.Iterator {
		if !f(k) {
			return false
		}
	}
	return true
}

// ContainsAll returns true if the set contains all of the provided elements. Returns true if no elements are provided (vacuous truth).
func ContainsAll[K comparable](s Set[K], elements ...K) bool {
	for _, k := range elements {
		if !s.Contains(k) {
			return false
		}
	}
	return true
}

// ContainsAny returns true if the set contains at least one of the provided elements. Returns false if no elements are provided.
func ContainsAny[K comparable](s Set[K], elements ...K) bool {
	return slices.ContainsFunc(elements, s.Contains)
}

// Random returns a random element from the set without removing it. The second return value is false if the set is empty.
// For ordered sets, this uses indexed access (O(log n) for this package's ordered implementations). For unordered sets,
// this is O(n) via iteration.
func Random[K comparable](s Set[K]) (K, bool) {
	n := s.Cardinality()
	if n == 0 {
		var zero K
		return zero, false
	}
	if idx, ok := s.(interface{ At(int) (K, bool) }); ok {
		return idx.At(rand.IntN(n))
	}
	skip := rand.IntN(n)
	var i int
	for k := range s.Iterator {
		if i == skip {
			return k, true
		}
		i++
	}
	var zero K
	return zero, false
}
