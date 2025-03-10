package set

import (
	"cmp"
	"iter"
	"sync"
)

type lockedOrdered[M cmp.Ordered] struct {
	set OrderedSet[M]
	*sync.RWMutex
	*sync.Cond
	iterating bool
}

var _ Set[int] = new(lockedOrdered[int])

// NewLockedOrdered returns an empty OrderedSet[M] instance that is safe for concurrent use.
func NewLockedOrdered[M cmp.Ordered]() OrderedSet[M] {
	mu := &sync.RWMutex{}
	return &lockedOrdered[M]{
		set:     NewOrdered[M](),
		RWMutex: mu,
		Cond:    sync.NewCond(mu),
	}
}

// NewLockedOrderedFrom returns a new OrderedSet[M] instance filled with the values from the sequence. The set is safe
// for concurrent use.
func NewLockedOrderedFrom[M cmp.Ordered](seq iter.Seq[M]) OrderedSet[M] {
	s := NewLockedOrdered[M]()
	for x := range seq {
		s.Add(x)
	}
	return s
}

// NewLockedOrderedWith returns an OrderedSet[M]. If set is already a locked set, then it is just returned as is. If set isn't a locked set
// then the returned set is wrapped so that it is safe for concurrent use.
func NewLockedOrderedWith[M cmp.Ordered](set OrderedSet[M]) OrderedSet[M] {
	if _, lok := set.(locker); lok {
		return set
	}
	mu := &sync.RWMutex{}
	return &lockedOrdered[M]{
		set:     set,
		RWMutex: mu,
		Cond:    sync.NewCond(mu),
	}
}

func (s *lockedOrdered[M]) Contains(m M) bool {
	s.RLock()
	defer s.RUnlock()
	return s.set.Contains(m)
}

func (s *lockedOrdered[M]) Clear() int {
	s.L.Lock()
	if s.iterating {
		s.Wait()
	}
	defer s.L.Unlock()
	return s.set.Clear()
}

func (s *lockedOrdered[M]) Add(m M) bool {
	s.L.Lock()
	if s.iterating {
		s.Wait()
	}
	defer s.L.Unlock()
	return s.set.Add(m)
}

func (s *lockedOrdered[M]) Remove(m M) bool {
	s.L.Lock()
	if s.iterating {
		s.Wait()
	}
	defer s.L.Unlock()
	return s.set.Remove(m)
}

func (s *lockedOrdered[M]) Cardinality() int {
	if s == nil {
		return 0
	}
	s.RLock()
	defer s.RUnlock()
	return s.set.Cardinality()
}

// Iterator yields all elements in the set in order. It holds a lock for the duration of iteration. Calling methods other than
// `Contains` and `Cardinality` will block until the iteration is complete.
func (s *lockedOrdered[M]) Iterator(yield func(M) bool) {
	s.L.Lock()
	s.iterating = true
	defer func() {
		s.iterating = false
		s.Broadcast()
		s.L.Unlock()
	}()

	s.set.Iterator(yield)
}

func (s *lockedOrdered[M]) Clone() Set[M] {
	return NewLockedOrderedFrom(s.Iterator)
}

// Ordered iteration yields the index and value of each element in the set in order.
func (s *lockedOrdered[M]) Ordered(yield func(int, M) bool) {
	s.L.Lock()
	s.iterating = true
	defer func() {
		s.iterating = false
		s.Broadcast()
		s.L.Unlock()
	}()

	s.set.Ordered(yield)
}

func (s *lockedOrdered[M]) Backwards(yield func(int, M) bool) {
	s.L.Lock()
	s.iterating = true
	defer func() {
		s.iterating = false
		s.Broadcast()
		s.L.Unlock()
	}()

	s.set.Backwards(yield)
}

func (s *lockedOrdered[M]) NewEmptyOrdered() OrderedSet[M] {
	return NewLockedOrdered[M]()
}

func (s *lockedOrdered[M]) NewEmpty() Set[M] {
	return NewLockedOrdered[M]()
}

func (s *lockedOrdered[M]) Sort() {
	s.L.Lock()
	if s.iterating {
		s.Wait()
	}
	defer s.L.Unlock()

	s.set.Sort()
}

func (s *lockedOrdered[M]) At(i int) (M, bool) {
	s.RLock()
	defer s.RUnlock()

	return s.set.At(i)
}

func (s *lockedOrdered[M]) Index(m M) int {
	s.RLock()
	defer s.RUnlock()

	return s.set.Index(m)
}

func (s *lockedOrdered[M]) String() string {
	s.RLock()
	defer s.RUnlock()

	return "Locked" + s.set.String()
}
