package sets

import (
	"cmp"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"iter"
	"slices"
)

// Ordered sets maintains the order that the elements were added in.
type Ordered[M cmp.Ordered] struct {
	idx    map[M]int
	values []M
}

var _ OrderedSet[int] = new(Ordered[int])
var _ driver.Valuer = new(Ordered[int])

// NewOrdered returns an empty *Ordered[M].
func NewOrdered[M cmp.Ordered]() *Ordered[M] {
	return &Ordered[M]{
		idx:    make(map[M]int),
		values: make([]M, 0),
	}
}

// NewOrderedFrom returns a new *Ordered[M] filled with the values from the sequence.
func NewOrderedFrom[M cmp.Ordered](seq iter.Seq[M]) *Ordered[M] {
	s := NewOrdered[M]()
	for x := range seq {
		s.Add(x)
	}
	return s
}

// NewOrderedWith returns a new *Ordered[M] with the values provided.
func NewOrderedWith[M cmp.Ordered](m ...M) *Ordered[M] {
	return NewOrderedFrom(slices.Values(m))
}

// Contains returns true if the set contains the element.
func (s *Ordered[M]) Contains(m M) bool {
	_, ok := s.idx[m]
	return ok
}

// Clear the set and returns the number of elements removed.
func (s *Ordered[M]) Clear() int {
	n := len(s.values)
	if s.idx == nil {
		s.idx = make(map[M]int)
	} else {
		for k := range s.idx {
			delete(s.idx, k)
		}
	}
	if s.values == nil {
		s.values = make([]M, 0)
	} else {
		s.values = s.values[:0]
	}
	return n
}

// Add an element to the set. Returns true if the element was added, false if it was already present. Elements are added
// to the end of the ordered set.
func (s *Ordered[M]) Add(m M) bool {
	if s.Contains(m) {
		return false
	}
	s.values = append(s.values, m)
	s.idx[m] = len(s.values) - 1
	return true
}

// Remove an element from the set. Returns true if the element was removed, false if it was not present.
func (s *Ordered[M]) Remove(m M) bool {
	if !s.Contains(m) {
		return false
	}
	d := s.idx[m]
	s.values = append(s.values[:d], s.values[d+1:]...)
	for i, v := range s.values[d:] {
		s.idx[v] = d + i
	}
	delete(s.idx, m)
	return true
}

// Cardinality returns the number of elements in the set.
func (s *Ordered[M]) Cardinality() int {
	if s == nil {
		return 0
	}
	return len(s.values)
}

// Iterator yields all elements in the set in order.
func (s *Ordered[M]) Iterator(yield func(M) bool) {
	for _, k := range s.values {
		if !yield(k) {
			return
		}
	}
}

// Clone returns a copy of the set. The underlying type is the same as the original set.
func (s *Ordered[M]) Clone() Set[M] {
	return NewOrderedFrom(s.Iterator)
}

// Ordered iteration yields the index and value of each element in the set in order.
func (s *Ordered[M]) Ordered(yield func(int, M) bool) {
	for i, k := range s.values {
		if !yield(i, k) {
			return
		}
	}
}

// Backwards iteration yields the index and value of each element in the set in reverse order.
func (s *Ordered[M]) Backwards(yield func(int, M) bool) {
	for i := len(s.values) - 1; i >= 0; i-- {
		if !yield(i, s.values[i]) {
			return
		}
	}
}

// NewEmptyOrdered returns a new empty ordered set of the same underlying type.
func (s *Ordered[M]) NewEmptyOrdered() OrderedSet[M] {
	return NewOrdered[M]()
}

// NewEmpty returns a new empty set of the same underlying type.
func (s *Ordered[M]) NewEmpty() Set[M] {
	return NewOrdered[M]()
}

// Pop removes and returns an element from the set. If the set is empty, it returns the zero value of M and false.
func (s *Ordered[M]) Pop() (M, bool) {
	for k := range s.idx {
		s.Remove(k)
		return k, true
	}
	var m M
	return m, false
}

// Sort the set in ascending order.
func (s *Ordered[M]) Sort() {
	slices.Sort(s.values)
	for i, v := range s.values {
		s.idx[v] = i
	}
}

// At returns the element at the index. If the index is out of bounds, the second return value is false.
func (s *Ordered[M]) At(i int) (M, bool) {
	var zero M
	if i < 0 || i >= len(s.values) {
		return zero, false
	}
	return s.values[i], true
}

// Index returns the index of the element in the set, or -1 if not present.
func (s *Ordered[M]) Index(m M) int {
	i, ok := s.idx[m]
	if !ok {
		return -1
	}
	return i
}

// String returns a string representation of the set. It returns a string of the form OrderedSet[T](<elements>).
func (s *Ordered[M]) String() string {
	var m M
	return fmt.Sprintf("OrderedSet[%T](%v)", m, s.values)
}

// Value implements the driver.Valuer interface. It returns the JSON representation of the set.
func (s *Ordered[M]) Value() (driver.Value, error) {
	return s.MarshalJSON()
}

// MarshalJSON implements json.Marshaler. It will marshal the set into a JSON array of the elements in the set. If the
// set is empty an empty JSON array is returned.
func (s *Ordered[M]) MarshalJSON() ([]byte, error) {
	if len(s.values) == 0 {
		return []byte("[]"), nil
	}

	d, err := json.Marshal(s.values)
	if err != nil {
		return d, fmt.Errorf("marshaling ordered set: %w", err)
	}
	return d, nil
}

// UnmarshalJSON implements json.Unmarshaler. It expects a JSON array of the elements in the set. If the set is empty,
// it returns an empty set. If the JSON is invalid, it returns an error.
func (s *Ordered[M]) UnmarshalJSON(d []byte) error {
	t := make([]M, 0)
	if err := json.Unmarshal(d, &t); err != nil {
		return fmt.Errorf("unmarshaling ordered set: %w", err)
	}

	s.Clear()
	for _, v := range t {
		s.Add(v)
	}

	return nil
}

// Scan implements the sql.Scanner interface. It scans the value from the database into the set. It expects a JSON array
// of the elements in the set. If the JSON is invalid an error is returned. If the value is nil an empty set is
// returned.
func (s *Ordered[M]) Scan(src any) error {
	return scanValue[M](src, s.Clear, s.UnmarshalJSON)
}
