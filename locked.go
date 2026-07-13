package sets

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"iter"
	"slices"
	"sync"
)

// Locked is a concurrency safe wrapper around a Set[M]. It uses a read-write lock to allow multiple readers to access
// the set concurrently, but only one writer at a time. The set is not ordered and does not guarantee the order of
// elements when iterating over them. It is safe for concurrent use.
//
// Locked delegates all of the package's optional optimization interfaces (Unioner,
// Intersectioner, Differencer, SymmetricDifferencer, Equaler, Disjointer, Subsetter, Maxer, and
// Minner) to the inner set under the read lock, so wrapping an optimizable set (e.g. a BitSet)
// keeps its fast paths. A locked operand is unwrapped with a non-blocking lock attempt and the
// delegation declines under contention, so these methods never deadlock; the declined call falls
// back to the package-level generic path, which locks one set at a time.
type Locked[M comparable] struct {
	set Set[M]
	sync.RWMutex
}

var _ Set[int] = new(Locked[int])
var _ driver.Valuer = new(Locked[int])
var _ Unioner[int] = new(Locked[int])
var _ Intersectioner[int] = new(Locked[int])
var _ Differencer[int] = new(Locked[int])
var _ SymmetricDifferencer[int] = new(Locked[int])
var _ Equaler[int] = new(Locked[int])
var _ Disjointer[int] = new(Locked[int])
var _ Subsetter[int] = new(Locked[int])
var _ Maxer[int] = new(Locked[int])
var _ Minner[int] = new(Locked[int])
var _ tryUnwrapper[int] = new(Locked[int])

// NewLocked returns an empty *Locked[M] that is safe for concurrent use.
func NewLocked[M comparable]() *Locked[M] {
	return &Locked[M]{set: New[M]()}
}

// NewLockedFrom returns a new *Locked[M] filled with the values from the sequence.
func NewLockedFrom[M comparable](seq iter.Seq[M]) *Locked[M] {
	s := NewLocked[M]()
	for x := range seq {
		s.Add(x)
	}
	return s
}

// NewLockedWith returns a *Locked[M] with the values provided.
func NewLockedWith[M comparable](m ...M) *Locked[M] {
	return NewLockedFrom(slices.Values(m))
}

// NewLockedWrapping returns a Set[M]. If set is already a locked set, then it is just returned as is. If set isn't a locked set
// then the returned set is wrapped so that it is safe for concurrent use.
func NewLockedWrapping[M comparable](set Set[M]) Set[M] {
	if _, ok := set.(Locker); ok {
		return set
	}

	lset := NewLocked[M]()
	lset.set = set

	return lset
}

// Contains returns true if the set contains the element.
func (s *Locked[M]) Contains(m M) bool {
	s.RLock()
	defer s.RUnlock()
	return s.set.Contains(m)
}

// Clear the set and returns the number of elements removed.
func (s *Locked[M]) Clear() int {
	s.Lock()
	defer s.Unlock()
	return s.set.Clear()
}

// Add an element to the set. Returns true if the element was added, false if it was already present.
func (s *Locked[M]) Add(m M) bool {
	s.Lock()
	defer s.Unlock()

	return s.set.Add(m)
}

// Remove an element from the set. Returns true if the element was removed, false if it was not present.
func (s *Locked[M]) Remove(m M) bool {
	s.Lock()
	defer s.Unlock()

	return s.set.Remove(m)
}

// Cardinality returns the number of elements in the set.
func (s *Locked[M]) Cardinality() int {
	if s == nil {
		return 0
	}
	s.RLock()
	defer s.RUnlock()

	return s.set.Cardinality()
}

// Iterator yields all elements in the set. It takes a snapshot of the elements under a read lock and then iterates
// without holding the lock. This means it is safe to call any method on the set from within the yield callback,
// but the iteration may not reflect concurrent modifications.
func (s *Locked[M]) Iterator(yield func(M) bool) {
	s.RLock()
	elems := make([]M, 0, s.set.Cardinality())
	for v := range s.set.Iterator {
		elems = append(elems, v)
	}
	s.RUnlock()

	for _, v := range elems {
		if !yield(v) {
			return
		}
	}
}

// Clone returns a new set of the same underlying type.
func (s *Locked[M]) Clone() Set[M] {
	s.RLock()
	defer s.RUnlock()
	if s.set == nil {
		return NewLocked[M]()
	}
	return &Locked[M]{set: s.set.Clone()}
}

// NewEmpty returns a new empty set of the same underlying type.
func (s *Locked[M]) NewEmpty() Set[M] {
	s.RLock()
	defer s.RUnlock()
	if s.set == nil {
		return NewLocked[M]()
	}
	return &Locked[M]{set: s.set.NewEmpty()}
}

// Pop removes and returns an element from the set. If the set is empty, it returns the zero value of M and false.
func (s *Locked[M]) Pop() (M, bool) {
	s.Lock()
	defer s.Unlock()

	return s.set.Pop()
}

// tryUnwrapper is implemented by the locked wrappers so that one wrapper's optional-interface
// delegation can reach another wrapper's inner set. The receiver's own lock is acquired normally,
// but an operand wrapper's lock is only try-acquired: two goroutines running mirror-image
// operations on the same pair of locked sets therefore can never deadlock — one of them declines
// and takes the generic path, which locks one set at a time.
type tryUnwrapper[M comparable] interface {
	tryUnwrap() (Set[M], func(), bool)
}

// tryUnwrapOperand returns the set an inner optional-interface method should be handed for other:
// a locked wrapper's inner set (read-locked, released via unlock) or other itself (unlock is a
// no-op). ok is false when a wrapper's lock is contended or unusable; the caller must decline to
// the generic path.
func tryUnwrapOperand[M comparable](other Set[M]) (_ Set[M], unlock func(), ok bool) {
	if w, ok := other.(tryUnwrapper[M]); ok {
		return w.tryUnwrap()
	}
	return other, func() {}, true
}

//lint:ignore U1000 reached via the tryUnwrapper[M] type assertion in tryUnwrapOperand
func (s *Locked[M]) tryUnwrap() (Set[M], func(), bool) {
	if s == nil || !s.TryRLock() {
		return nil, nil, false
	}
	if s.set == nil { // zero value: nothing to hand to an inner method
		s.RUnlock()
		return nil, nil, false
	}
	return s.set, s.RUnlock, true
}

// Union implements Unioner by delegating to the inner set's Unioner under the read lock,
// unwrapping a locked other operand so two locked wrappers around optimizable sets still combine
// on the fast path. On success the result is a new Locked wrapping the inner result. It declines
// — sending the package-level Union down the generic path — when the inner set doesn't implement
// Unioner or declines the operand, or when the operand wrapper's lock is contended (see
// tryUnwrapper for why that cannot deadlock).
func (s *Locked[M]) Union(other Set[M]) (Set[M], bool) {
	if s == nil {
		return nil, false
	}
	s.RLock()
	defer s.RUnlock()
	u, ok := s.set.(Unioner[M])
	if !ok {
		return nil, false
	}
	operand, unlock, ok := tryUnwrapOperand(other)
	if !ok {
		return nil, false
	}
	defer unlock()
	c, ok := u.Union(operand)
	if !ok {
		return nil, false
	}
	return &Locked[M]{set: c}, true
}

// Intersection implements Intersectioner by delegating to the inner set's Intersectioner; see
// Union for the delegation and locking rules shared by all of Locked's optional-interface
// methods.
func (s *Locked[M]) Intersection(other Set[M]) (Set[M], bool) {
	if s == nil {
		return nil, false
	}
	s.RLock()
	defer s.RUnlock()
	i, ok := s.set.(Intersectioner[M])
	if !ok {
		return nil, false
	}
	operand, unlock, ok := tryUnwrapOperand(other)
	if !ok {
		return nil, false
	}
	defer unlock()
	c, ok := i.Intersection(operand)
	if !ok {
		return nil, false
	}
	return &Locked[M]{set: c}, true
}

// Difference implements Differencer by delegating to the inner set's Differencer; see Union for
// the delegation and locking rules shared by all of Locked's optional-interface methods.
func (s *Locked[M]) Difference(other Set[M]) (Set[M], bool) {
	if s == nil {
		return nil, false
	}
	s.RLock()
	defer s.RUnlock()
	d, ok := s.set.(Differencer[M])
	if !ok {
		return nil, false
	}
	operand, unlock, ok := tryUnwrapOperand(other)
	if !ok {
		return nil, false
	}
	defer unlock()
	c, ok := d.Difference(operand)
	if !ok {
		return nil, false
	}
	return &Locked[M]{set: c}, true
}

// SymmetricDifference implements SymmetricDifferencer by delegating to the inner set's
// SymmetricDifferencer; see Union for the delegation and locking rules shared by all of Locked's
// optional-interface methods.
func (s *Locked[M]) SymmetricDifference(other Set[M]) (Set[M], bool) {
	if s == nil {
		return nil, false
	}
	s.RLock()
	defer s.RUnlock()
	sd, ok := s.set.(SymmetricDifferencer[M])
	if !ok {
		return nil, false
	}
	operand, unlock, ok := tryUnwrapOperand(other)
	if !ok {
		return nil, false
	}
	defer unlock()
	c, ok := sd.SymmetricDifference(operand)
	if !ok {
		return nil, false
	}
	return &Locked[M]{set: c}, true
}

// Equal implements Equaler by delegating to the inner set's Equaler; see Union for the delegation
// and locking rules shared by all of Locked's optional-interface methods.
func (s *Locked[M]) Equal(other Set[M]) (bool, bool) {
	if s == nil {
		return false, false
	}
	s.RLock()
	defer s.RUnlock()
	e, ok := s.set.(Equaler[M])
	if !ok {
		return false, false
	}
	operand, unlock, ok := tryUnwrapOperand(other)
	if !ok {
		return false, false
	}
	defer unlock()
	return e.Equal(operand)
}

// Disjoint implements Disjointer by delegating to the inner set's Disjointer; see Union for the
// delegation and locking rules shared by all of Locked's optional-interface methods.
func (s *Locked[M]) Disjoint(other Set[M]) (bool, bool) {
	if s == nil {
		return false, false
	}
	s.RLock()
	defer s.RUnlock()
	d, ok := s.set.(Disjointer[M])
	if !ok {
		return false, false
	}
	operand, unlock, ok := tryUnwrapOperand(other)
	if !ok {
		return false, false
	}
	defer unlock()
	return d.Disjoint(operand)
}

// Subset implements Subsetter by delegating to the inner set's Subsetter; see Union for the
// delegation and locking rules shared by all of Locked's optional-interface methods.
func (s *Locked[M]) Subset(other Set[M]) (bool, bool) {
	if s == nil {
		return false, false
	}
	s.RLock()
	defer s.RUnlock()
	sub, ok := s.set.(Subsetter[M])
	if !ok {
		return false, false
	}
	operand, unlock, ok := tryUnwrapOperand(other)
	if !ok {
		return false, false
	}
	defer unlock()
	return sub.Subset(operand)
}

// Max implements Maxer by delegating to the inner set's Max under the read lock. Locked's element
// type is only comparable, so the inner set is matched structurally rather than via Maxer[M]
// (which requires cmp.Ordered); when Locked is instantiated with an ordered element type it
// satisfies Maxer[M] and the package-level Max uses this automatically.
func (s *Locked[M]) Max() (M, bool) {
	var zero M
	if s == nil {
		return zero, false
	}
	s.RLock()
	defer s.RUnlock()
	if mx, ok := s.set.(interface{ Max() (M, bool) }); ok {
		return mx.Max()
	}
	return zero, false
}

// Min implements Minner by delegating to the inner set's Min under the read lock; see Max for how
// the inner set is matched.
func (s *Locked[M]) Min() (M, bool) {
	var zero M
	if s == nil {
		return zero, false
	}
	s.RLock()
	defer s.RUnlock()
	if mn, ok := s.set.(interface{ Min() (M, bool) }); ok {
		return mn.Min()
	}
	return zero, false
}

// String returns a string representation of the set. It returns a string of the form LockedSet[T](<elements>).
func (s *Locked[M]) String() string {
	s.RLock()
	defer s.RUnlock()
	return "Locked" + s.set.String()
}

// Value implements the driver.Valuer interface. It returns the JSON representation of the set.
func (s *Locked[M]) Value() (driver.Value, error) {
	return s.MarshalJSON()
}

// MarshalJSON implements json.Marshaler. It will marshal the set into a JSON array of the elements in the set. If the
// set is empty an empty JSON array is returned.
func (s *Locked[M]) MarshalJSON() ([]byte, error) {
	s.RLock()
	defer s.RUnlock()

	jm, ok := s.set.(json.Marshaler)
	if !ok {
		return nil, fmt.Errorf("cannot marshal set of type %T - not json.Marshaler", s.set)
	}

	d, err := jm.MarshalJSON()
	if err != nil {
		return d, fmt.Errorf("marshaling locked set: %w", err)
	}
	return d, nil
}

// UnmarshalJSON implements json.Unmarshaler. It expects a JSON array of the elements in the set. If the set is empty,
// it returns an empty set. If the JSON is invalid, it returns an error.
func (s *Locked[M]) UnmarshalJSON(d []byte) error {
	s.Lock()
	defer s.Unlock()

	if s.set == nil {
		s.set = New[M]()
	}
	um, ok := s.set.(json.Unmarshaler)
	if !ok {
		return fmt.Errorf("cannot unmarshal set of type %T - not json.Unmarshaler", s.set)
	}

	if err := um.UnmarshalJSON(d); err != nil {
		return fmt.Errorf("unmarshaling locked set: %w", err)
	}

	return nil
}

// Scan implements the sql.Scanner interface. It scans the value from the database into the set. It expects a JSON array
// of the elements in the set. If the JSON is invalid an error is returned. If the value is nil an empty set is
// returned.
func (s *Locked[M]) Scan(src any) error {
	s.Lock()
	defer s.Unlock()

	if s.set == nil {
		s.set = New[M]()
	}

	return scanValue[M](src, s.set.Clear, func(data []byte) error {
		um, ok := s.set.(json.Unmarshaler)
		if !ok {
			return fmt.Errorf("cannot unmarshal set of type %T - not json.Unmarshaler", s.set)
		}
		return um.UnmarshalJSON(data)
	})
}
