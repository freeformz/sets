package sets

import (
	"cmp"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"iter"
	"math/rand/v2"
	"slices"
)

// SortedSet maintains its elements in ascending sorted order at all times. Unlike Ordered, which
// preserves insertion order and can only be sorted transiently via Sort, the sorted order of a
// SortedSet is an invariant that Add maintains. It is backed by a single sorted slice with no
// companion map, so it uses less memory per element than the other implementations and iterates with
// good cache locality. It is not safe for concurrent use; wrap it with NewLockedOrderedWrapping when
// concurrency is needed.
//
// SortedSet's zero value is ready to use.
//
// Complexity:
//   - Add: O(N) (O(log N) search + shift)
//   - Remove: O(N) (O(log N) search + shift)
//   - Contains: O(log N)
//   - At: O(1)
//   - Index: O(log N)
//   - Iterator: O(N)
//   - Max, Min: O(1)
//   - Union, Intersection, Difference, SymmetricDifference with another SortedSet of the same
//     element type: O(N+M), a single linear merge of the two sorted slices, via the package-level
//     functions (SortedSet implements the optional Unioner, Intersectioner, Differencer, and
//     SymmetricDifferencer interfaces)
//   - Equal, Disjoint, Subset (and thus Superset) with another SortedSet of the same element
//     type: O(N+M) worst case, a short-circuiting scan of the two sorted slices with no hashing
//     or allocation, via the package-level functions (SortedSet implements the optional Equaler,
//     Disjointer, and Subsetter interfaces)
//
// The O(N) Add/Remove shifts make SortedSet best suited to read-heavy workloads (build once, query
// many times). For add/remove-heavy workloads prefer Ordered or Map.
type SortedSet[M cmp.Ordered] struct {
	el []M // sorted ascending, no duplicates
}

var _ OrderedSet[int] = new(SortedSet[int])
var _ driver.Valuer = new(SortedSet[int])
var _ Unioner[int] = new(SortedSet[int])
var _ Intersectioner[int] = new(SortedSet[int])
var _ Differencer[int] = new(SortedSet[int])
var _ SymmetricDifferencer[int] = new(SortedSet[int])
var _ Maxer[int] = new(SortedSet[int])
var _ Minner[int] = new(SortedSet[int])
var _ Equaler[int] = new(SortedSet[int])
var _ Disjointer[int] = new(SortedSet[int])
var _ Subsetter[int] = new(SortedSet[int])

// NewSortedSet returns an empty *SortedSet[M].
func NewSortedSet[M cmp.Ordered]() *SortedSet[M] {
	return &SortedSet[M]{el: make([]M, 0)}
}

// NewSortedSetFrom returns a new *SortedSet[M] filled with the values from the sequence.
func NewSortedSetFrom[M cmp.Ordered](seq iter.Seq[M]) *SortedSet[M] {
	// bulk sort + compact instead of adding element by element, which would pay an
	// O(N) insertion shift per element
	el := slices.Collect(seq)
	slices.Sort(el)
	return &SortedSet[M]{el: slices.Compact(el)}
}

// NewSortedSetWith returns a new *SortedSet[M] with the values provided.
func NewSortedSetWith[M cmp.Ordered](m ...M) *SortedSet[M] {
	return NewSortedSetFrom(slices.Values(m))
}

// Contains returns true if the set contains the element.
func (s *SortedSet[M]) Contains(m M) bool {
	_, ok := slices.BinarySearch(s.el, m)
	return ok
}

// Clear clears the set and returns the number of elements removed.
func (s *SortedSet[M]) Clear() int {
	n := len(s.el)
	clear(s.el) // zero the retained backing array so element values can be collected
	s.el = s.el[:0]
	return n
}

// Add an element to the set. Returns true if the element was added, false if it was already present.
// The element is inserted at its sorted position.
func (s *SortedSet[M]) Add(m M) bool {
	i, ok := slices.BinarySearch(s.el, m)
	if ok {
		return false
	}
	s.el = slices.Insert(s.el, i, m)
	return true
}

// Remove an element from the set. Returns true if the element was removed, false if it was not present.
func (s *SortedSet[M]) Remove(m M) bool {
	i, ok := slices.BinarySearch(s.el, m)
	if !ok {
		return false
	}
	s.el = slices.Delete(s.el, i, i+1)
	return true
}

// Cardinality returns the number of elements in the set.
func (s *SortedSet[M]) Cardinality() int {
	if s == nil {
		return 0
	}
	return len(s.el)
}

// Iterator yields all elements in the set in ascending order.
func (s *SortedSet[M]) Iterator(yield func(M) bool) {
	for _, v := range s.el {
		if !yield(v) {
			return
		}
	}
}

// Clone returns a copy of the set. The underlying type is the same as the original set.
func (s *SortedSet[M]) Clone() Set[M] {
	return &SortedSet[M]{el: slices.Clone(s.el)}
}

// Ordered iteration yields the index and value of each element in the set in ascending order.
func (s *SortedSet[M]) Ordered(yield func(int, M) bool) {
	for i, v := range s.el {
		if !yield(i, v) {
			return
		}
	}
}

// Backwards iteration yields the index and value of each element in the set in descending order.
func (s *SortedSet[M]) Backwards(yield func(int, M) bool) {
	for i := len(s.el) - 1; i >= 0; i-- {
		if !yield(i, s.el[i]) {
			return
		}
	}
}

// Range returns an iterator over the elements v for which lo <= v <= hi, in ascending order. The
// lower bound is located by binary search, so a call costs O(log N) plus the number of elements
// yielded. If lo > hi the iterator yields nothing.
func (s *SortedSet[M]) Range(lo, hi M) iter.Seq[M] {
	return func(yield func(M) bool) {
		start, _ := slices.BinarySearch(s.el, lo)
		for _, v := range s.el[start:] {
			if cmp.Less(hi, v) || !yield(v) {
				return
			}
		}
	}
}

// NewEmptyOrdered returns a new empty ordered set of the same underlying type.
func (s *SortedSet[M]) NewEmptyOrdered() OrderedSet[M] {
	return NewSortedSet[M]()
}

// NewEmpty returns a new empty set of the same underlying type.
func (s *SortedSet[M]) NewEmpty() Set[M] {
	return NewSortedSet[M]()
}

// Pop removes and returns a random element from the set. If the set is empty, it returns the zero
// value of M and false. Removing an element other than the largest shifts the elements after it,
// so Pop is O(N).
func (s *SortedSet[M]) Pop() (M, bool) {
	if len(s.el) == 0 {
		var m M
		return m, false
	}
	i := rand.IntN(len(s.el))
	m := s.el[i]
	s.el = slices.Delete(s.el, i, i+1)
	return m, true
}

// Sort is a no-op: the set is always sorted in ascending order.
func (s *SortedSet[M]) Sort() {}

// At returns the element at the index. If the index is out of bounds, the second return value is false.
func (s *SortedSet[M]) At(i int) (M, bool) {
	if i < 0 || i >= len(s.el) {
		var zero M
		return zero, false
	}
	return s.el[i], true
}

// Index returns the index of the element in the set, or -1 if not present.
func (s *SortedSet[M]) Index(m M) int {
	i, ok := slices.BinarySearch(s.el, m)
	if !ok {
		return -1
	}
	return i
}

// String returns a string representation of the set. It returns a string of the form SortedSet[T](<elements>).
func (s *SortedSet[M]) String() string {
	var m M
	return fmt.Sprintf("SortedSet[%T](%v)", m, s.el)
}

// Value implements the driver.Valuer interface. It returns the JSON representation of the set.
func (s *SortedSet[M]) Value() (driver.Value, error) {
	return s.MarshalJSON()
}

// MarshalJSON implements json.Marshaler. It will marshal the set into a JSON array of the elements in
// the set in ascending order. If the set is empty an empty JSON array is returned.
func (s *SortedSet[M]) MarshalJSON() ([]byte, error) {
	if len(s.el) == 0 {
		return []byte("[]"), nil
	}

	d, err := json.Marshal(s.el)
	if err != nil {
		return d, fmt.Errorf("marshaling sorted set: %w", err)
	}
	return d, nil
}

// UnmarshalJSON implements json.Unmarshaler. It expects a JSON array of the elements in the set. The
// elements do not need to be sorted or unique; the set sorts and de-duplicates them. If the JSON is
// invalid, it returns an error and the set is left unchanged.
func (s *SortedSet[M]) UnmarshalJSON(d []byte) error {
	t := make([]M, 0)
	if err := json.Unmarshal(d, &t); err != nil {
		return fmt.Errorf("unmarshaling sorted set: %w", err)
	}

	slices.Sort(t)
	s.el = slices.Compact(t)
	return nil
}

// Scan implements the sql.Scanner interface. It scans the value from the database into the set. It expects a JSON array
// of the elements in the set. If the JSON is invalid an error is returned. If the value is nil an empty set is
// returned.
func (s *SortedSet[M]) Scan(src any) error {
	return scanValue[M](src, s.Clear, s.UnmarshalJSON)
}

// Max implements Maxer: it returns the largest element, read in O(1) from the end of the sorted
// backing slice. The second return value is false if the set is empty. Prefer the package-level
// Max function, which uses this automatically.
func (s *SortedSet[M]) Max() (M, bool) {
	if s == nil || len(s.el) == 0 {
		var zero M
		return zero, false
	}
	return s.el[len(s.el)-1], true
}

// Min implements Minner: it returns the smallest element, read in O(1) from the start of the
// sorted backing slice. The second return value is false if the set is empty. Prefer the
// package-level Min function, which uses this automatically.
func (s *SortedSet[M]) Min() (M, bool) {
	if s == nil || len(s.el) == 0 {
		var zero M
		return zero, false
	}
	return s.el[0], true
}

// Union implements Unioner: when other is also a *SortedSet[M] it returns the union computed by a
// single O(N+M) linear merge of the two sorted slices and true; otherwise it returns nil and
// false. Prefer the package-level Union function, which uses this automatically and handles the
// fallback.
func (s *SortedSet[M]) Union(other Set[M]) (Set[M], bool) {
	o, ok := other.(*SortedSet[M])
	if !ok {
		return nil, false
	}
	return &SortedSet[M]{el: mergeSorted(s.el, o.el, true, true, true)}, true
}

// Intersection implements Intersectioner: when other is also a *SortedSet[M] it returns the
// intersection computed by a single O(N+M) linear merge of the two sorted slices and true;
// otherwise it returns nil and false. Prefer the package-level Intersection function, which uses
// this automatically and handles the fallback.
func (s *SortedSet[M]) Intersection(other Set[M]) (Set[M], bool) {
	o, ok := other.(*SortedSet[M])
	if !ok {
		return nil, false
	}
	return &SortedSet[M]{el: mergeSorted(s.el, o.el, false, false, true)}, true
}

// Difference implements Differencer: when other is also a *SortedSet[M] it returns the difference
// computed by a single O(N+M) linear merge of the two sorted slices and true; otherwise it returns
// nil and false. Prefer the package-level Difference function, which uses this automatically and
// handles the fallback.
func (s *SortedSet[M]) Difference(other Set[M]) (Set[M], bool) {
	o, ok := other.(*SortedSet[M])
	if !ok {
		return nil, false
	}
	return &SortedSet[M]{el: mergeSorted(s.el, o.el, true, false, false)}, true
}

// SymmetricDifference implements SymmetricDifferencer: when other is also a *SortedSet[M] it
// returns the symmetric difference computed by a single O(N+M) linear merge of the two sorted
// slices and true; otherwise it returns nil and false. Prefer the package-level
// SymmetricDifference function, which uses this automatically and handles the fallback.
func (s *SortedSet[M]) SymmetricDifference(other Set[M]) (Set[M], bool) {
	o, ok := other.(*SortedSet[M])
	if !ok {
		return nil, false
	}
	return &SortedSet[M]{el: mergeSorted(s.el, o.el, true, true, false)}, true
}

// Equal implements Equaler: when other is also a *SortedSet[M] it compares the two sorted backing
// slices element-wise and reports handled; otherwise it reports false, false. Prefer the
// package-level Equal function, which uses this automatically and handles the fallback.
func (s *SortedSet[M]) Equal(other Set[M]) (bool, bool) {
	o, ok := other.(*SortedSet[M])
	if !ok || s == nil || o == nil { // typed-nil operands decline to the generic path
		return false, false
	}
	return slices.Equal(s.el, o.el), true
}

// Disjoint implements Disjointer: when other is also a *SortedSet[M] it reports whether the two
// sorted backing slices share an element, found by a short-circuiting two-pointer scan, and
// handled; otherwise it reports false, false. Prefer the package-level Disjoint function, which
// uses this automatically and handles the fallback.
func (s *SortedSet[M]) Disjoint(other Set[M]) (bool, bool) {
	o, ok := other.(*SortedSet[M])
	if !ok || s == nil || o == nil { // typed-nil operands decline to the generic path
		return false, false
	}
	var i, j int
	for i < len(s.el) && j < len(o.el) {
		switch {
		case cmp.Less(s.el[i], o.el[j]):
			i++
		case cmp.Less(o.el[j], s.el[i]):
			j++
		default:
			return false, true
		}
	}
	return true, true
}

// Subset implements Subsetter: when other is also a *SortedSet[M] it reports whether every
// element of the receiver appears in other, found by a short-circuiting two-pointer scan, and
// handled; otherwise it reports false, false. Prefer the package-level Subset function, which
// uses this automatically and handles the fallback.
func (s *SortedSet[M]) Subset(other Set[M]) (bool, bool) {
	o, ok := other.(*SortedSet[M])
	if !ok || s == nil || o == nil { // typed-nil operands decline to the generic path
		return false, false
	}
	if len(s.el) > len(o.el) {
		return false, true
	}
	var j int
	for _, v := range s.el {
		for j < len(o.el) && cmp.Less(o.el[j], v) {
			j++
		}
		if j == len(o.el) || cmp.Less(v, o.el[j]) {
			return false, true
		}
		j++
	}
	return true, true
}

// mergeSorted linearly merges two ascending, duplicate-free slices into a new ascending,
// duplicate-free slice, keeping the elements found only in a, only in b, or in both, according to
// the flags. Each set-algebra operation is a flag combination: union keeps everything (true, true,
// true), intersection keeps only the common elements (false, false, true), difference keeps the
// elements only in a (true, false, false), and symmetric difference keeps the elements in exactly
// one input (true, true, false).
func mergeSorted[M cmp.Ordered](a, b []M, onlyA, onlyB, both bool) []M {
	n := 0
	if onlyA {
		n += len(a)
	}
	if onlyB {
		n += len(b)
	}
	if !onlyA && !onlyB && both { // intersection: bounded by the smaller input
		n = min(len(a), len(b))
	}
	out := make([]M, 0, n)
	var i, j int
	for i < len(a) && j < len(b) {
		switch {
		case cmp.Less(a[i], b[j]):
			if onlyA {
				out = append(out, a[i])
			}
			i++
		case cmp.Less(b[j], a[i]):
			if onlyB {
				out = append(out, b[j])
			}
			j++
		default:
			if both {
				out = append(out, a[i])
			}
			i++
			j++
		}
	}
	if onlyA {
		out = append(out, a[i:]...)
	}
	if onlyB {
		out = append(out, b[j:]...)
	}
	return out
}
