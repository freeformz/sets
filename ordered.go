package sets

import "iter"

// OrderedSet is a set implementation that maintains the order of elements. Instances of this type are not safe to be
// used concurrently. If the same item is added multiple times, the first insertion determines the order.
type OrderedSet[M comparable] struct {
	m map[M]int
	i []M
}

var _ Set[int] = new(OrderedSet[int])

// NewOrdered returns an empty OrderedSets instance.
func NewOrdered[M comparable]() *OrderedSet[M] {
	return &OrderedSet[M]{
		m: make(map[M]int),
		i: make([]M, 0),
	}
}

// NewOrderedFrom returns a new OrderedSets instance filled with the values from the sequence.
func NewOrderedFrom[M comparable](seq iter.Seq[M]) *OrderedSet[M] {
	s := NewOrdered[M]()
	for x := range seq {
		s.Add(x)
	}
	return s
}

func (s *OrderedSet[M]) Contains(m M) bool {
	_, ok := s.m[m]
	return ok
}

func (s *OrderedSet[M]) Add(m M) bool {
	if s.Contains(m) {
		return false
	}
	s.m[m] = len(s.i) - 1
	s.i = append(s.i, m)
	return true
}

func (s *OrderedSet[M]) Remove(m M) bool {
	if !s.Contains(m) {
		return false
	}
	i := s.m[m]
	s.i = append(s.i[:i], s.i[i+1:]...)
	delete(s.m, m)
	return true
}

func (s *OrderedSet[M]) Cardinality() int {
	return len(s.i)
}

func (s *OrderedSet[M]) Iterator(yield func(M) bool) {
	for _, k := range s.i {
		if !yield(k) {
			return
		}
	}
}

func (s *OrderedSet[M]) Clone() Set[M] {
	return NewOrderedFrom(s.Iterator)
}
