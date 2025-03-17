package sets

import (
	"encoding/json"
	"fmt"
	"iter"
	"maps"
	"slices"
)

type mapSet[M comparable] struct {
	set map[M]struct{}
}

// New returns an empty Set[M] instance.
func New[M comparable]() Set[M] {
	return &mapSet[M]{
		set: make(map[M]struct{}),
	}
}

// NewFrom returns a new Set[M] filled with the values from the sequence.
func NewFrom[M comparable](seq iter.Seq[M]) Set[M] {
	s := New[M]()
	for x := range seq {
		s.Add(x)
	}
	return s
}

// NewWith the values provides. Duplicates are removed.
func NewWith[M comparable](m ...M) Set[M] {
	return NewFrom(slices.Values(m))
}

func (s *mapSet[M]) Contains(m M) bool {
	_, ok := s.set[m]
	return ok
}

func (s *mapSet[M]) Clear() int {
	n := len(s.set)
	for k := range s.set {
		delete(s.set, k)
	}
	return n
}

func (s *mapSet[M]) Add(m M) bool {
	if s.Contains(m) {
		return false
	}
	s.set[m] = struct{}{}
	return true
}

func (s *mapSet[M]) Remove(m M) bool {
	if !s.Contains(m) {
		return false
	}
	delete(s.set, m)
	return true
}

func (s *mapSet[M]) Cardinality() int {
	return len(s.set)
}

// Iterator yields all elements in the set.
func (s *mapSet[M]) Iterator(yield func(M) bool) {
	for k := range s.set {
		if !yield(k) {
			return
		}
	}
}

func (s *mapSet[M]) Clone() Set[M] {
	return NewFrom(s.Iterator)
}

func (s *mapSet[M]) NewEmpty() Set[M] {
	return New[M]()
}

func (s *mapSet[M]) String() string {
	var m M
	return fmt.Sprintf("Set[%T](%v)", m, slices.Collect(maps.Keys(s.set)))
}

func (s *mapSet[M]) MarshalJSON() ([]byte, error) {
	v := slices.Collect(s.Iterator)
	if len(v) == 0 {
		return []byte("[]"), nil
	}

	d, err := json.Marshal(v)
	if err != nil {
		return d, fmt.Errorf("marshaling map set: %w", err)
	}
	return d, nil
}

func (s *mapSet[M]) UnmarshalJSON(d []byte) error {
	var um []M
	if err := json.Unmarshal(d, &um); err != nil {
		return fmt.Errorf("unmarshaling map set: %w", err)
	}

	s.Clear()
	for _, m := range um {
		s.Add(m)
	}

	return nil
}
