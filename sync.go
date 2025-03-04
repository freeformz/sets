package set

import (
	"iter"
	"sync"
)

// SyncMapSet is a set implementation using the built in sync.Map type. Instances of this type are safe to be used
// concurrently. All sync.Map characteristics apply. Iteration is done via sync.Map.Range.
type SyncMapSet[M comparable] struct {
	m sync.Map
}

var _ Set[int] = new(SyncMapSet[int])

// NewSync returns an empty SyncMapSet instance.
func NewSync[M comparable]() *SyncMapSet[M] {
	return &SyncMapSet[M]{
		m: sync.Map{},
	}
}

// NewSyncFrom returns a new SyncMapSet instance filled with the values from the sequence.
func NewSyncFrom[M comparable](seq iter.Seq[M]) *SyncMapSet[M] {
	s := NewSync[M]()
	for x := range seq {
		s.Add(x)
	}
	return s
}

func (s *SyncMapSet[M]) Contains(m M) bool {
	_, ok := s.m.Load(m)
	return ok
}

func (s *SyncMapSet[M]) Add(m M) bool {
	_, loaded := s.m.LoadOrStore(m, struct{}{})
	return !loaded
}

func (s *SyncMapSet[M]) Remove(m M) bool {
	_, ok := s.m.LoadAndDelete(m)
	return ok
}

func (s *SyncMapSet[M]) Cardinality() int {
	var n int
	s.m.Range(func(_, _ interface{}) bool {
		n++
		return true
	})
	return n
}

func (s *SyncMapSet[M]) Iterator(yield func(M) bool) {
	s.m.Range(func(key, _ interface{}) bool {
		return yield(key.(M))
	})
}

func (s *SyncMapSet[M]) Clone() Set[M] {
	return NewSyncFrom(s.Iterator)
}
