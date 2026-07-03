package sets

import (
	"cmp"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"iter"
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
//
// The O(N) Add/Remove shifts make SortedSet best suited to read-heavy workloads (build once, query
// many times). For add/remove-heavy workloads prefer Ordered or Map.
type SortedSet[M cmp.Ordered] struct {
	el []M // sorted ascending, no duplicates
}

var _ OrderedSet[int] = new(SortedSet[int])
var _ driver.Valuer = new(SortedSet[int])

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

// Pop removes and returns the largest element of the set. If the set is empty, it returns the zero
// value of M and false. Removing from the end avoids the shift other removals pay, so Pop is O(1).
func (s *SortedSet[M]) Pop() (M, bool) {
	if len(s.el) == 0 {
		var m M
		return m, false
	}
	m := s.el[len(s.el)-1]
	s.el = slices.Delete(s.el, len(s.el)-1, len(s.el))
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
