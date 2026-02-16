package sets

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"iter"
	"slices"
	"sync"
)

// Locked is a concurrency safe wrapper around a Set[M]. It uses a read-write lock to allow multiple readers to access
// the set concurrently, but only one writer at a time. The set is not ordered and does not guarantee the order of
// elements when iterating over them. It is safe for concurrent use.
type Locked[M comparable] struct {
	set Set[M]
	sync.RWMutex
}

var _ Set[int] = new(Locked[int])
var _ driver.Valuer = new(Locked[int])

// NewLocked returns an empty *Locked[M] that is safe for concurrent use.
func NewLocked[M comparable]() *Locked[M] {
	return &Locked[M]{set: New[M]()}
}

// NewLockedFrom returns a new *Locked[M] filled with the values from the sequence.
func NewLockedFrom[M comparable](seq iter.Seq[M]) *Locked[M] {
	s := NewLocked[M]()
	for x := range seq {
		s.Add(x)
	}
	return s
}

// NewLockedWith returns a *Locked[M] with the values provided.
func NewLockedWith[M comparable](m ...M) *Locked[M] {
	return NewLockedFrom(slices.Values(m))
}

// NewLockedWrapping returns a Set[M]. If set is already a locked set, then it is just returned as is. If set isn't a locked set
// then the returned set is wrapped so that it is safe for concurrent use.
func NewLockedWrapping[M comparable](set Set[M]) Set[M] {
	if _, ok := set.(Locker); ok {
		return set
	}

	lset := NewLocked[M]()
	lset.set = set

	return lset
}

// Contains returns true if the set contains the element.
func (s *Locked[M]) Contains(m M) bool {
	s.RLock()
	defer s.RUnlock()
	return s.set.Contains(m)
}

// Clear the set and returns the number of elements removed.
func (s *Locked[M]) Clear() int {
	s.Lock()
	defer s.Unlock()
	return s.set.Clear()
}

// Add an element to the set. Returns true if the element was added, false if it was already present.
func (s *Locked[M]) Add(m M) bool {
	s.Lock()
	defer s.Unlock()

	return s.set.Add(m)
}

// Remove an element from the set. Returns true if the element was removed, false if it was not present.
func (s *Locked[M]) Remove(m M) bool {
	s.Lock()
	defer s.Unlock()

	return s.set.Remove(m)
}

// Cardinality returns the number of elements in the set.
func (s *Locked[M]) Cardinality() int {
	s.RLock()
	defer s.RUnlock()

	return s.set.Cardinality()
}

// Iterator yields all elements in the set. It holds a read lock for the duration of iteration. Calling any method that
// modifies the set while iteration is happening will block until the iteration is complete.
func (s *Locked[M]) Iterator(yield func(M) bool) {
	s.RLock()
	defer s.RUnlock()

	s.set.Iterator(yield)
}

// Clone returns a new set of the same underlying type.
func (s *Locked[M]) Clone() Set[M] {
	return NewLockedFrom(s.Iterator)
}

// NewEmpty returns a new empty set of the same underlying type.
func (s *Locked[M]) NewEmpty() Set[M] {
	return NewLocked[M]()
}

// Pop removes and returns an element from the set. If the set is empty, it returns the zero value of M and false.
func (s *Locked[M]) Pop() (M, bool) {
	s.Lock()
	defer s.Unlock()

	return s.set.Pop()
}

// String returns a string representation of the set. It returns a string of the form LockedSet[T](<elements>).
func (s *Locked[M]) String() string {
	s.RLock()
	defer s.RUnlock()
	return "Locked" + s.set.String()
}

// Value implements the driver.Valuer interface. It returns the JSON representation of the set.
func (s *Locked[M]) Value() (driver.Value, error) {
	return s.MarshalJSON()
}

// MarshalJSON implements json.Marshaler. It will marshal the set into a JSON array of the elements in the set. If the
// set is empty an empty JSON array is returned.
func (s *Locked[M]) MarshalJSON() ([]byte, error) {
	s.RLock()
	defer s.RUnlock()

	jm, ok := s.set.(json.Marshaler)
	if !ok {
		return nil, fmt.Errorf("cannot marshal set of type %T - not json.Marshaler", s.set)
	}

	d, err := jm.MarshalJSON()
	if err != nil {
		return d, fmt.Errorf("marshaling locked set: %w", err)
	}
	return d, nil
}

// UnmarshalJSON implements json.Unmarshaler. It expects a JSON array of the elements in the set. If the set is empty,
// it returns an empty set. If the JSON is invalid, it returns an error.
func (s *Locked[M]) UnmarshalJSON(d []byte) error {
	s.Lock()
	defer s.Unlock()

	if s.set == nil {
		s.set = New[M]()
	}
	um, ok := s.set.(json.Unmarshaler)
	if !ok {
		return fmt.Errorf("cannot unmarshal set of type %T - not json.Unmarshaler", s.set)
	}

	if err := um.UnmarshalJSON(d); err != nil {
		return fmt.Errorf("unmarshaling locked set: %w", err)
	}

	return nil
}

// Scan implements the sql.Scanner interface. It scans the value from the database into the set. It expects a JSON array
// of the elements in the set. If the JSON is invalid an error is returned. If the value is nil an empty set is
// returned.
func (s *Locked[M]) Scan(src any) error {
	s.Lock()
	defer s.Unlock()

	if s.set == nil {
		s.set = New[M]()
	}

	return scanValue[M](src, s.set.Clear, func(data []byte) error {
		um, ok := s.set.(json.Unmarshaler)
		if !ok {
			return fmt.Errorf("cannot unmarshal set of type %T - not json.Unmarshaler", s.set)
		}
		return um.UnmarshalJSON(data)
	})
}
