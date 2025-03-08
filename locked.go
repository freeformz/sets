package sets

import (
	"iter"
	"sync"
	"sync/atomic"
)

// LockedMap is a set implementation using a map and a read-write mutex. Instances of this type are safe to be used
// concurrently. Iteration holds the read lock for the duration of the iteration.
type LockedMap[M comparable] struct {
	sync.RWMutex
	m         map[M]struct{}
	iterating atomic.Bool
}

var _ Set[int] = new(LockedMap[int])

// NewLocked returns an empty LockedMapSets instance.
func NewLocked[M comparable]() *LockedMap[M] {
	return &LockedMap[M]{
		m: make(map[M]struct{}),
	}
}

// NewLockedFrom returns a new LockedMapSets instance filled with the values from the sequence.
func NewLockedFrom[M comparable](seq iter.Seq[M]) *LockedMap[M] {
	s := NewLocked[M]()
	for x := range seq {
		s.Add(x)
	}
	return s
}

func (s *LockedMap[M]) Contains(m M) bool {
	if !s.iterating.Load() {
		s.RLock()
		defer s.RUnlock()
	}

	return s.contains(m)
}

func (s *LockedMap[M]) Clear() int {
	if !s.iterating.Load() {
		s.Lock()
		defer s.Unlock()
	}
	n := len(s.m)
	s.m = make(map[M]struct{})
	return n
}

func (s *LockedMap[M]) contains(m M) bool {
	_, ok := s.m[m]
	return ok
}

func (s *LockedMap[M]) Add(m M) bool {
	if !s.iterating.Load() {
		s.Lock()
		defer s.Unlock()
	}

	if s.contains(m) {
		return false
	}
	s.m[m] = struct{}{}

	return true
}

func (s *LockedMap[M]) Remove(m M) bool {
	if !s.iterating.Load() {
		s.Lock()
		defer s.Unlock()
	}

	if !s.contains(m) {
		return false
	}
	delete(s.m, m)
	return true
}

func (s *LockedMap[M]) Cardinality() int {
	if !s.iterating.Load() {
		s.RLock()
		defer s.RUnlock()
	}
	return len(s.m)
}

func (s *LockedMap[M]) Iterator(yield func(M) bool) {
	for !s.iterating.CompareAndSwap(false, true) {
	}
	s.RLock()
	defer func() {
		s.iterating.Store(false)
		s.RUnlock()
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
