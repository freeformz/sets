package sets

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"iter"
	"math"
	"math/bits"
	"math/rand/v2"
	"slices"
)

// Integer is the element constraint for BitSet: any integer type, including named
// types via the ~ forms. It is a subset of cmp.Ordered, so every Integer type can
// also be used with the OrderedSet helpers.
type Integer interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr
}

// BitSet is an integer set backed by a dense bitmap: one bit per value in the span
// between the smallest and largest element the set currently covers. It is always
// sorted in ascending order (like SortedSet, Sort is a no-op). It is not safe for
// concurrent use; wrap it with NewLockedOrderedWrapping when concurrency is needed.
//
// BitSet's zero value is ready to use.
//
// # Memory model
//
// Memory is proportional to the SPAN of the elements (max − min), not to how many
// elements are stored: covering a span of S values costs S/8 bytes, rounded up to
// 8-byte words. Compared to Map's roughly 50 bytes per element, a BitSet is smaller
// whenever more than about 1 in every 400 values of the span is present, and is
// dramatically smaller for dense data (a full uint16 universe costs 8 KiB total).
// The flip side is pathological for sparse, far-apart values: after Add(0),
// Add(1 << 40) must allocate ~128 GiB of words. Adding an element far outside the
// current span panics if the required backing array cannot be allocated (or, on
// 32-bit platforms, cannot be represented). BitSet is therefore best suited to
// naturally bounded, reasonably dense domains: IDs, ports, enum values, rune
// tables, and the like.
//
// Storage grows to cover new elements but is never released automatically: Remove
// clears a bit without shrinking the span, so a set that once held 0 and 1<<20
// keeps ~128 KiB until Compact or Clear is called. Reserve preallocates a span up
// front to avoid regrowth; Compact reallocates to the tightest span covering the
// current elements; Clear releases the backing array entirely.
//
// Complexity (S = span in values, W = S/64 words):
//   - Add: O(1) within the current span; O(W) when extending it
//   - Remove: O(1)
//   - Contains: O(1)
//   - At, Index: O(W) (popcount scan)
//   - Iterator: O(W + N)
//   - Union, Intersection, Difference, SymmetricDifference with another BitSet
//     of the same element type: O(W) word-wise, via the package-level functions
type BitSet[M Integer] struct {
	words []uint64 // bit j of words[i] covers universe index (start+i)*64 + j
	start uint64   // universe word index of words[0]; meaningful only when len(words) > 0
	card  int
}

var _ OrderedSet[int] = new(BitSet[int])
var _ driver.Valuer = new(BitSet[int])
var _ bitwiseSet = new(BitSet[int])

// NewBitSet returns an empty *BitSet[M].
func NewBitSet[M Integer]() *BitSet[M] {
	return &BitSet[M]{}
}

// NewBitSetFrom returns a new *BitSet[M] filled with the values from the sequence.
func NewBitSetFrom[M Integer](seq iter.Seq[M]) *BitSet[M] {
	var s BitSet[M]
	for m := range seq {
		s.Add(m)
	}
	return &s
}

// NewBitSetWith returns a new *BitSet[M] with the values provided.
func NewBitSetWith[M Integer](m ...M) *BitSet[M] {
	return NewBitSetFrom(slices.Values(m))
}

// signBit is 1<<63 when M is signed and 0 when M is unsigned. XORing it onto a
// value converted to uint64 maps M's ordering onto uint64 ordering, giving every
// element a "universe index" in [0, 2^64) that sorts the same way the elements do.
func signBit[M Integer]() uint64 {
	var zero M
	if zero-1 < zero { // signed types wrap to -1; unsigned types wrap to their maximum
		return 1 << 63
	}
	return 0
}

func toUniverse[M Integer](m M) uint64 { return uint64(m) ^ signBit[M]() }

func fromUniverse[M Integer](u uint64) M { return M(u ^ signBit[M]()) }

// spanWords returns the number of words needed to cover universe word indexes lo
// through hi inclusive. It panics when that count would overflow an int or imply
// an allocation larger than the address space allows; make itself panics below
// that bound. Either way an unallocatable span fails loudly instead of wrapping.
func spanWords(lo, hi uint64) int {
	n := hi - lo + 1
	if n == 0 || n > math.MaxInt>>3 {
		panic("sets: BitSet span too large to allocate")
	}
	return int(n)
}

// grow extends the backing slice to cover universe word w. Growth allocates whole
// words for the entire new span — the memory cost documented on the type.
func (s *BitSet[M]) grow(w uint64) {
	if len(s.words) == 0 {
		s.start = w
		s.words = append(s.words, 0)
		return
	}
	end := s.start + uint64(len(s.words)) // one past the last covered word
	switch {
	case w < s.start:
		grown := make([]uint64, spanWords(w, end-1))
		copy(grown[s.start-w:], s.words)
		s.words = grown
		s.start = w
	case w >= end:
		n := spanWords(s.start, w)
		old := len(s.words)
		s.words = slices.Grow(s.words, n-old)[:n]
		clear(s.words[old:]) // spare capacity may hold stale words from a past trim
	}
}

// trim re-slices away leading and trailing zero words. It does not release memory
// (see Compact); it keeps the span tight for rank scans and word-wise operations.
func (s *BitSet[M]) trim() {
	i := 0
	for i < len(s.words) && s.words[i] == 0 {
		i++
	}
	if i == len(s.words) {
		s.words, s.start = nil, 0
		return
	}
	s.words = s.words[i:]
	s.start += uint64(i)
	j := len(s.words)
	for s.words[j-1] == 0 {
		j--
	}
	s.words = s.words[:j]
}

// Contains returns true if the set contains the element.
func (s *BitSet[M]) Contains(m M) bool {
	u := toUniverse(m)
	w := u >> 6
	if len(s.words) == 0 || w < s.start || w >= s.start+uint64(len(s.words)) {
		return false
	}
	return s.words[w-s.start]&(1<<(u&63)) != 0
}

// Clear removes all elements from the set and returns the number of elements
// removed. Unlike the other implementations, Clear releases the backing array:
// its size is proportional to the retired span, which is usually exactly the
// memory a caller clearing a BitSet wants back.
func (s *BitSet[M]) Clear() int {
	n := s.card
	s.words, s.start, s.card = nil, 0, 0
	return n
}

// Add an element to the set. Returns true if the element was added, false if it was
// already present. Adding an element outside the current span grows the backing
// array to cover it (see the type comment for the memory implications) and panics
// if that allocation is impossible.
func (s *BitSet[M]) Add(m M) bool {
	u := toUniverse(m)
	s.grow(u >> 6)
	i := (u >> 6) - s.start
	bit := uint64(1) << (u & 63)
	if s.words[i]&bit != 0 {
		return false
	}
	s.words[i] |= bit
	s.card++
	return true
}

// Remove an element from the set. Returns true if the element was removed, false if
// it was not present. Remove never shrinks the backing array; call Compact to
// release memory after removing span-extreme elements.
func (s *BitSet[M]) Remove(m M) bool {
	u := toUniverse(m)
	w := u >> 6
	if len(s.words) == 0 || w < s.start || w >= s.start+uint64(len(s.words)) {
		return false
	}
	bit := uint64(1) << (u & 63)
	if s.words[w-s.start]&bit == 0 {
		return false
	}
	s.words[w-s.start] &^= bit
	s.card--
	return true
}

// Cardinality returns the number of elements in the set.
func (s *BitSet[M]) Cardinality() int {
	if s == nil {
		return 0
	}
	return s.card
}

// Iterator yields all elements in the set in ascending order.
func (s *BitSet[M]) Iterator(yield func(M) bool) {
	for i, w := range s.words {
		base := (s.start + uint64(i)) << 6
		for w != 0 {
			b := bits.TrailingZeros64(w)
			if !yield(fromUniverse[M](base | uint64(b))) {
				return
			}
			w &= w - 1 // clear the lowest set bit
		}
	}
}

// Clone returns a copy of the set. The underlying type is the same as the original set.
func (s *BitSet[M]) Clone() Set[M] {
	return &BitSet[M]{words: slices.Clone(s.words), start: s.start, card: s.card}
}

// Ordered iteration yields the index and value of each element in the set in ascending order.
func (s *BitSet[M]) Ordered(yield func(int, M) bool) {
	var n int
	for i, w := range s.words {
		base := (s.start + uint64(i)) << 6
		for w != 0 {
			b := bits.TrailingZeros64(w)
			if !yield(n, fromUniverse[M](base|uint64(b))) {
				return
			}
			n++
			w &= w - 1
		}
	}
}

// Backwards iteration yields the index and value of each element in the set in descending order.
func (s *BitSet[M]) Backwards(yield func(int, M) bool) {
	n := s.card - 1
	for i := len(s.words) - 1; i >= 0; i-- {
		w := s.words[i]
		base := (s.start + uint64(i)) << 6
		for w != 0 {
			b := 63 - bits.LeadingZeros64(w)
			if !yield(n, fromUniverse[M](base|uint64(b))) {
				return
			}
			n--
			w &^= 1 << uint(b)
		}
	}
}

// Range returns an iterator over the elements v for which lo <= v <= hi, in
// ascending order. A call costs O(W) for the words overlapping [lo, hi] plus the
// number of elements yielded. If lo > hi the iterator yields nothing.
func (s *BitSet[M]) Range(lo, hi M) iter.Seq[M] {
	return func(yield func(M) bool) {
		if hi < lo || len(s.words) == 0 {
			return
		}
		ul, uh := toUniverse(lo), toUniverse(hi)
		// clamp to the covered span; the -1 wraps correctly when the span
		// reaches the top of the universe
		if first := s.start << 6; ul < first {
			ul = first
		}
		if last := ((s.start + uint64(len(s.words))) << 6) - 1; uh > last {
			uh = last
		}
		if uh < ul {
			return
		}
		for wi := ul >> 6; wi <= uh>>6; wi++ {
			w := s.words[wi-s.start]
			if wi == ul>>6 {
				w &= ^uint64(0) << (ul & 63)
			}
			if wi == uh>>6 {
				w &= ^uint64(0) >> (63 - uh&63)
			}
			base := wi << 6
			for w != 0 {
				b := bits.TrailingZeros64(w)
				if !yield(fromUniverse[M](base | uint64(b))) {
					return
				}
				w &= w - 1
			}
		}
	}
}

// Reserve grows the backing array to cover the inclusive element range [lo, hi] in
// a single allocation, so subsequent Adds within the range never regrow. It does
// not add any elements. If lo > hi, Reserve does nothing.
func (s *BitSet[M]) Reserve(lo, hi M) {
	if hi < lo {
		return
	}
	s.grow(toUniverse(lo) >> 6)
	s.grow(toUniverse(hi) >> 6)
}

// Compact reallocates the backing array to the smallest size covering the current
// elements. Use it to release memory after Remove has shrunk the occupied span or
// after a Reserve that turned out to be too generous. If the set is empty the
// backing array is released entirely.
func (s *BitSet[M]) Compact() {
	s.trim()
	if s.words != nil {
		s.words = slices.Clip(slices.Clone(s.words))
	}
}

// NewEmptyOrdered returns a new empty ordered set of the same underlying type.
func (s *BitSet[M]) NewEmptyOrdered() OrderedSet[M] {
	return NewBitSet[M]()
}

// NewEmpty returns a new empty set of the same underlying type.
func (s *BitSet[M]) NewEmpty() Set[M] {
	return NewBitSet[M]()
}

// Pop removes and returns a random element from the set. If the set is empty, it
// returns the zero value of M and false. Selecting the element is a rank scan, so
// Pop is O(W).
func (s *BitSet[M]) Pop() (M, bool) {
	if s.card == 0 {
		var m M
		return m, false
	}
	m, _ := s.At(rand.IntN(s.card))
	s.Remove(m)
	return m, true
}

// Sort is a no-op: the set is always sorted in ascending order.
func (s *BitSet[M]) Sort() {}

// At returns the element at the index. If the index is out of bounds, the second
// return value is false.
func (s *BitSet[M]) At(i int) (M, bool) {
	if i < 0 || i >= s.card {
		var zero M
		return zero, false
	}
	rank := i
	for wi, w := range s.words {
		n := bits.OnesCount64(w)
		if rank >= n {
			rank -= n
			continue
		}
		for ; rank > 0; rank-- {
			w &= w - 1
		}
		b := bits.TrailingZeros64(w)
		return fromUniverse[M]((s.start+uint64(wi))<<6 | uint64(b)), true
	}
	var zero M // unreachable while card matches the stored bits
	return zero, false
}

// Index returns the index of the element in the set, or -1 if not present.
func (s *BitSet[M]) Index(m M) int {
	u := toUniverse(m)
	w := u >> 6
	if len(s.words) == 0 || w < s.start || w >= s.start+uint64(len(s.words)) {
		return -1
	}
	bit := uint64(1) << (u & 63)
	if s.words[w-s.start]&bit == 0 {
		return -1
	}
	rank := bits.OnesCount64(s.words[w-s.start] & (bit - 1))
	for _, pw := range s.words[:w-s.start] {
		rank += bits.OnesCount64(pw)
	}
	return rank
}

func (s *BitSet[M]) elements() []M {
	out := make([]M, 0, s.card)
	for m := range s.Iterator {
		out = append(out, m)
	}
	return out
}

// String returns a string representation of the set. It returns a string of the form BitSet[T](<elements>).
func (s *BitSet[M]) String() string {
	var m M
	return fmt.Sprintf("BitSet[%T](%v)", m, s.elements())
}

// Value implements the driver.Valuer interface. It returns the JSON representation of the set.
func (s *BitSet[M]) Value() (driver.Value, error) {
	return s.MarshalJSON()
}

// MarshalJSON implements json.Marshaler. It will marshal the set into a JSON array
// of the elements in the set in ascending order. If the set is empty an empty JSON
// array is returned.
func (s *BitSet[M]) MarshalJSON() ([]byte, error) {
	if s.card == 0 {
		return []byte("[]"), nil
	}
	d, err := json.Marshal(s.elements())
	if err != nil {
		return d, fmt.Errorf("marshaling bit set: %w", err)
	}
	return d, nil
}

// UnmarshalJSON implements json.Unmarshaler. It expects a JSON array of the
// elements in the set. The elements do not need to be sorted or unique. If the
// JSON is invalid, it returns an error and the set is left unchanged.
func (s *BitSet[M]) UnmarshalJSON(d []byte) error {
	var t []M
	if err := json.Unmarshal(d, &t); err != nil {
		return fmt.Errorf("unmarshaling bit set: %w", err)
	}
	s.Clear()
	for _, m := range t {
		s.Add(m)
	}
	return nil
}

// Scan implements the sql.Scanner interface. It scans the value from the database into the set. It expects a JSON array
// of the elements in the set. If the JSON is invalid an error is returned. If the value is nil an empty set is
// returned.
func (s *BitSet[M]) Scan(src any) error {
	return scanValue[M](src, s.Clear, s.UnmarshalJSON)
}

// bitwiseSet is implemented by BitSet so the package-level set functions can use
// word-wise operations when both operands are BitSets of the same element type.
// Each method reports false when other is not a compatible BitSet, in which case
// the caller falls back to the generic element-wise path. When ok is true the
// returned value is a Set of the receiver's element type. The interface is
// deliberately non-generic: inside a func constrained only by comparable the
// *BitSet[K] type cannot even be named, so the concrete type check has to happen
// here, where M is known to be an Integer.
type bitwiseSet interface {
	bitwiseUnion(other any) (any, bool)
	bitwiseIntersection(other any) (any, bool)
	bitwiseDifference(other any) (any, bool)
	bitwiseSymmetricDifference(other any) (any, bool)
}

func (s *BitSet[M]) bitwiseUnion(other any) (any, bool) {
	o, ok := other.(*BitSet[M])
	if !ok {
		return nil, false
	}
	return s.union(o), true
}

func (s *BitSet[M]) bitwiseIntersection(other any) (any, bool) {
	o, ok := other.(*BitSet[M])
	if !ok {
		return nil, false
	}
	return s.intersect(o), true
}

func (s *BitSet[M]) bitwiseDifference(other any) (any, bool) {
	o, ok := other.(*BitSet[M])
	if !ok {
		return nil, false
	}
	return s.difference(o), true
}

func (s *BitSet[M]) bitwiseSymmetricDifference(other any) (any, bool) {
	o, ok := other.(*BitSet[M])
	if !ok {
		return nil, false
	}
	return s.symmetricDifference(o), true
}

func (s *BitSet[M]) recount() {
	s.card = 0
	for _, w := range s.words {
		s.card += bits.OnesCount64(w)
	}
}

// merge returns a new BitSet covering the combined span of s and o, with each word
// initialized to s's bits combined with o's bits by op. The combined span can be
// much larger than either input's when the two sets are far apart (see the type
// comment on memory).
func (s *BitSet[M]) merge(o *BitSet[M], op func(a, b uint64) uint64) *BitSet[M] {
	if len(s.words) == 0 {
		return o.Clone().(*BitSet[M])
	}
	if len(o.words) == 0 {
		return s.Clone().(*BitSet[M])
	}
	start := min(s.start, o.start)
	end := max(s.start+uint64(len(s.words)), o.start+uint64(len(o.words)))
	c := &BitSet[M]{words: make([]uint64, spanWords(start, end-1)), start: start}
	copy(c.words[s.start-start:], s.words)
	for i, w := range o.words {
		j := o.start - start + uint64(i)
		c.words[j] = op(c.words[j], w)
	}
	c.recount()
	c.trim()
	return c
}

func (s *BitSet[M]) union(o *BitSet[M]) *BitSet[M] {
	return s.merge(o, func(a, b uint64) uint64 { return a | b })
}

func (s *BitSet[M]) symmetricDifference(o *BitSet[M]) *BitSet[M] {
	return s.merge(o, func(a, b uint64) uint64 { return a ^ b })
}

func (s *BitSet[M]) intersect(o *BitSet[M]) *BitSet[M] {
	c := &BitSet[M]{}
	if len(s.words) == 0 || len(o.words) == 0 {
		return c
	}
	start := max(s.start, o.start)
	end := min(s.start+uint64(len(s.words)), o.start+uint64(len(o.words)))
	if end <= start {
		return c
	}
	c.words = make([]uint64, end-start) // bounded by the smaller input, no overflow
	c.start = start
	for i := range c.words {
		c.words[i] = s.words[start-s.start+uint64(i)] & o.words[start-o.start+uint64(i)]
	}
	c.recount()
	c.trim()
	return c
}

func (s *BitSet[M]) difference(o *BitSet[M]) *BitSet[M] {
	c := s.Clone().(*BitSet[M])
	if len(c.words) == 0 || len(o.words) == 0 {
		return c
	}
	start := max(s.start, o.start)
	end := min(s.start+uint64(len(s.words)), o.start+uint64(len(o.words)))
	for w := start; w < end; w++ {
		c.words[w-c.start] &^= o.words[w-o.start]
	}
	c.recount()
	c.trim()
	return c
}
