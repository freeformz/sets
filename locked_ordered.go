package sets

import (
	"iter"
	"sync"
	"sync/atomic"
)

// LockedOrdered is a set implementation that maintains the order of elements. If the same item is added multiple times,
// the first insertion determines the order.
type LockedOrdered[M comparable] struct {
	sync.RWMutex
	iteration atomic.Bool
	idx       map[M]int
	values    []M
}

var _ Set[int] = new(LockedOrdered[int])

// NewLockedOrdered returns an empty LockedOrderedSets instance.
func NewLockedOrdered[M comparable]() *LockedOrdered[M] {
	return &LockedOrdered[M]{
		idx:    make(map[M]int),
		values: make([]M, 0),
	}
}

// NewLockedOrderedFrom returns a new LockedOrderedSets instance filled with the values from the sequence.
func NewLockedOrderedFrom[M comparable](seq iter.Seq[M]) *LockedOrdered[M] {
	s := NewLockedOrdered[M]()
	for x := range seq {
		s.Add(x)
	}
	return s
}

func (s *LockedOrdered[M]) Contains(m M) bool {
	if !s.iteration.Load() {
		s.RLock()
		defer s.RUnlock()
	}
	return s.contains(m)
}

func (s *LockedOrdered[M]) Clear() int {
	if !s.iteration.Load() {
		s.Lock()
		defer s.Unlock()
	}
	n := len(s.values)
	for k := range s.idx {
		delete(s.idx, k)
	}
	s.values = s.values[:0]
	return n
}

func (s *LockedOrdered[M]) contains(m M) bool {
	_, ok := s.idx[m]
	return ok
}

func (s *LockedOrdered[M]) Add(m M) bool {
	if !s.iteration.Load() {
		s.Lock()
		defer s.Unlock()
	}
	if s.contains(m) {
		return false
	}
	s.values = append(s.values, m)
	s.idx[m] = len(s.values) - 1
	return true
}

func (s *LockedOrdered[M]) Remove(m M) bool {
	if !s.iteration.Load() {
		s.Lock()
		defer s.Unlock()
	}
	if !s.contains(m) {
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

func (s *LockedOrdered[M]) Cardinality() int {
	if !s.iteration.Load() {
		s.RLock()
		defer s.RUnlock()
	}
	return len(s.values)
}

func (s *LockedOrdered[M]) Iterator(yield func(M) bool) {
	for !s.iteration.CompareAndSwap(false, true) {
	}
	s.RLock()
	defer func() {
		s.iteration.Store(false)
		s.RUnlock()
	}()

	for _, k := range s.values {
		if !yield(k) {
			return
		}
	}
}

func (s *LockedOrdered[M]) Clone() Set[M] {
	return NewLockedOrderedFrom(s.Iterator)
}
