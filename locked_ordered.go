package sets

import (
	"cmp"
	"encoding/json"
	"fmt"
	"iter"
	"slices"
	"sync"
)

// LockedOrdered is a concurrency safe wrapper around an OrderedSet[M]. It uses a read-write lock to allow multiple readers.
type LockedOrdered[M cmp.Ordered] struct {
	set OrderedSet[M]
	sync.RWMutex
}

var _ Set[int] = new(LockedOrdered[int])

// NewLockedOrdered returns an empty *LockedOrdered[M] instance that is safe for concurrent use.
func NewLockedOrdered[M cmp.Ordered]() *LockedOrdered[M] {
	return &LockedOrdered[M]{set: NewOrdered[M]()}
}

// NewLockedOrderedFrom returns a new *LockedOrdered[M] instance filled with the values from the sequence. The set is safe
// for concurrent use.
func NewLockedOrderedFrom[M cmp.Ordered](seq iter.Seq[M]) *LockedOrdered[M] {
	s := NewLockedOrdered[M]()
	for x := range seq {
		s.Add(x)
	}
	return s
}

// NewLockedOrderedWith returns a *LockedOrdered[M] with the values provided.
func NewLockedOrderedWith[M cmp.Ordered](m ...M) *LockedOrdered[M] {
	return NewLockedOrderedFrom(slices.Values(m))
}

// NewLockedOrderedWrapping returns an OrderedSet[M]. If the set is already a locked set, then it is just returned as
// is. If the set isn't a locked set then the returned set is wrapped so that it is safe for concurrent use.
func NewLockedOrderedWrapping[M cmp.Ordered](set OrderedSet[M]) OrderedSet[M] {
	if _, ok := set.(Locker); ok {
		return set
	}
	lset := NewLockedOrdered[M]()
	lset.set = set
	return lset
}

// Contains returns true if the set contains the element.
func (s *LockedOrdered[M]) Contains(m M) bool {
	s.RLock()
	defer s.RUnlock()
	return s.set.Contains(m)
}

// Clear the set and returns the number of elements removed.
func (s *LockedOrdered[M]) Clear() int {
	s.Lock()
	defer s.Unlock()
	return s.set.Clear()
}

// Add an element to the set. Returns true if the element was added, false if it was already present.
func (s *LockedOrdered[M]) Add(m M) bool {
	s.Lock()
	defer s.Unlock()
	return s.set.Add(m)
}

// Remove an element from the set. Returns true if the element was removed, false if it was not present.
func (s *LockedOrdered[M]) Remove(m M) bool {
	s.Lock()
	defer s.Unlock()
	return s.set.Remove(m)
}

// Cardinality returns the number of elements in the set.
func (s *LockedOrdered[M]) Cardinality() int {
	s.RLock()
	defer s.RUnlock()

	return s.set.Cardinality()
}

// Iterator yields all elements in the set in order. It holds a read lock for the duration of iteration. Calling any
// method that modifies the set while iteration is happening will block until the iteration is complete.
func (s *LockedOrdered[M]) Iterator(yield func(M) bool) {
	s.RLock()
	defer s.RUnlock()

	s.set.Iterator(yield)
}

// Clone returns a new set of the same underlying type.
func (s *LockedOrdered[M]) Clone() Set[M] {
	s.RLock()
	defer s.RUnlock()
	return NewLockedOrderedFrom(s.Iterator)
}

// Ordered iteration yields the index and value of each element in the set in order. It holds a read lock for the
// duration of iteration. Calling any method that modifies the set while iteration is happening will block until the
// iteration is complete.
func (s *LockedOrdered[M]) Ordered(yield func(int, M) bool) {
	s.RLock()
	defer s.RUnlock()

	s.set.Ordered(yield)
}

// Backwards iteration yields the index and value of each element in the set in reverse order. It holds a read lock for
// the duration of iteration. Calling any method that modifies the set while iteration is happening will block until the
// iteration is complete.
func (s *LockedOrdered[M]) Backwards(yield func(int, M) bool) {
	s.RLock()
	defer s.RUnlock()

	s.set.Backwards(yield)
}

// NewEmptyOrdered returns a new empty ordered set of the same underlying type.
func (s *LockedOrdered[M]) NewEmptyOrdered() OrderedSet[M] {
	return NewLockedOrdered[M]()
}

// NewEmpty returns a new empty set of the same underlying type.
func (s *LockedOrdered[M]) NewEmpty() Set[M] {
	return NewLockedOrdered[M]()
}

// Pop removes and returns an element from the set. If the set is empty, it returns the zero value of M and false.
func (s *LockedOrdered[M]) Pop() (M, bool) {
	s.Lock()
	defer s.Unlock()

	return s.set.Pop()
}

// Sort the set in ascending order.
func (s *LockedOrdered[M]) Sort() {
	s.Lock()
	defer s.Unlock()

	s.set.Sort()
}

// At returns the element at the index. If the index is out of bounds, the second return value is false.
func (s *LockedOrdered[M]) At(i int) (M, bool) {
	s.RLock()
	defer s.RUnlock()

	return s.set.At(i)
}

// Index returns the index of the element in the set, or -1 if not present.
func (s *LockedOrdered[M]) Index(m M) int {
	s.RLock()
	defer s.RUnlock()

	return s.set.Index(m)
}

// String returns a string representation of the set. It returns a string of the form LockedOrderedSet[T](<elements>).
func (s *LockedOrdered[M]) String() string {
	s.RLock()
	defer s.RUnlock()

	return "Locked" + s.set.String()
}

// MarshalJSON implements json.Marshaler. It will marshal the set to JSON. It returns a JSON array of the elements in
// the set. If the set is empty, it returns an empty JSON array.
func (s *LockedOrdered[M]) MarshalJSON() ([]byte, error) {
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
func (s *LockedOrdered[M]) UnmarshalJSON(d []byte) error {
	s.Lock()
	defer s.Unlock()

	if s.set == nil {
		s.set = NewOrdered[M]()
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
