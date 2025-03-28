package sets

import (
	"encoding/json"
	"fmt"
	"iter"
	"maps"
	"slices"
)

type Map[M comparable] struct {
	set map[M]struct{}
}

var _ Set[int] = new(Map[int])

// NewMap returns an empty Set[M] instance.
func NewMap[M comparable]() *Map[M] {
	return &Map[M]{
		set: make(map[M]struct{}),
	}
}

// NewMapFrom returns a new Set[M] filled with the values from the sequence.
func NewMapFrom[M comparable](seq iter.Seq[M]) *Map[M] {
	s := NewMap[M]()
	for x := range seq {
		s.Add(x)
	}
	return s
}

// NewMapWith the values provides. Duplicates are removed.
func NewMapWith[M comparable](m ...M) *Map[M] {
	return NewMapFrom(slices.Values(m))
}

func (s *Map[M]) Contains(m M) bool {
	_, ok := s.set[m]
	return ok
}

func (s *Map[M]) Clear() int {
	n := len(s.set)
	for k := range s.set {
		delete(s.set, k)
	}
	return n
}

func (s *Map[M]) Add(m M) bool {
	if s.Contains(m) {
		return false
	}
	s.set[m] = struct{}{}
	return true
}

func (s *Map[M]) Remove(m M) bool {
	if !s.Contains(m) {
		return false
	}
	delete(s.set, m)
	return true
}

func (s *Map[M]) Cardinality() int {
	return len(s.set)
}

// Iterator yields all elements in the set.
func (s *Map[M]) Iterator(yield func(M) bool) {
	for k := range s.set {
		if !yield(k) {
			return
		}
	}
}

func (s *Map[M]) Clone() Set[M] {
	return NewMapFrom(s.Iterator)
}

func (s *Map[M]) NewEmpty() Set[M] {
	return NewMap[M]()
}

func (s *Map[M]) Pop() (M, bool) {
	for k := range s.set {
		delete(s.set, k)
		return k, true
	}
	var m M
	return m, false
}

func (s *Map[M]) String() string {
	var m M
	return fmt.Sprintf("Set[%T](%v)", m, slices.Collect(maps.Keys(s.set)))
}

func (s *Map[M]) MarshalJSON() ([]byte, error) {
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

func (s *Map[M]) UnmarshalJSON(d []byte) error {
	var um []M
	if err := json.Unmarshal(d, &um); err != nil {
		return fmt.Errorf("unmarshaling map set: %w", err)
	}

	s.Clear()
	if s.set == nil {
		s.set = make(map[M]struct{})
	}
	for _, m := range um {
		s.Add(m)
	}

	return nil
}
