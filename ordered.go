package sets

import (
	"cmp"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"iter"
	"slices"
)

// Ordered maintains the order that elements were added in. It uses a gap buffer with a Fenwick tree
// (binary indexed tree) to provide O(log N) Remove, At, and Index operations while keeping Add and
// Contains at O(1) amortized and O(1) respectively. It is not safe for concurrent use.
//
// Complexity:
//   - Add: O(1) amortized
//   - Remove: O(log N) amortized
//   - Contains: O(1)
//   - At: O(log N)
//   - Index: O(log N)
//   - Iterator: O(N)
type Ordered[M cmp.Ordered] struct {
	idx   map[M]int // element -> physical slot index
	slots []M       // physical slots (may contain gaps from removals)
	alive []bool    // slot occupancy bitmap
	bit   []int     // Fenwick tree (1-indexed) for prefix sums of alive slots
	count int       // number of alive elements
}

var _ OrderedSet[int] = new(Ordered[int])
var _ driver.Valuer = new(Ordered[int])

// NewOrdered returns an empty *Ordered[M].
func NewOrdered[M cmp.Ordered]() *Ordered[M] {
	return &Ordered[M]{
		idx:   make(map[M]int),
		slots: make([]M, 0),
		alive: make([]bool, 0),
		bit:   make([]int, 1), // bit[0] is unused sentinel
	}
}

// NewOrderedFrom returns a new *Ordered[M] filled with the values from the sequence.
func NewOrderedFrom[M cmp.Ordered](seq iter.Seq[M]) *Ordered[M] {
	s := NewOrdered[M]()
	for x := range seq {
		s.Add(x)
	}
	return s
}

// NewOrderedWith returns a new *Ordered[M] with the values provided.
func NewOrderedWith[M cmp.Ordered](m ...M) *Ordered[M] {
	return NewOrderedFrom(slices.Values(m))
}

// --- Fenwick tree (binary indexed tree) operations ---

func (s *Ordered[M]) bitUpdate(i, delta int) {
	for i++; i < len(s.bit); i += i & (-i) {
		s.bit[i] += delta
	}
}

func (s *Ordered[M]) bitQuery(i int) int {
	var sum int
	for i++; i > 0; i -= i & (-i) {
		sum += s.bit[i]
	}
	return sum
}

// bitFindKth finds the physical slot index of the k-th alive element (0-indexed k).
func (s *Ordered[M]) bitFindKth(k int) int {
	k++ // convert to 1-indexed count
	var pos int
	bitmask := 1
	for bitmask < len(s.bit) {
		bitmask <<= 1
	}
	bitmask >>= 1
	for bitmask > 0 {
		next := pos + bitmask
		if next < len(s.bit) && s.bit[next] < k {
			k -= s.bit[next]
			pos = next
		}
		bitmask >>= 1
	}
	return pos
}

// rebuildBIT reconstructs the Fenwick tree from the alive bitmap.
// When growing, it doubles capacity to amortize rebuild cost.
func (s *Ordered[M]) rebuildBIT() {
	needed := len(s.slots) + 1
	newCap := needed
	if needed > len(s.bit) {
		newCap = max(needed, len(s.bit)*2)
	}
	newCap = max(newCap, 2)
	s.bit = make([]int, newCap)

	// O(N) Fenwick tree construction
	for i := range s.slots {
		if s.alive[i] {
			s.bit[i+1] = 1
		}
	}
	for i := 1; i < len(s.bit); i++ {
		j := i + (i & (-i))
		if j < len(s.bit) {
			s.bit[j] += s.bit[i]
		}
	}
}

// compact removes gaps by rebuilding the slots, alive, and bit arrays.
func (s *Ordered[M]) compact() {
	if s.count == len(s.slots) {
		return
	}
	newSlots := make([]M, 0, s.count)
	newAlive := make([]bool, 0, s.count)
	for i, v := range s.slots {
		if s.alive[i] {
			s.idx[v] = len(newSlots)
			newSlots = append(newSlots, v)
			newAlive = append(newAlive, true)
		}
	}
	s.slots = newSlots
	s.alive = newAlive
	s.rebuildBIT()
}

func (s *Ordered[M]) maybeCompact() {
	if s.count > 0 && len(s.slots) > 2*s.count {
		s.compact()
	}
}

// elements returns a slice of all alive elements in insertion order.
func (s *Ordered[M]) elements() []M {
	out := make([]M, 0, s.count)
	for i, v := range s.slots {
		if s.alive[i] {
			out = append(out, v)
		}
	}
	return out
}

// --- Set interface ---

// Contains returns true if the set contains the element.
func (s *Ordered[M]) Contains(m M) bool {
	_, ok := s.idx[m]
	return ok
}

// Clear the set and returns the number of elements removed.
func (s *Ordered[M]) Clear() int {
	n := s.count
	if s.idx == nil {
		s.idx = make(map[M]int)
	} else {
		clear(s.idx)
	}
	if s.slots == nil {
		s.slots = make([]M, 0)
	} else {
		clear(s.slots) // zero retained backing array so element values can be collected
		s.slots = s.slots[:0]
	}
	if s.alive == nil {
		s.alive = make([]bool, 0)
	} else {
		s.alive = s.alive[:0]
	}
	s.bit = make([]int, 1)
	s.count = 0
	return n
}

// Add an element to the set. Returns true if the element was added, false if it was already present. Elements are added
// to the end of the ordered set.
func (s *Ordered[M]) Add(m M) bool {
	if s.Contains(m) {
		return false
	}
	p := len(s.slots)
	s.slots = append(s.slots, m)
	s.alive = append(s.alive, true)
	s.idx[m] = p
	s.count++
	if p+2 > len(s.bit) {
		s.rebuildBIT()
	} else {
		s.bitUpdate(p, 1)
	}
	return true
}

// Remove an element from the set. Returns true if the element was removed, false if it was not present.
func (s *Ordered[M]) Remove(m M) bool {
	p, ok := s.idx[m]
	if !ok {
		return false
	}
	s.alive[p] = false
	s.bitUpdate(p, -1)
	delete(s.idx, m)
	s.count--
	s.maybeCompact()
	return true
}

// Cardinality returns the number of elements in the set.
func (s *Ordered[M]) Cardinality() int {
	if s == nil {
		return 0
	}
	return s.count
}

// Iterator yields all elements in the set in order.
func (s *Ordered[M]) Iterator(yield func(M) bool) {
	for i, v := range s.slots {
		if s.alive[i] {
			if !yield(v) {
				return
			}
		}
	}
}

// Clone returns a copy of the set. The underlying type is the same as the original set.
func (s *Ordered[M]) Clone() Set[M] {
	// bulk copy (compacted) instead of re-adding element by element, which would
	// pay a map insert and Fenwick tree update per element
	c := &Ordered[M]{
		idx:   make(map[M]int, s.count),
		slots: s.elements(),
		alive: make([]bool, s.count),
		count: s.count,
	}
	for i, v := range c.slots {
		c.alive[i] = true
		c.idx[v] = i
	}
	c.rebuildBIT()
	return c
}

// Ordered iteration yields the index and value of each element in the set in order.
func (s *Ordered[M]) Ordered(yield func(int, M) bool) {
	var j int
	for i, v := range s.slots {
		if s.alive[i] {
			if !yield(j, v) {
				return
			}
			j++
		}
	}
}

// Backwards iteration yields the index and value of each element in the set in reverse order.
func (s *Ordered[M]) Backwards(yield func(int, M) bool) {
	j := s.count - 1
	for i := len(s.slots) - 1; i >= 0; i-- {
		if s.alive[i] {
			if !yield(j, s.slots[i]) {
				return
			}
			j--
		}
	}
}

// NewEmptyOrdered returns a new empty ordered set of the same underlying type.
func (s *Ordered[M]) NewEmptyOrdered() OrderedSet[M] {
	return NewOrdered[M]()
}

// NewEmpty returns a new empty set of the same underlying type.
func (s *Ordered[M]) NewEmpty() Set[M] {
	return NewOrdered[M]()
}

// Pop removes and returns an element from the set. If the set is empty, it returns the zero value of M and false.
func (s *Ordered[M]) Pop() (M, bool) {
	for k := range s.idx {
		s.Remove(k)
		return k, true
	}
	var m M
	return m, false
}

// Sort the set in ascending order.
func (s *Ordered[M]) Sort() {
	s.compact()
	slices.Sort(s.slots)
	for i, v := range s.slots {
		s.idx[v] = i
	}
	// BIT is all-ones after compact; sort doesn't change alive status.
}

// At returns the element at the index. If the index is out of bounds, the second return value is false.
func (s *Ordered[M]) At(i int) (M, bool) {
	var zero M
	if i < 0 || i >= s.count {
		return zero, false
	}
	p := s.bitFindKth(i)
	return s.slots[p], true
}

// Index returns the index of the element in the set, or -1 if not present.
func (s *Ordered[M]) Index(m M) int {
	p, ok := s.idx[m]
	if !ok {
		return -1
	}
	return s.bitQuery(p) - 1
}

// String returns a string representation of the set. It returns a string of the form OrderedSet[T](<elements>).
func (s *Ordered[M]) String() string {
	var m M
	return fmt.Sprintf("OrderedSet[%T](%v)", m, s.elements())
}

// Value implements the driver.Valuer interface. It returns the JSON representation of the set.
func (s *Ordered[M]) Value() (driver.Value, error) {
	return s.MarshalJSON()
}

// MarshalJSON implements json.Marshaler. It will marshal the set into a JSON array of the elements in the set. If the
// set is empty an empty JSON array is returned.
func (s *Ordered[M]) MarshalJSON() ([]byte, error) {
	vals := s.elements()
	if len(vals) == 0 {
		return []byte("[]"), nil
	}

	d, err := json.Marshal(vals)
	if err != nil {
		return d, fmt.Errorf("marshaling ordered set: %w", err)
	}
	return d, nil
}

// UnmarshalJSON implements json.Unmarshaler. It expects a JSON array of the elements in the set. If the set is empty,
// it returns an empty set. If the JSON is invalid, it returns an error.
func (s *Ordered[M]) UnmarshalJSON(d []byte) error {
	t := make([]M, 0)
	if err := json.Unmarshal(d, &t); err != nil {
		return fmt.Errorf("unmarshaling ordered set: %w", err)
	}

	s.Clear()
	for _, v := range t {
		s.Add(v)
	}

	return nil
}

// Scan implements the sql.Scanner interface. It scans the value from the database into the set. It expects a JSON array
// of the elements in the set. If the JSON is invalid an error is returned. If the value is nil an empty set is
// returned.
func (s *Ordered[M]) Scan(src any) error {
	return scanValue[M](src, s.Clear, s.UnmarshalJSON)
}
