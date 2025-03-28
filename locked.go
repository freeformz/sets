package sets

import (
	"encoding/json"
	"fmt"
	"iter"
	"slices"
	"sync"
)

type Locked[M comparable] struct {
	set Set[M]
	sync.RWMutex
	*sync.Cond
	iterating bool
}

var _ Set[int] = new(Locked[int])

// NewLocked returns an empty Set[M] that is safe for concurrent use.
func NewLocked[M comparable]() *Locked[M] {
	l := &Locked[M]{set: NewMap[M]()}
	l.Cond = sync.NewCond(&l.RWMutex)
	return l
}

// NewLockedFrom returns a new Set[M] filled with the values from the sequence.
func NewLockedFrom[M comparable](seq iter.Seq[M]) *Locked[M] {
	s := NewLocked[M]()
	for x := range seq {
		s.Add(x)
	}
	return s
}

// NewLockedWith the values provides. Duplicates are removed.
func NewLockedWith[M comparable](m ...M) *Locked[M] {
	return NewLockedFrom(slices.Values(m))
}

// NewLockedWrapping returns a Set[M]. If set is already a locked set, then it is just returned as is. If set isn't a locked set
// then the returned set is wrapped so that it is safe for concurrent use.
func NewLockedWrapping[M comparable](set Set[M]) Set[M] {
	if _, ok := set.(locker); ok {
		return set
	}

	lset := NewLocked[M]()
	lset.set = set

	return lset
}

func (s *Locked[M]) Contains(m M) bool {
	s.RWMutex.RLock()
	defer s.RWMutex.RUnlock()
	return s.set.Contains(m)
}

func (s *Locked[M]) Clear() int {
	s.Cond.L.Lock()
	if s.iterating {
		s.Cond.Wait()
	}
	defer s.Cond.L.Unlock()
	return s.set.Clear()
}

func (s *Locked[M]) Add(m M) bool {
	s.Cond.L.Lock()
	if s.iterating {
		s.Cond.Wait()
	}
	defer s.Cond.L.Unlock()

	return s.set.Add(m)
}

func (s *Locked[M]) Remove(m M) bool {
	s.Cond.L.Lock()
	if s.iterating {
		s.Cond.Wait()
	}
	defer s.Cond.L.Unlock()

	return s.set.Remove(m)
}

func (s *Locked[M]) Cardinality() int {
	if s == nil {
		return 0
	}
	s.RWMutex.RLock()
	defer s.RWMutex.RUnlock()

	return s.set.Cardinality()
}

// Iterator yields all elements in the set. It holds a lock for the duration of iteration. Calling methods other than
// `Contains` and `Cardinality` will block until the iteration is complete.
func (s *Locked[M]) Iterator(yield func(M) bool) {
	s.Cond.L.Lock()
	s.iterating = true
	defer func() {
		s.iterating = false
		s.Cond.Broadcast()
		s.Cond.L.Unlock()
	}()

	s.set.Iterator(yield)
}

func (s *Locked[M]) Clone() Set[M] {
	return NewLockedFrom(s.Iterator)
}

func (s *Locked[M]) NewEmpty() Set[M] {
	return NewLocked[M]()
}

func (s *Locked[M]) Pop() (M, bool) {
	s.L.Lock()
	if s.iterating {
		s.Wait()
	}
	defer s.L.Unlock()

	return s.set.Pop()
}

func (s *Locked[M]) String() string {
	s.RLock()
	defer s.RUnlock()
	return "Locked" + s.set.String()
}

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

// UnmarshalJSON implements json.Unmarshaler. It will unmarshal the JSON data into the set.
func (s *Locked[M]) UnmarshalJSON(d []byte) error {
	s.Lock()
	if s.Cond == nil {
		s.Cond = sync.NewCond(&s.RWMutex)
	}
	if s.iterating {
		s.Wait()
	}
	defer s.Unlock()

	if s.set == nil {
		s.set = NewMap[M]()
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
