package set

import (
	"fmt"
	"iter"
	"maps"
	"slices"
)

type mapSet[M comparable] map[M]struct{}

// New returns an empty Set[M] instance.
func New[M comparable]() Set[M] {
	return make(mapSet[M])
}

// NewFrom returns a new Set[M] filled with the values from the sequence.
func NewFrom[M comparable](seq iter.Seq[M]) Set[M] {
	s := make(mapSet[M])
	for x := range seq {
		s.Add(x)
	}
	return s
}

func (s mapSet[M]) Contains(m M) bool {
	_, ok := s[m]
	return ok
}

func (s mapSet[M]) Clear() int {
	n := len(s)
	for k := range s {
		delete(s, k)
	}
	return n
}

func (s mapSet[M]) Add(m M) bool {
	if s.Contains(m) {
		return false
	}
	s[m] = struct{}{}
	return true
}

func (s mapSet[M]) Remove(m M) bool {
	if !s.Contains(m) {
		return false
	}
	delete(s, m)
	return true
}

func (s mapSet[M]) Cardinality() int {
	if s == nil {
		return 0
	}
	return len(s)
}

// Iterator yields all elements in the set.
func (s mapSet[M]) Iterator(yield func(M) bool) {
	for k := range s {
		if !yield(k) {
			return
		}
	}
}

func (s mapSet[M]) Clone() Set[M] {
	return NewFrom(s.Iterator)
}

func (s mapSet[M]) NewEmpty() Set[M] {
	return New[M]()
}

func (s mapSet[M]) String() string {
	var m M
	return fmt.Sprintf("Set[%T](%v)", m, slices.Collect(maps.Keys(s)))
}
