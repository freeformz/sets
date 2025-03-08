package sets

import "iter"

// Map is a set implementation using the built in map type.
// Instances of this type are not safe to be used concurrently.
type Map[M comparable] map[M]struct{}

var _ Set[int] = Map[int]{}

// New returns an empty MapSets instance.
func New[M comparable]() Map[M] {
	return make(Map[M])
}

// NewFrom returns a new MapSets instance filled with the values from the sequence.
func NewFrom[M comparable](seq iter.Seq[M]) Map[M] {
	s := make(Map[M])
	for x := range seq {
		s.Add(x)
	}
	return s
}

func (s Map[M]) Contains(m M) bool {
	_, ok := s[m]
	return ok
}

func (s Map[M]) Clear() int {
	n := len(s)
	for k := range s {
		delete(s, k)
	}
	return n
}

func (s Map[M]) Add(m M) bool {
	if s.Contains(m) {
		return false
	}
	s[m] = struct{}{}
	return true
}

func (s Map[M]) Remove(m M) bool {
	if !s.Contains(m) {
		return false
	}
	delete(s, m)
	return true
}

func (s Map[M]) Cardinality() int {
	return len(s)
}

func (s Map[M]) Iterator(yield func(M) bool) {
	for k := range s {
		if !yield(k) {
			return
		}
	}
}

func (s Map[M]) Clone() Set[M] {
	return NewFrom(s.Iterator)
}
