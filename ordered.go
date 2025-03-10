package sets

import (
	"cmp"
	"encoding/json"
	"fmt"
	"iter"
	"slices"
)

type ordered[M cmp.Ordered] struct {
	idx    map[M]int
	values []M
}

// NewOrdered returns an empty OrderedSet[M].
func NewOrdered[M cmp.Ordered]() OrderedSet[M] {
	return &ordered[M]{
		idx:    make(map[M]int),
		values: make([]M, 0),
	}
}

// NewOrderedFrom returns a new OrderedSet[M] filled with the values from the sequence.
func NewOrderedFrom[M cmp.Ordered](seq iter.Seq[M]) OrderedSet[M] {
	s := NewOrdered[M]()
	for x := range seq {
		s.Add(x)
	}
	return s
}

func (s *ordered[M]) Contains(m M) bool {
	_, ok := s.idx[m]
	return ok
}

func (s *ordered[M]) Clear() int {
	n := len(s.values)
	for k := range s.idx {
		delete(s.idx, k)
	}
	s.values = s.values[:0]
	return n
}

func (s *ordered[M]) Add(m M) bool {
	if s.Contains(m) {
		return false
	}
	s.values = append(s.values, m)
	s.idx[m] = len(s.values) - 1
	return true
}

func (s *ordered[M]) Remove(m M) bool {
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

func (s *ordered[M]) Cardinality() int {
	if s == nil {
		return 0
	}
	return len(s.values)
}

// Iterator yields all elements in the set in order.
func (s *ordered[M]) Iterator(yield func(M) bool) {
	for _, k := range s.values {
		if !yield(k) {
			return
		}
	}
}

func (s *ordered[M]) Clone() Set[M] {
	return NewOrderedFrom(s.Iterator)
}

// Ordered iteration yields the index and value of each element in the set in order.
func (s *ordered[M]) Ordered(yield func(int, M) bool) {
	for i, k := range s.values {
		if !yield(i, k) {
			return
		}
	}
}

func (s *ordered[M]) Backwards(yield func(int, M) bool) {
	for i := len(s.values) - 1; i >= 0; i-- {
		if !yield(i, s.values[i]) {
			return
		}
	}
}

func (s *ordered[M]) NewEmptyOrdered() OrderedSet[M] {
	return NewOrdered[M]()
}

func (s *ordered[M]) NewEmpty() Set[M] {
	return NewOrdered[M]()
}

func (s *ordered[M]) Sort() {
	slices.Sort(s.values)
	for i, v := range s.values {
		s.idx[v] = i
	}
}

func (s *ordered[M]) At(i int) (M, bool) {
	var zero M
	if i < 0 || i >= len(s.values) {
		return zero, false
	}
	return s.values[i], true
}

func (s *ordered[M]) Index(m M) int {
	i, ok := s.idx[m]
	if !ok {
		return -1
	}
	return i
}

func (s *ordered[M]) String() string {
	var m M
	return fmt.Sprintf("OrderedSet[%T](%v)", m, s.values)
}

func (s *ordered[M]) MarshalJSON() ([]byte, error) {
	if len(s.values) == 0 {
		return []byte("[]"), nil
	}

	d, err := json.Marshal(s.values)
	if err != nil {
		return d, fmt.Errorf("marshaling ordered set: %w", err)
	}
	return d, nil
}

func (s *ordered[M]) UnmarshalJSON(d []byte) error {
	s.Clear()
	if err := json.Unmarshal(d, &s.values); err != nil {
		return fmt.Errorf("unmarshaling ordered set: %w", err)
	}

	for i, v := range s.values {
		s.idx[v] = i
	}

	return nil
}
