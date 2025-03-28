package sets

import (
	"cmp"
	"encoding/json"
	"fmt"
	"iter"
	"slices"
	"sync"
)

type LockedOrdered[M cmp.Ordered] struct {
	set OrderedSet[M]
	sync.RWMutex
	*sync.Cond
	iterating bool
}

var _ Set[int] = new(LockedOrdered[int])

// NewLockedOrdered returns an empty OrderedSet[M] instance that is safe for concurrent use.
func NewLockedOrdered[M cmp.Ordered]() *LockedOrdered[M] {
	set := &LockedOrdered[M]{set: NewOrdered[M]()}
	set.Cond = sync.NewCond(&set.RWMutex)
	return set
}

// NewLockedOrderedFrom returns a new OrderedSet[M] instance filled with the values from the sequence. The set is safe
// for concurrent use.
func NewLockedOrderedFrom[M cmp.Ordered](seq iter.Seq[M]) *LockedOrdered[M] {
	s := NewLockedOrdered[M]()
	for x := range seq {
		s.Add(x)
	}
	return s
}

// NewOrderedWith the values provides. Duplicates are removed.
func NewLockedOrderedWith[M cmp.Ordered](m ...M) *LockedOrdered[M] {
	return NewLockedOrderedFrom(slices.Values(m))
}

// NewLockedOrderedWrapping returns an OrderedSet[M]. If set is already a locked set, then it is just returned as is. If set isn't a locked set
// then the returned set is wrapped so that it is safe for concurrent use.
func NewLockedOrderedWrapping[M cmp.Ordered](set OrderedSet[M]) OrderedSet[M] {
	if lset, ok := set.(*LockedOrdered[M]); ok {
		return lset
	}
	lset := NewLockedOrdered[M]()
	lset.set = set
	return lset
}

func (s *LockedOrdered[M]) Contains(m M) bool {
	s.RLock()
	defer s.RUnlock()
	return s.set.Contains(m)
}

func (s *LockedOrdered[M]) Clear() int {
	s.L.Lock()
	if s.iterating {
		s.Wait()
	}
	defer s.L.Unlock()
	return s.set.Clear()
}

func (s *LockedOrdered[M]) Add(m M) bool {
	s.L.Lock()
	if s.iterating {
		s.Wait()
	}
	defer s.L.Unlock()
	return s.set.Add(m)
}

func (s *LockedOrdered[M]) Remove(m M) bool {
	s.L.Lock()
	if s.iterating {
		s.Wait()
	}
	defer s.L.Unlock()
	return s.set.Remove(m)
}

func (s *LockedOrdered[M]) Cardinality() int {
	if s == nil {
		return 0
	}
	s.RLock()
	defer s.RUnlock()
	return s.set.Cardinality()
}

// Iterator yields all elements in the set in order. It holds a lock for the duration of iteration. Calling methods other than
// `Contains` and `Cardinality` will block until the iteration is complete.
func (s *LockedOrdered[M]) Iterator(yield func(M) bool) {
	s.L.Lock()
	s.iterating = true
	defer func() {
		s.iterating = false
		s.Broadcast()
		s.L.Unlock()
	}()

	s.set.Iterator(yield)
}

func (s *LockedOrdered[M]) Clone() Set[M] {
	return NewLockedOrderedFrom(s.Iterator)
}

// Ordered iteration yields the index and value of each element in the set in order.
func (s *LockedOrdered[M]) Ordered(yield func(int, M) bool) {
	s.L.Lock()
	s.iterating = true
	defer func() {
		s.iterating = false
		s.Broadcast()
		s.L.Unlock()
	}()

	s.set.Ordered(yield)
}

func (s *LockedOrdered[M]) Backwards(yield func(int, M) bool) {
	s.L.Lock()
	s.iterating = true
	defer func() {
		s.iterating = false
		s.Broadcast()
		s.L.Unlock()
	}()

	s.set.Backwards(yield)
}

func (s *LockedOrdered[M]) NewEmptyOrdered() OrderedSet[M] {
	return NewLockedOrdered[M]()
}

func (s *LockedOrdered[M]) NewEmpty() Set[M] {
	return NewLockedOrdered[M]()
}

func (s *LockedOrdered[M]) Pop() (M, bool) {
	s.L.Lock()
	if s.iterating {
		s.Wait()
	}
	defer s.L.Unlock()

	return s.set.Pop()
}

func (s *LockedOrdered[M]) Sort() {
	s.L.Lock()
	if s.iterating {
		s.Wait()
	}
	defer s.L.Unlock()

	s.set.Sort()
}

func (s *LockedOrdered[M]) At(i int) (M, bool) {
	s.RLock()
	defer s.RUnlock()

	return s.set.At(i)
}

func (s *LockedOrdered[M]) Index(m M) int {
	s.RLock()
	defer s.RUnlock()

	return s.set.Index(m)
}

func (s *LockedOrdered[M]) String() string {
	s.RLock()
	defer s.RUnlock()

	return "Locked" + s.set.String()
}

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

// UnmarshalJSON implements json.Unmarshaler. It will unmarshal the JSON data into the set.
func (s *LockedOrdered[M]) UnmarshalJSON(d []byte) error {
	s.Lock()
	if s.Cond == nil {
		s.Cond = sync.NewCond(&s.RWMutex)
	}
	if s.iterating {
		s.Wait()
	}
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
