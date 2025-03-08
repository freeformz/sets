package sets

import "iter"

// Ordered is a set implementation that maintains the order of elements. If the same item is added multiple times,
// the first insertion determines the order.
type Ordered[M comparable] struct {
	idx    map[M]int
	values []M
}

var _ Set[int] = new(Ordered[int])

// NewOrdered returns an empty OrderedSets instance.
func NewOrdered[M comparable]() *Ordered[M] {
	return &Ordered[M]{
		idx:    make(map[M]int),
		values: make([]M, 0),
	}
}

// NewOrderedFrom returns a new OrderedSets instance filled with the values from the sequence.
func NewOrderedFrom[M comparable](seq iter.Seq[M]) *Ordered[M] {
	s := NewOrdered[M]()
	for x := range seq {
		s.Add(x)
	}
	return s
}

func (s *Ordered[M]) Contains(m M) bool {
	_, ok := s.idx[m]
	return ok
}

func (s *Ordered[M]) Clear() int {
	n := len(s.values)
	for k := range s.idx {
		delete(s.idx, k)
	}
	s.values = s.values[:0]
	return n
}

func (s *Ordered[M]) Add(m M) bool {
	if s.Contains(m) {
		return false
	}
	s.values = append(s.values, m)
	s.idx[m] = len(s.values) - 1
	return true
}

func (s *Ordered[M]) Remove(m M) bool {
	if !s.Contains(m) {
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

func (s *Ordered[M]) Cardinality() int {
	return len(s.values)
}

// Iterator yields all elements in the set in order.
func (s *Ordered[M]) Iterator(yield func(M) bool) {
	for _, k := range s.values {
		if !yield(k) {
			return
		}
	}
}

func (s *Ordered[M]) Clone() Set[M] {
	return NewOrderedFrom(s.Iterator)
}
