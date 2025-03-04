package set

import (
	"iter"
	"sync"
)

// LockedMapSet is a set implementation using a map and a read-write mutex. Instances of this type are safe to be used
// concurrently. Iteration holds the read lock for the duration of the iteration.
type LockedMapSet[M comparable] struct {
	mu sync.RWMutex
	m  map[M]struct{}
}

var _ Set[int] = new(LockedMapSet[int])

// NewLocked returns an empty LockedMapSet instance.
func NewLocked[M comparable]() *LockedMapSet[M] {
	return &LockedMapSet[M]{
		m: make(map[M]struct{}),
	}
}

// NewLockedFrom returns a new LockedMapSet instance filled with the values from the sequence.
func NewLockedFrom[M comparable](seq iter.Seq[M]) *LockedMapSet[M] {
	s := NewLocked[M]()
	for x := range seq {
		s.Add(x)
	}
	return s
}

func (s *LockedMapSet[M]) Contains(m M) bool {
	s.mu.RLock()
	_, ok := s.m[m]
	s.mu.RUnlock()
	return ok
}

func (s *LockedMapSet[M]) Add(m M) bool {
	if s.Contains(m) {
		return false
	}
	s.mu.Lock()
	s.m[m] = struct{}{}
	s.mu.Unlock()

	return true
}

func (s *LockedMapSet[M]) Remove(m M) bool {
	if !s.Contains(m) {
		return false
	}
	s.mu.Lock()
	delete(s.m, m)
	s.mu.Unlock()

	return true
}

func (s *LockedMapSet[M]) Cardinality() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.m)
}

func (s *LockedMapSet[M]) Iterator(yield func(M) bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for k := range s.m {
		if !yield(k) {
			return
		}
	}
}

func (s *LockedMapSet[M]) Clone() Set[M] {
	return NewLockedFrom(s.Iterator)
}
