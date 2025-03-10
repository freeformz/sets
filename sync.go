package sets

import (
	"encoding/json"
	"fmt"
	"iter"
	"slices"
	"sync"
)

type syncMap[M comparable] struct {
	m sync.Map
}

var _ Set[int] = new(syncMap[int])

// NewSync returns an empty Set[M] that is backed by a sync.Map, making it safe for concurrent use.
// Please read the documentation for [sync.Map] to understand the behavior of modifying the map.
func NewSync[M comparable]() Set[M] {
	return &syncMap[M]{
		m: sync.Map{},
	}
}

// NewSyncFrom returns a new Set[M] filled with the values from the sequence and is backed by a sync.Mao, making it safe
// for concurrent use. Please read the documentation for [sync.Map] to understand the behavior of modifying the map.
func NewSyncFrom[M comparable](seq iter.Seq[M]) Set[M] {
	s := NewSync[M]()
	for x := range seq {
		s.Add(x)
	}
	return s
}

func (s *syncMap[M]) Contains(m M) bool {
	_, ok := s.m.Load(m)
	return ok
}

func (s *syncMap[M]) Clear() int {
	var n int
	s.m.Range(func(_, _ interface{}) bool {
		n++
		return true
	})
	s.m.Clear()
	return n
}

func (s *syncMap[M]) Add(m M) bool {
	_, loaded := s.m.LoadOrStore(m, struct{}{})
	return !loaded
}

func (s *syncMap[M]) Remove(m M) bool {
	_, ok := s.m.LoadAndDelete(m)
	return ok
}

func (s *syncMap[M]) Cardinality() int {
	if s == nil {
		return 0
	}
	var n int
	s.m.Range(func(_, _ interface{}) bool {
		n++
		return true
	})
	return n
}

// Iterator yields all elements in the set. It is safe to call concurrently with other methods, but the order and
// behavior is undefined, as per [sync.Map]'s `Range`.
func (s *syncMap[M]) Iterator(yield func(M) bool) {
	s.m.Range(func(key, _ interface{}) bool {
		return yield(key.(M))
	})
}

func (s *syncMap[M]) Clone() Set[M] {
	return NewSyncFrom(s.Iterator)
}

func (s *syncMap[M]) NewEmpty() Set[M] {
	return NewSync[M]()
}

func (s *syncMap[M]) String() string {
	var m M
	return fmt.Sprintf("SyncSet[%T](%v)", m, slices.Collect(s.Iterator))
}

func (s *syncMap[M]) MarshalJSON() ([]byte, error) {
	v := slices.Collect(s.Iterator)
	if len(v) == 0 {
		return []byte("[]"), nil
	}

	d, err := json.Marshal(v)
	if err != nil {
		return d, fmt.Errorf("marshaling sync set: %w", err)
	}
	return d, nil
}

func (s *syncMap[M]) UnmarshalJSON(d []byte) error {
	var x []M
	if err := json.Unmarshal(d, &x); err != nil {
		return fmt.Errorf("unmarshaling sync set: %w", err)
	}
	s.m.Clear()
	for _, m := range x {
		s.Add(m)
	}
	return nil
}
