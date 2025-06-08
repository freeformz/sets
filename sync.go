package sets

import (
	"encoding/json"
	"fmt"
	"iter"
	"slices"
	"sync"
)

// SyncMap is a concurrency safe set type that uses a sync.Map.
type SyncMap[M comparable] struct {
	m sync.Map
}

var _ Set[int] = new(SyncMap[int])

// NewSyncMap returns an empty Set[M] that is backed by a sync.Map, making it safe for concurrent use.
// Please read the documentation for [sync.Map] to understand the behavior of modifying the map.
func NewSyncMap[M comparable]() *SyncMap[M] {
	return &SyncMap[M]{
		m: sync.Map{},
	}
}

// NewSyncMapFrom returns a new Set[M] filled with the values from the sequence and is backed by a sync.Map, making it safe
// for concurrent use. Please read the documentation for [sync.Map] to understand the behavior of modifying the map.
func NewSyncMapFrom[M comparable](seq iter.Seq[M]) *SyncMap[M] {
	s := NewSyncMap[M]()
	for x := range seq {
		s.Add(x)
	}
	return s
}

// NewSyncMapWith returns a new Set[M] filled with the values provided and is backed by a sync.Map, making it safe
// for concurrent use. Please read the documentation for [sync.Map] to understand the behavior of modifying the map.
func NewSyncMapWith[M comparable](m ...M) *SyncMap[M] {
	return NewSyncMapFrom(slices.Values(m))
}

func (s *SyncMap[M]) Contains(m M) bool {
	_, ok := s.m.Load(m)
	return ok
}

func (s *SyncMap[M]) Clear() int {
	var n int
	s.m.Range(func(_, _ interface{}) bool {
		n++
		return true
	})
	s.m.Clear()
	return n
}

func (s *SyncMap[M]) Add(m M) bool {
	_, loaded := s.m.LoadOrStore(m, struct{}{})
	return !loaded
}

func (s *SyncMap[M]) Pop() (M, bool) {
	var m M
	var ok bool

	s.m.Range(func(key, _ interface{}) bool {
		if _, ok = s.m.LoadAndDelete(key); ok {
			m = key.(M)
			ok = true
			return false
		}
		return true
	})
	return m, ok
}

func (s *SyncMap[M]) Remove(m M) bool {
	_, ok := s.m.LoadAndDelete(m)
	return ok
}

func (s *SyncMap[M]) Cardinality() int {
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
func (s *SyncMap[M]) Iterator(yield func(M) bool) {
	s.m.Range(func(key, _ interface{}) bool {
		return yield(key.(M))
	})
}

func (s *SyncMap[M]) Clone() Set[M] {
	return NewSyncMapFrom(s.Iterator)
}

func (s *SyncMap[M]) NewEmpty() Set[M] {
	return NewSyncMap[M]()
}

func (s *SyncMap[M]) String() string {
	var m M
	return fmt.Sprintf("SyncSet[%T](%v)", m, slices.Collect(s.Iterator))
}

func (s *SyncMap[M]) MarshalJSON() ([]byte, error) {
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

func (s *SyncMap[M]) UnmarshalJSON(d []byte) error {
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

// Scan implements the sql.Scanner interface. It scans the value from the database into the set. It expects a JSON array
// of the elements in the set. If the JSON is invalid an error is returned. If the value is nil an empty set is
// returned.
func (s *SyncMap[M]) Scan(src any) error {
	return scanValue[M](src, s.Clear, s.UnmarshalJSON)
}
