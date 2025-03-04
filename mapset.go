package sets

import "iter"

// MapSets is a set implementation using the built in map type.
// Instances of this type are not safe to be used concurrently.
type MapSet[M comparable] map[M]struct{}

var _ Set[int] = MapSet[int]{}

// New returns an empty MapSets instance.
func New[M comparable]() MapSet[M] {
	return make(MapSet[M])
}

// NewFrom returns a new MapSets instance filled with the values from the sequence.
func NewFrom[M comparable](seq iter.Seq[M]) MapSet[M] {
	s := make(MapSet[M])
	for x := range seq {
		s.Add(x)
	}
	return s
}

func (s MapSet[M]) Contains(m M) bool {
	_, ok := s[m]
	return ok
}

func (s MapSet[M]) Add(m M) bool {
	if s.Contains(m) {
		return false
	}
	s[m] = struct{}{}
	return true
}

func (s MapSet[M]) Remove(m M) bool {
	if !s.Contains(m) {
		return false
	}
	delete(s, m)
	return true
}

func (s MapSet[M]) Cardinality() int {
	return len(s)
}

func (s MapSet[M]) Iterator(yield func(M) bool) {
	for k := range s {
		if !yield(k) {
			return
		}
	}
}

func (s MapSet[M]) Clone() Set[M] {
	return NewFrom(s.Iterator)
}
