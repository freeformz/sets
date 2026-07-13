package sets

import (
	"cmp"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"iter"
	"slices"
	"sync"
)

// LockedOrdered is a concurrency safe wrapper around an OrderedSet[M]. It uses a read-write lock to allow multiple readers.
//
// LockedOrdered delegates all of the package's optional optimization interfaces (Unioner,
// Intersectioner, Differencer, SymmetricDifferencer, Equaler, Disjointer, Subsetter, Maxer, and
// Minner) to the inner set under the read lock, exactly as Locked does — see Locked for the
// locking rules that make the delegation deadlock-free.
type LockedOrdered[M cmp.Ordered] struct {
	set OrderedSet[M]
	sync.RWMutex
}

var _ Set[int] = new(LockedOrdered[int])
var _ driver.Valuer = new(LockedOrdered[int])
var _ Unioner[int] = new(LockedOrdered[int])
var _ Intersectioner[int] = new(LockedOrdered[int])
var _ Differencer[int] = new(LockedOrdered[int])
var _ SymmetricDifferencer[int] = new(LockedOrdered[int])
var _ Equaler[int] = new(LockedOrdered[int])
var _ Disjointer[int] = new(LockedOrdered[int])
var _ Subsetter[int] = new(LockedOrdered[int])
var _ Maxer[int] = new(LockedOrdered[int])
var _ Minner[int] = new(LockedOrdered[int])
var _ tryUnwrapper[int] = new(LockedOrdered[int])

// NewLockedOrdered returns an empty *LockedOrdered[M] instance that is safe for concurrent use.
func NewLockedOrdered[M cmp.Ordered]() *LockedOrdered[M] {
	return &LockedOrdered[M]{set: NewOrdered[M]()}
}

// NewLockedOrderedFrom returns a new *LockedOrdered[M] instance filled with the values from the sequence. The set is safe
// for concurrent use.
func NewLockedOrderedFrom[M cmp.Ordered](seq iter.Seq[M]) *LockedOrdered[M] {
	s := NewLockedOrdered[M]()
	for x := range seq {
		s.Add(x)
	}
	return s
}

// NewLockedOrderedWith returns a *LockedOrdered[M] with the values provided.
func NewLockedOrderedWith[M cmp.Ordered](m ...M) *LockedOrdered[M] {
	return NewLockedOrderedFrom(slices.Values(m))
}

// NewLockedOrderedWrapping returns an OrderedSet[M]. If the set is already a locked set, then it is just returned as
// is. If the set isn't a locked set then the returned set is wrapped so that it is safe for concurrent use.
func NewLockedOrderedWrapping[M cmp.Ordered](set OrderedSet[M]) OrderedSet[M] {
	if _, ok := set.(Locker); ok {
		return set
	}
	lset := NewLockedOrdered[M]()
	lset.set = set
	return lset
}

// Contains returns true if the set contains the element.
func (s *LockedOrdered[M]) Contains(m M) bool {
	s.RLock()
	defer s.RUnlock()
	return s.set.Contains(m)
}

// Clear the set and returns the number of elements removed.
func (s *LockedOrdered[M]) Clear() int {
	s.Lock()
	defer s.Unlock()
	return s.set.Clear()
}

// Add an element to the set. Returns true if the element was added, false if it was already present.
func (s *LockedOrdered[M]) Add(m M) bool {
	s.Lock()
	defer s.Unlock()
	return s.set.Add(m)
}

// Remove an element from the set. Returns true if the element was removed, false if it was not present.
func (s *LockedOrdered[M]) Remove(m M) bool {
	s.Lock()
	defer s.Unlock()
	return s.set.Remove(m)
}

// Cardinality returns the number of elements in the set.
func (s *LockedOrdered[M]) Cardinality() int {
	if s == nil {
		return 0
	}
	s.RLock()
	defer s.RUnlock()

	return s.set.Cardinality()
}

// snapshot returns the elements in order, collected under a read lock.
func (s *LockedOrdered[M]) snapshot() []M {
	s.RLock()
	defer s.RUnlock()
	elems := make([]M, 0, s.set.Cardinality())
	for v := range s.set.Iterator {
		elems = append(elems, v)
	}
	return elems
}

// Iterator yields all elements in the set in order. It takes a snapshot of the elements under a read lock and then
// iterates without holding the lock. This means it is safe to call any method on the set from within the yield
// callback, but the iteration may not reflect concurrent modifications.
func (s *LockedOrdered[M]) Iterator(yield func(M) bool) {
	for _, v := range s.snapshot() {
		if !yield(v) {
			return
		}
	}
}

// Clone returns a new set of the same underlying type.
func (s *LockedOrdered[M]) Clone() Set[M] {
	s.RLock()
	defer s.RUnlock()
	if s.set == nil {
		return NewLockedOrdered[M]()
	}
	// OrderedSet documents that Clone always returns an ordered set.
	return &LockedOrdered[M]{set: s.set.Clone().(OrderedSet[M])}
}

// Ordered iteration yields the index and value of each element in the set in order. It takes a snapshot of the
// elements under a read lock and then iterates without holding the lock. This means it is safe to call any method
// on the set from within the yield callback, but the iteration may not reflect concurrent modifications.
func (s *LockedOrdered[M]) Ordered(yield func(int, M) bool) {
	for i, v := range s.snapshot() {
		if !yield(i, v) {
			return
		}
	}
}

// Backwards iteration yields the index and value of each element in the set in reverse order. It takes a snapshot of
// the elements under a read lock and then iterates without holding the lock. This means it is safe to call any method
// on the set from within the yield callback, but the iteration may not reflect concurrent modifications.
func (s *LockedOrdered[M]) Backwards(yield func(int, M) bool) {
	elems := s.snapshot()
	for i := len(elems) - 1; i >= 0; i-- {
		if !yield(i, elems[i]) {
			return
		}
	}
}

// NewEmptyOrdered returns a new empty ordered set of the same underlying type.
func (s *LockedOrdered[M]) NewEmptyOrdered() OrderedSet[M] {
	s.RLock()
	defer s.RUnlock()
	if s.set == nil {
		return NewLockedOrdered[M]()
	}
	return &LockedOrdered[M]{set: s.set.NewEmptyOrdered()}
}

// NewEmpty returns a new empty set of the same underlying type.
func (s *LockedOrdered[M]) NewEmpty() Set[M] {
	return s.NewEmptyOrdered()
}

// Pop removes and returns an element from the set. If the set is empty, it returns the zero value of M and false.
func (s *LockedOrdered[M]) Pop() (M, bool) {
	s.Lock()
	defer s.Unlock()

	return s.set.Pop()
}

// Sort the set in ascending order.
func (s *LockedOrdered[M]) Sort() {
	s.Lock()
	defer s.Unlock()

	s.set.Sort()
}

// At returns the element at the index. If the index is out of bounds, the second return value is false.
func (s *LockedOrdered[M]) At(i int) (M, bool) {
	s.RLock()
	defer s.RUnlock()

	return s.set.At(i)
}

// Index returns the index of the element in the set, or -1 if not present.
func (s *LockedOrdered[M]) Index(m M) int {
	s.RLock()
	defer s.RUnlock()

	return s.set.Index(m)
}

//lint:ignore U1000 reached via the tryUnwrapper[M] type assertion in tryUnwrapOperand
func (s *LockedOrdered[M]) tryUnwrap() (Set[M], func(), bool) {
	if s == nil || !s.TryRLock() {
		return nil, nil, false
	}
	if s.set == nil { // zero value: nothing to hand to an inner method
		s.RUnlock()
		return nil, nil, false
	}
	return s.set, s.RUnlock, true
}

// delegated returns the result of an inner set-algebra delegation rewrapped in a new
// LockedOrdered. It declines when the inner result is not an OrderedSet (a conforming inner
// OrderedSet's algebra methods always produce one, so this is purely defensive).
func (s *LockedOrdered[M]) delegated(c Set[M], ok bool) (Set[M], bool) {
	if !ok {
		return nil, false
	}
	co, ok := c.(OrderedSet[M])
	if !ok {
		return nil, false
	}
	return &LockedOrdered[M]{set: co}, true
}

// Union implements Unioner by delegating to the inner set's Unioner under the read lock,
// unwrapping a locked other operand so two locked wrappers around optimizable sets still combine
// on the fast path. On success the result is a new LockedOrdered wrapping the inner result. It
// declines — sending the package-level Union down the generic path — when the inner set doesn't
// implement Unioner or declines the operand, or when the operand wrapper's lock is contended (see
// tryUnwrapper for why that cannot deadlock).
func (s *LockedOrdered[M]) Union(other Set[M]) (Set[M], bool) {
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
	return s.delegated(u.Union(operand))
}

// Intersection implements Intersectioner by delegating to the inner set's Intersectioner; see
// Union for the delegation and locking rules shared by all of LockedOrdered's optional-interface
// methods.
func (s *LockedOrdered[M]) Intersection(other Set[M]) (Set[M], bool) {
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
	return s.delegated(i.Intersection(operand))
}

// Difference implements Differencer by delegating to the inner set's Differencer; see Union for
// the delegation and locking rules shared by all of LockedOrdered's optional-interface methods.
func (s *LockedOrdered[M]) Difference(other Set[M]) (Set[M], bool) {
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
	return s.delegated(d.Difference(operand))
}

// SymmetricDifference implements SymmetricDifferencer by delegating to the inner set's
// SymmetricDifferencer; see Union for the delegation and locking rules shared by all of
// LockedOrdered's optional-interface methods.
func (s *LockedOrdered[M]) SymmetricDifference(other Set[M]) (Set[M], bool) {
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
	return s.delegated(sd.SymmetricDifference(operand))
}

// Equal implements Equaler by delegating to the inner set's Equaler; see Union for the delegation
// and locking rules shared by all of LockedOrdered's optional-interface methods.
func (s *LockedOrdered[M]) Equal(other Set[M]) (bool, bool) {
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
// delegation and locking rules shared by all of LockedOrdered's optional-interface methods.
func (s *LockedOrdered[M]) Disjoint(other Set[M]) (bool, bool) {
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
// delegation and locking rules shared by all of LockedOrdered's optional-interface methods.
func (s *LockedOrdered[M]) Subset(other Set[M]) (bool, bool) {
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

// Max implements Maxer by delegating to the inner set's Maxer under the read lock. It declines
// when the inner set doesn't implement Maxer or cannot answer, sending the package-level Max down
// the generic path.
func (s *LockedOrdered[M]) Max() (M, bool) {
	var zero M
	if s == nil {
		return zero, false
	}
	s.RLock()
	defer s.RUnlock()
	if mx, ok := s.set.(Maxer[M]); ok {
		return mx.Max()
	}
	return zero, false
}

// Min implements Minner by delegating to the inner set's Minner under the read lock; see Max.
func (s *LockedOrdered[M]) Min() (M, bool) {
	var zero M
	if s == nil {
		return zero, false
	}
	s.RLock()
	defer s.RUnlock()
	if mn, ok := s.set.(Minner[M]); ok {
		return mn.Min()
	}
	return zero, false
}

// String returns a string representation of the set. It returns a string of the form LockedOrderedSet[T](<elements>).
func (s *LockedOrdered[M]) String() string {
	s.RLock()
	defer s.RUnlock()

	return "Locked" + s.set.String()
}

// Value implements the driver.Valuer interface. It returns the JSON representation of the set.
func (s *LockedOrdered[M]) Value() (driver.Value, error) {
	return s.MarshalJSON()
}

// MarshalJSON implements json.Marshaler. It will marshal the set to JSON. It returns a JSON array of the elements in
// the set. If the set is empty, it returns an empty JSON array.
func (s *LockedOrdered[M]) MarshalJSON() ([]byte, error) {
	s.RLock()
	defer s.RUnlock()

	jm, ok := s.set.(json.Marshaler)
	if !ok {
		return nil, fmt.Errorf("cannot marshal set of type %T - not json.Marshaler", s.set)
	}

	d, err := jm.MarshalJSON()
	if err != nil {
		return d, fmt.Errorf("marshaling locked ordered set: %w", err)
	}

	return d, nil
}

// UnmarshalJSON implements json.Unmarshaler. It expects a JSON array of the elements in the set. If the set is empty,
// it returns an empty set. If the JSON is invalid, it returns an error.
func (s *LockedOrdered[M]) UnmarshalJSON(d []byte) error {
	s.Lock()
	defer s.Unlock()

	if s.set == nil {
		s.set = NewOrdered[M]()
	}

	um, ok := s.set.(json.Unmarshaler)
	if !ok {
		return fmt.Errorf("cannot unmarshal set of type %T - not json.Unmarshaler", s.set)
	}

	if err := um.UnmarshalJSON(d); err != nil {
		return fmt.Errorf("unmarshaling locked ordered set: %w", err)
	}
	return nil
}

// Scan implements the sql.Scanner interface. It scans the value from the database into the set. It expects a JSON array
// of the elements in the set. If the JSON is invalid an error is returned. If the value is nil an empty set is
// returned.
func (s *LockedOrdered[M]) Scan(src any) error {
	s.Lock()
	defer s.Unlock()

	if s.set == nil {
		s.set = NewOrdered[M]()
	}

	return scanValue[M](src, s.set.Clear, func(data []byte) error {
		um, ok := s.set.(json.Unmarshaler)
		if !ok {
			return fmt.Errorf("cannot unmarshal set of type %T - not json.Unmarshaler", s.set)
		}
		return um.UnmarshalJSON(data)
	})
}
