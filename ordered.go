package sets

import (
	"cmp"
	"encoding/json"
	"fmt"
	"iter"
	"slices"
)

type Ordered[M cmp.Ordered] struct {
	idx    map[M]int
	values []M
}

var _ OrderedSet[int] = new(Ordered[int])

// NewOrdered returns an empty OrderedSet[M].
func NewOrdered[M cmp.Ordered]() *Ordered[M] {
	return &Ordered[M]{
		idx:    make(map[M]int),
		values: make([]M, 0),
	}
}

// NewOrderedFrom returns a new OrderedSet[M] filled with the values from the sequence.
func NewOrderedFrom[M cmp.Ordered](seq iter.Seq[M]) *Ordered[M] {
	s := NewOrdered[M]()
	for x := range seq {
		s.Add(x)
	}
	return s
}

// NewOrderedWith the values provides. Duplicates are removed.
func NewOrderedWith[M cmp.Ordered](m ...M) *Ordered[M] {
	return NewOrderedFrom(slices.Values(m))
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
	if s == nil {
		return 0
	}
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

// Ordered iteration yields the index and value of each element in the set in order.
func (s *Ordered[M]) Ordered(yield func(int, M) bool) {
	for i, k := range s.values {
		if !yield(i, k) {
			return
		}
	}
}

func (s *Ordered[M]) Backwards(yield func(int, M) bool) {
	for i := len(s.values) - 1; i >= 0; i-- {
		if !yield(i, s.values[i]) {
			return
		}
	}
}

func (s *Ordered[M]) NewEmptyOrdered() OrderedSet[M] {
	return NewOrdered[M]()
}

func (s *Ordered[M]) NewEmpty() Set[M] {
	return NewOrdered[M]()
}

func (s *Ordered[M]) Pop() (M, bool) {
	for k := range s.idx {
		s.Remove(k)
		return k, true
	}
	var m M
	return m, false
}

func (s *Ordered[M]) Sort() {
	slices.Sort(s.values)
	for i, v := range s.values {
		s.idx[v] = i
	}
}

func (s *Ordered[M]) At(i int) (M, bool) {
	var zero M
	if i < 0 || i >= len(s.values) {
		return zero, false
	}
	return s.values[i], true
}

func (s *Ordered[M]) Index(m M) int {
	i, ok := s.idx[m]
	if !ok {
		return -1
	}
	return i
}

func (s *Ordered[M]) String() string {
	var m M
	return fmt.Sprintf("OrderedSet[%T](%v)", m, s.values)
}

func (s *Ordered[M]) MarshalJSON() ([]byte, error) {
	if len(s.values) == 0 {
		return []byte("[]"), nil
	}

	d, err := json.Marshal(s.values)
	if err != nil {
		return d, fmt.Errorf("marshaling ordered set: %w", err)
	}
	return d, nil
}

func (s *Ordered[M]) UnmarshalJSON(d []byte) error {
	s.Clear()
	if s.values == nil {
		s.values = make([]M, 0)
	}
	if err := json.Unmarshal(d, &s.values); err != nil {
		return fmt.Errorf("unmarshaling ordered set: %w", err)
	}

	if s.idx == nil {
		s.idx = make(map[M]int)
	}
	for i, v := range s.values {
		s.idx[v] = i
	}

	return nil
}
