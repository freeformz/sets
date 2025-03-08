package set

import (
	"iter"
	"sync"
)

// LockedMap is a set implementation using a map and a mutex (via the sync.Cond). Instances of this type are safe to be used
// concurrently. Iteration holds the lock for the duration of the iteration.
type LockedMap[M comparable] struct {
	*sync.RWMutex
	*sync.Cond
	iterating bool
	m         map[M]struct{}
}

var _ Set[int] = new(LockedMap[int])

// NewLocked returns an empty LockedMapSet instance.
func NewLocked[M comparable]() *LockedMap[M] {
	mu := &sync.RWMutex{}
	return &LockedMap[M]{
		m:       make(map[M]struct{}),
		RWMutex: mu,
		Cond:    sync.NewCond(mu),
	}
}

// NewLockedFrom returns a new LockedMapSet instance filled with the values from the sequence.
func NewLockedFrom[M comparable](seq iter.Seq[M]) *LockedMap[M] {
	s := NewLocked[M]()
	for x := range seq {
		s.Add(x)
	}
	return s
}

func (s *LockedMap[M]) Contains(m M) bool {
	s.RWMutex.RLock()
	defer s.RWMutex.RUnlock()
	return s.contains(m)
}

func (s *LockedMap[M]) Clear() int {
	s.Cond.L.Lock()
	if s.iterating {
		s.Cond.Wait()
	}
	defer s.Cond.L.Unlock()

	n := len(s.m)
	for k := range s.m {
		delete(s.m, k)
	}
	return n
}

func (s *LockedMap[M]) contains(m M) bool {
	_, ok := s.m[m]
	return ok
}

func (s *LockedMap[M]) Add(m M) bool {
	s.Cond.L.Lock()
	if s.iterating {
		s.Cond.Wait()
	}
	defer s.Cond.L.Unlock()

	if s.contains(m) {
		return false
	}
	s.m[m] = struct{}{}

	return true
}

func (s *LockedMap[M]) Remove(m M) bool {
	s.Cond.L.Lock()
	if s.iterating {
		s.Cond.Wait()
	}
	defer s.Cond.L.Unlock()

	if !s.contains(m) {
		return false
	}
	delete(s.m, m)
	return true
}

func (s *LockedMap[M]) Cardinality() int {
	s.RWMutex.RLock()
	defer s.RWMutex.RUnlock()

	return len(s.m)
}

// Iterator yields all elements in the set. It holds a lock for the duration of iteration. Calling methods other than
// `Contains` and `Cardinality` will block until the iteration is complete.
func (s *LockedMap[M]) Iterator(yield func(M) bool) {
	s.Cond.L.Lock()
	s.iterating = true
	defer func() {
		s.iterating = false
		s.Cond.Broadcast()
		s.Cond.L.Unlock()
	}()
	for k := range s.m {
		if !yield(k) {
			return
		}
	}
}

func (s *LockedMap[M]) Clone() Set[M] {
	return NewLockedFrom(s.Iterator)
}
