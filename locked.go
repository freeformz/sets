package set

import (
	"iter"
	"sync"
)

type lockedMap[M comparable] struct {
	*sync.RWMutex
	*sync.Cond
	iterating bool
	set       Set[M]
}

var _ Set[int] = new(lockedMap[int])

// NewLocked returns an empty Set[M] that is safe for concurrent use.
func NewLocked[M comparable]() Set[M] {
	mu := &sync.RWMutex{}
	return &lockedMap[M]{
		set:     New[M](),
		RWMutex: mu,
		Cond:    sync.NewCond(mu),
	}
}

// NewLockedFrom returns a new Set[M] filled with the values from the sequence.
func NewLockedFrom[M comparable](seq iter.Seq[M]) Set[M] {
	s := NewLocked[M]()
	for x := range seq {
		s.Add(x)
	}
	return s
}

type locker interface {
	Lock()
	Unlock()
	RLock()
	RUnlock()
	Wait()
	Broadcast()
}

// NewLockedWith returns a Set[M]. If set is already a locked set, then it is just returned as is. If set isn't a locked set
// then the returned set is wrapped so that it is safe for concurrent use.
func NewLockedWith[M comparable](set Set[M]) Set[M] {
	if _, lok := set.(locker); lok {
		return set
	}
	mu := &sync.RWMutex{}
	return &lockedMap[M]{
		set:     set,
		RWMutex: mu,
		Cond:    sync.NewCond(mu),
	}
}

func (s *lockedMap[M]) Contains(m M) bool {
	s.RWMutex.RLock()
	defer s.RWMutex.RUnlock()
	return s.set.Contains(m)
}

func (s *lockedMap[M]) Clear() int {
	s.Cond.L.Lock()
	if s.iterating {
		s.Cond.Wait()
	}
	defer s.Cond.L.Unlock()
	return s.set.Clear()
}

func (s *lockedMap[M]) Add(m M) bool {
	s.Cond.L.Lock()
	if s.iterating {
		s.Cond.Wait()
	}
	defer s.Cond.L.Unlock()

	return s.set.Add(m)
}

func (s *lockedMap[M]) Remove(m M) bool {
	s.Cond.L.Lock()
	if s.iterating {
		s.Cond.Wait()
	}
	defer s.Cond.L.Unlock()

	return s.set.Remove(m)
}

func (s *lockedMap[M]) Cardinality() int {
	if s == nil {
		return 0
	}
	s.RWMutex.RLock()
	defer s.RWMutex.RUnlock()

	return s.set.Cardinality()
}

// Iterator yields all elements in the set. It holds a lock for the duration of iteration. Calling methods other than
// `Contains` and `Cardinality` will block until the iteration is complete.
func (s *lockedMap[M]) Iterator(yield func(M) bool) {
	s.Cond.L.Lock()
	s.iterating = true
	defer func() {
		s.iterating = false
		s.Cond.Broadcast()
		s.Cond.L.Unlock()
	}()

	s.set.Iterator(yield)
}

func (s *lockedMap[M]) Clone() Set[M] {
	return NewLockedFrom(s.Iterator)
}

func (s *lockedMap[M]) NewEmpty() Set[M] {
	return NewLocked[M]()
}

func (s *lockedMap[M]) String() string {
	s.RLock()
	defer s.RUnlock()
	return "Locked" + s.set.String()
}
