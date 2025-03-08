package set

import (
	"iter"
	"sync"
)

// SyncMap is a set implementation using the built in sync.Map type. Instances of this type are safe to be used
// concurrently. All sync.Map characteristics apply. Iteration is done via sync.Map.Range.
type SyncMap[M comparable] struct {
	m sync.Map
}

var _ Set[int] = new(SyncMap[int])

// NewSync returns an empty SyncMapSet instance.
func NewSync[M comparable]() *SyncMap[M] {
	return &SyncMap[M]{
		m: sync.Map{},
	}
}

// NewSyncFrom returns a new SyncMapSet instance filled with the values from the sequence.
func NewSyncFrom[M comparable](seq iter.Seq[M]) *SyncMap[M] {
	s := NewSync[M]()
	for x := range seq {
		s.Add(x)
	}
	return s
}

func (s *SyncMap[M]) Contains(m M) bool {
	_, ok := s.m.Load(m)
	return ok
}

func (s *SyncMap[M]) Clear() int {
	var n int
	s.m.Range(func(_, _ interface{}) bool {
		n++
		return true
	})
	s.m.Clear()
	return n
}

func (s *SyncMap[M]) Add(m M) bool {
	_, loaded := s.m.LoadOrStore(m, struct{}{})
	return !loaded
}

func (s *SyncMap[M]) Remove(m M) bool {
	_, ok := s.m.LoadAndDelete(m)
	return ok
}

func (s *SyncMap[M]) Cardinality() int {
	var n int
	s.m.Range(func(_, _ interface{}) bool {
		n++
		return true
	})
	return n
}

func (s *SyncMap[M]) Iterator(yield func(M) bool) {
	s.m.Range(func(key, _ interface{}) bool {
		return yield(key.(M))
	})
}

func (s *SyncMap[M]) Clone() Set[M] {
	return NewSyncFrom(s.Iterator)
}
