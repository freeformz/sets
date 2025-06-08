package sets

import (
	"encoding/json"
	"fmt"
	"iter"
	"maps"
	"slices"
)

// Map is the default set implementation based on top of go's map type. It is not ordered and does not guarantee
// the order of elements when iterating over them. It is not safe for concurrent use.
type Map[M comparable] struct {
	set map[M]struct{}
}

var _ Set[int] = new(Map[int])

// New returns an empty *Map[M] instance.
func New[M comparable]() *Map[M] {
	return &Map[M]{
		set: make(map[M]struct{}),
	}
}

// NewFrom returns a new *Map[M] filled with the values from the sequence.
func NewFrom[M comparable](seq iter.Seq[M]) *Map[M] {
	s := New[M]()
	for x := range seq {
		s.Add(x)
	}
	return s
}

// NewWith returns a new *Map[M] with the values provided.
func NewWith[M comparable](m ...M) *Map[M] {
	return NewFrom(slices.Values(m))
}

// Contains returns true if the set contains the element.
func (s *Map[M]) Contains(m M) bool {
	_, ok := s.set[m]
	return ok
}

// Clear the set and returns the number of elements removed.
func (s *Map[M]) Clear() int {
	if s.set == nil {
		s.set = make(map[M]struct{})
	}
	n := len(s.set)
	for k := range s.set {
		delete(s.set, k)
	}
	return n
}

// Add an element to the set. Returns true if the element was added, false if it was already present.
func (s *Map[M]) Add(m M) bool {
	if s.Contains(m) {
		return false
	}
	s.set[m] = struct{}{}
	return true
}

// Remove an element from the set. Returns true if the element was removed, false if it was not present.
func (s *Map[M]) Remove(m M) bool {
	if !s.Contains(m) {
		return false
	}
	delete(s.set, m)
	return true
}

// Cardinality returns the number of elements in the set.
func (s *Map[M]) Cardinality() int {
	return len(s.set)
}

// Iterator yields all elements in the set.
func (s *Map[M]) Iterator(yield func(M) bool) {
	for k := range s.set {
		if !yield(k) {
			return
		}
	}
}

// Clones the set. Returns a new set of the same underlying type.
func (s *Map[M]) Clone() Set[M] {
	return NewFrom(s.Iterator)
}

// NewEmpty set of the same underlying type.
func (s *Map[M]) NewEmpty() Set[M] {
	return New[M]()
}

// Pop removes and returns an element from the set. If the set is empty, it returns the zero value of M and false.
func (s *Map[M]) Pop() (M, bool) {
	for k := range s.set {
		delete(s.set, k)
		return k, true
	}
	var m M
	return m, false
}

// String representation of the set. It returns a string of the form Set[T](<elements>).
func (s *Map[M]) String() string {
	var m M
	return fmt.Sprintf("Set[%T](%v)", m, slices.Collect(maps.Keys(s.set)))
}

// MarshalJSON marshals the set to JSON. It returns a JSON array of the elements in the set. If the set is empty, it
// returns an empty JSON array.
func (s *Map[M]) MarshalJSON() ([]byte, error) {
	v := slices.Collect(s.Iterator)
	if len(v) == 0 {
		return []byte("[]"), nil
	}

	d, err := json.Marshal(v)
	if err != nil {
		return d, fmt.Errorf("marshaling map set: %w", err)
	}
	return d, nil
}

// UnmarshalJSON unmarshals the set from JSON. It expects a JSON array of the elements in the set. If the set is empty,
// it returns an empty set. If the JSON is invalid, it returns an error.
func (s *Map[M]) UnmarshalJSON(d []byte) error {
	var um []M
	if err := json.Unmarshal(d, &um); err != nil {
		return fmt.Errorf("unmarshaling map set: %w", err)
	}

	s.Clear()
	for _, m := range um {
		s.Add(m)
	}

	return nil
}

// scanValue is a helper function that implements the common logic for scanning values into sets.
// It handles nil, []byte, and string types, delegating to the provided unmarshal function.
func scanValue[M comparable](src any, clear func() int, unmarshal func([]byte) error) error {
	switch st := src.(type) {
	case nil:
		clear()
		return nil
	case []byte:
		return unmarshal(st)
	case string:
		return unmarshal([]byte(st))
	default:
		return fmt.Errorf("cannot scan set of type %T - not []byte or string", st)
	}
}

// Scan implements the sql.Scanner interface. It scans the value from the database into the set. It expects a JSON array
// of the elements in the set. If the JSON is invalid an error is returned. If the value is nil an empty set is
// returned.
func (s *Map[M]) Scan(src any) error {
	return scanValue[M](src, s.Clear, s.UnmarshalJSON)
}
