package sets

import (
	"encoding/json"
	"fmt"
	"slices"

	"github.com/google/go-cmp/cmp"
)

func ExampleSet() {
	ints := New[int]()
	ints.Add(5)
	ints.Add(1)
	ints.Add(9)

	if !ints.Add(1) { // 1 is already present, returns false
		fmt.Println("1 was not added again")
	}

	if ints.Add(33) { // 33 is not present, returns true
		fmt.Println("33 was added")
	}

	if ints.Cardinality() == 3 {
		fmt.Println("ints has 3 elements")
	}
	other := ints.Clone()
	if other.Cardinality() == 3 {
		fmt.Println("Cloned set has 3 elements")
	}

	if ints.Contains(5) {
		fmt.Println("5 is present")
	}

	if !ints.Contains(2) {
		fmt.Println("2 is not present")
	}

	if !ints.Remove(2) { // 2 is not present, returns false
		fmt.Println("2 was not removed")
	}

	if ints.Remove(5) { // 5 is present, returns true
		fmt.Println("5 was removed")
	}

	if _, ok := ints.Pop(); ok { // 1 || 33 || 9 removed
		// not printing since random
		fmt.Println("Popped a number")
	}

	if x := ints.Clear(); x == 2 {
		fmt.Println("Clear removed all remaining elements")
	}

	// Sets aren't ordered, so collect into a slice and sort
	// using iterator
	items := slices.Collect(other.Iterator)
	slices.Sort(items)
	for _, i := range items {
		fmt.Println(i)
	}

	other = ints.NewEmpty()
	if other.Cardinality() == 0 {
		fmt.Println("other is empty")
	}

	other.Add(0)
	fmt.Println(other.String())

	// Output:
	// 1 was not added again
	// 33 was added
	// 5 is present
	// 2 is not present
	// 2 was not removed
	// 5 was removed
	// Popped a number
	// Clear removed all remaining elements
	// 1
	// 5
	// 9
	// 33
	// other is empty
	// Set[int]([0])
}

func ExampleOrderedSet() {
	ints := NewOrdered[int]()
	ints.Add(5)
	ints.Add(3)

	// adds 2, 4, 1 in order
	AppendSeq(ints, slices.Values([]int{2, 4, 1}))
	// adds 6 as it's the only new element
	AppendSeq(ints, slices.Values([]int{5, 6, 1}))

	// 0,5 1,3 2,2 3,4 4,1 5,6
	for idx, i := range ints.Ordered {
		fmt.Println(idx, i)
	}

	// 5,6 4,1 3,4 2,2 1,3 0,5
	for idx, i := range ints.Backwards {
		fmt.Println(idx, i)
	}

	if v, ok := ints.At(1); v == 3 && ok {
		fmt.Println("3 is at position 1")
	}

	if ints.Index(3) == 1 {
		fmt.Println("3 is at index 1")
	}

	if ints.Index(100) == -1 {
		fmt.Println("100 is not present")
	}

	ints.Sort()
	// 1 2 3 4 5 6
	for i := range ints.Iterator {
		fmt.Println(i)
	}
	// Output:
	// 0 5
	// 1 3
	// 2 2
	// 3 4
	// 4 1
	// 5 6
	// 5 6
	// 4 1
	// 3 4
	// 2 2
	// 1 3
	// 0 5
	// 3 is at position 1
	// 3 is at index 1
	// 100 is not present
	// 1
	// 2
	// 3
	// 4
	// 5
	// 6
}

func ExampleElements() {
	ints := NewWith(5, 3, 2)

	// []T is returned
	elements := Elements(ints)
	for _, i := range elements {
		fmt.Println(i)
	}
	// Unsorted output:
	// 2
	// 3
	// 5
}

func ExampleAppendSeq() {
	ints := NewWith(5, 3)

	// adds 2,4,1 to the set since 5 and 3 already exist
	added := AppendSeq(ints, slices.Values([]int{5, 3, 2, 4, 1}))
	fmt.Println(added)
	// Output: 3
}

func ExampleRemoveSeq() {
	ints := NewWith(5, 3, 2)

	// removes 2 from the set since 5 and 3 exist
	removed := RemoveSeq(ints, slices.Values([]int{2, 4, 1}))
	fmt.Println(removed)
	// Output: 1
}

func ExampleUnion() {
	a := NewWith(5, 3)
	b := NewWith(3, 2)

	c := Union(a, b)
	out := make([]int, 0, c.Cardinality())
	for i := range c.Iterator {
		out = append(out, i)
	}
	slices.Sort(out)
	for _, i := range out {
		fmt.Println(i)
	}
	// Output:
	// 2
	// 3
	// 5
}

func ExampleIntersection() {
	a := NewWith(5, 3)
	b := NewWith(3, 2)

	c := Intersection(a, b)
	out := make([]int, 0, c.Cardinality())
	for i := range c.Iterator {
		out = append(out, i)
	}
	for _, i := range out {
		fmt.Println(i)
	}
	// Output:
	// 3
}

func ExampleDifference() {
	a := NewWith(5, 3)
	b := NewWith(3, 2)

	c := Difference(a, b)
	out := make([]int, 0, c.Cardinality())
	for i := range c.Iterator {
		out = append(out, i)
	}
	for _, i := range out {
		fmt.Println(i)
	}
	// Output:
	// 5
}

func ExampleSymmetricDifference() {
	a := NewWith(5, 3)
	b := NewWith(3, 2)

	c := SymmetricDifference(a, b)
	for i := range c.Iterator {
		fmt.Println(i)
	}
	// Unordered output:
	// 2
	// 5
}

func ExampleSubset() {
	a := NewWith(5, 3)
	b := NewWith(5, 3, 2)

	if Subset(a, b) {
		fmt.Println("a is a subset of b")
	}

	if !Subset(b, a) {
		fmt.Println("b is not a subset of a")
	}
	// Output:
	// a is a subset of b
	// b is not a subset of a
}

func ExampleSuperset() {
	a := NewWith(5, 3)
	b := NewWith(5, 3, 2)

	if !Superset(a, b) {
		fmt.Println("a is not a superset of b")
	}

	if Superset(b, a) {
		fmt.Println("b is a superset of a")
	}
	// Output:
	// a is not a superset of b
	// b is a superset of a
}

func ExampleEqual() {
	a := NewWith(5, 3)
	b := NewWith(5, 3)

	if Equal(a, b) {
		fmt.Println("a and b are equal")
	}

	// how to compare two sets using [cmp.Diff] - pass `cmp.Comparer(Equal[T])`
	if diff := cmp.Diff(a, b, cmp.Comparer(Equal[int])); diff != "" {
		fmt.Println(diff)
	} else {
		fmt.Println("no diff")
	}

	b.Add(2)
	if !Equal(a, b) {
		fmt.Println("a and b are not equal now")
	}
	// Output:
	// a and b are equal
	// no diff
	// a and b are not equal now
}

func ExampleContainsSeq() {
	ints := New[int]()
	if ContainsSeq(ints, slices.Values([]int{})) {
		fmt.Println("Empty set contains empty sequence")
	}

	ints.Add(5)
	ints.Add(3)
	ints.Add(2)

	if ContainsSeq(ints, slices.Values([]int{3, 5})) {
		fmt.Println("3 and 5 are present")
	}

	if !ContainsSeq(ints, slices.Values([]int{3, 5, 6})) {
		fmt.Println("6 is not present")
	}
	// Output:
	// Empty set contains empty sequence
	// 3 and 5 are present
	// 6 is not present
}

func ExampleDisjoint() {
	a := NewWith(5, 3)
	b := NewWith(2, 4)

	if Disjoint(a, b) {
		fmt.Println("a and b are disjoint")
	}

	b.Add(3)
	if !Disjoint(a, b) {
		fmt.Println("a and b are not disjoint now")
	}
	// Output:
	// a and b are disjoint
	// a and b are not disjoint now
}

func ExampleEqualOrdered() {
	a := NewOrderedWith(5, 3, 1)
	b := NewOrderedWith(5, 3, 1)

	if EqualOrdered(a, b) {
		fmt.Println("a and b are equal")
	}

	// how to compare two ordered sets using [cmp.Diff] - pass `cmp.Comparer(EqualOrdered[T])`
	if diff := cmp.Diff(a, b, cmp.Comparer(EqualOrdered[int])); diff != "" {
		fmt.Println(diff)
	} else {
		fmt.Println("no ordered diff")
	}

	b.Add(2)
	if !EqualOrdered(a, b) {
		fmt.Println("a and b are not equal now")
	}
	// Output:
	// a and b are equal
	// no ordered diff
	// a and b are not equal now
}

func ExampleMin() {
	ints := NewWith(3, 2, 5)

	min := Min(ints)
	fmt.Println(min)
	// Output: 2
}

func ExampleMax() {
	ints := NewWith(3, 5, 2)

	max := Max(ints)
	fmt.Println(max)
	// Output: 5
}

func ExampleIsSorted() {
	ints := NewOrderedWith(2, 3, 5)

	if IsSorted(ints) {
		fmt.Println("ints is sorted")
	}

	ints.Add(4)
	if !IsSorted(ints) {
		fmt.Println("ints is not sorted now")
	}

	ints.Sort()
	if IsSorted(ints) {
		fmt.Println("ints is sorted")
	}
	// Output:
	// ints is sorted
	// ints is not sorted now
	// ints is sorted
}

func ExampleReverse() {
	ints := NewOrderedWith(2, 3, 5)

	reversed := Reverse(ints)
	for i := range reversed.Iterator {
		fmt.Println(i)
	}
	// Output:
	// 5
	// 3
	// 2
}

func ExampleSorted() {
	ints := NewOrderedWith(2, 5, 3)

	sorted := Sorted(ints)
	for i := range sorted.Iterator {
		fmt.Println(i)
	}
	// Output:
	// 2
	// 3
	// 5
}

func ExampleChunk() {
	ints := NewOrderedWith(1, 2, 3, 4, 5)

	// this example test won't work with an unordered set
	// as the order of the chunks is based on the order of
	// the set elements, which isn't stable in an unordered set
	chunks := Chunk(ints, 2)
	for chunk := range chunks {
		fmt.Println(chunk)
		for v := range chunk.Iterator {
			fmt.Println(v)
		}
	}
	// Output:
	// OrderedSet[int]([1 2])
	// 1
	// 2
	// OrderedSet[int]([3 4])
	// 3
	// 4
	// OrderedSet[int]([5])
	// 5
}

func ExampleIter2() {
	ints := NewOrderedWith(1, 2, 3, 4, 5)

	// this example test won't work with an unordered set
	// as the iter2 function relies on the order of the set
	// elements, which isn't stable in an unordered set
	for i, v := range Iter2(ints.Iterator) {
		fmt.Println("idx:", i, "value:", v)
	}

	// Output:
	// idx: 0 value: 1
	// idx: 1 value: 2
	// idx: 2 value: 3
	// idx: 3 value: 4
	// idx: 4 value: 5
}

func Example_json() {
	set := NewOrderedWith(1.0, 1.2, 1.3, 1.4, 1.5)
	b, err := json.Marshal(set)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(string(b))

	set2 := NewOrdered[float32]()
	if err := json.Unmarshal(b, &set2); err != nil {
		fmt.Println(err)
	}
	fmt.Println(set2)

	// Output:
	// [1,1.2,1.3,1.4,1.5]
	// OrderedSet[float32]([1 1.2 1.3 1.4 1.5])
}

func ExampleNewWith() {
	set := NewWith("a", "b", "c", "b")
	fmt.Println(set.Cardinality())

	// Output: 3
}

func ExampleNewLockedWith() {
	set := NewLockedWith("a", "b", "c", "b")
	fmt.Println(set.Cardinality())

	// Output: 3
}

func ExampleNewOrderedWith() {
	set := NewOrderedWith("a", "b", "c", "b")
	fmt.Println(set.Cardinality())

	for i := range set.Iterator {
		fmt.Println(i)
	}

	// Output:
	// 3
	// a
	// b
	// c
}

func ExampleNewLockedOrderedWith() {
	set := NewLockedOrderedWith("a", "b", "c", "b")
	fmt.Println(set.Cardinality())

	for i := range set.Iterator {
		fmt.Println(i)
	}

	// Output:
	// 3
	// a
	// b
	// c
}

func ExampleNewSyncMapWith() {
	set := NewSyncMapWith("a", "b", "c", "b")
	fmt.Println(set.Cardinality())

	// Output: 3
}

func ExampleNew() {
	set := New[string]()
	set.Add("a")
	set.Add("b")
	set.Add("c")
	set.Add("b")
	fmt.Println(set.Cardinality())

	// Output: 3
}

func ExampleNewLocked() {
	set := NewLocked[string]()
	set.Add("a")
	set.Add("b")
	set.Add("c")
	set.Add("b")
	fmt.Println(set.Cardinality())

	// Output: 3
}

func ExampleNewOrdered() {
	set := NewOrdered[string]()
	set.Add("a")
	set.Add("b")
	set.Add("c")
	set.Add("b")
	fmt.Println(set.Cardinality())

	for i := range set.Iterator {
		fmt.Println(i)
	}

	// Output:
	// 3
	// a
	// b
	// c
}

func ExampleNewLockedOrdered() {
	set := NewLockedOrdered[string]()
	set.Add("a")
	set.Add("b")
	set.Add("c")
	set.Add("b")
	fmt.Println(set.Cardinality())

	for i := range set.Iterator {
		fmt.Println(i)
	}

	// Output:
	// 3
	// a
	// b
	// c
}

func ExampleNewSyncMap() {
	set := NewSyncMap[string]()
	set.Add("a")
	set.Add("b")
	set.Add("c")
	set.Add("b")
	fmt.Println(set.Cardinality())

	// Output: 3
}

func ExampleNewFrom() {
	m := []string{"a", "b", "c", "b"}
	set := NewFrom(slices.Values(m))
	fmt.Println(set.Cardinality())

	// Output: 3
}

func ExampleNewLockedFrom() {
	m := []string{"a", "b", "c", "b"}
	set := NewLockedFrom(slices.Values(m))
	fmt.Println(set.Cardinality())

	// Output: 3
}

func ExampleNewOrderedFrom() {
	m := []string{"a", "b", "c", "b"}
	set := NewOrderedFrom(slices.Values(m))
	fmt.Println(set.Cardinality())

	for i := range set.Iterator {
		fmt.Println(i)
	}

	// Output:
	// 3
	// a
	// b
	// c
}

func ExampleNewLockedOrderedFrom() {
	m := []string{"a", "b", "c", "b"}
	set := NewLockedOrderedFrom(slices.Values(m))
	fmt.Println(set.Cardinality())

	for i := range set.Iterator {
		fmt.Println(i)
	}

	// Output:
	// 3
	// a
	// b
	// c
}

func ExampleNewSyncMapFrom() {
	m := []string{"a", "b", "c", "b"}
	set := NewSyncMapFrom(slices.Values(m))
	fmt.Println(set.Cardinality())

	// Output: 3
}

func ExampleNewLockedWrapping() {
	set := NewWith("a", "b", "c", "b")

	wrapped := NewLockedWrapping(set)
	// wrapped is safe for concurrent use
	fmt.Println(wrapped.Cardinality())

	// Output: 3
}

func ExampleNewLockedOrderedWrapping() {
	set := NewOrderedWith("a", "b", "c", "b")

	wrapped := NewLockedOrderedWrapping(set)
	// wrapped is safe for concurrent use
	fmt.Println(wrapped.Cardinality())

	// Output: 3
}

func ExampleIsEmpty() {
	set := New[int]()
	if IsEmpty(set) {
		fmt.Println("set is empty")
	}

	set.Add(5)
	if !IsEmpty(set) {
		fmt.Println("set is not empty")
	}
	// Output:
	// set is empty
	// set is not empty
}

func ExampleMapBy() {
	set := NewWith(1, 2, 3)

	mapped := MapBy(set, func(i int) int {
		return i * 2
	})
	for i := range mapped.Iterator {
		fmt.Println(i)
	}

	mapped2 := MapBy(set, func(i int) string {
		return fmt.Sprintf("%d", i)
	})
	for i := range mapped2.Iterator {
		fmt.Println(i)
	}
	// Unordered output:
	// 2
	// 4
	// 6
	// 1
	// 2
	// 3
}

func ExampleMapTo() {
	set := NewOrderedWith(3, 1, 2)

	dest := New[string]()
	MapTo(set, dest, func(i int) string {
		return fmt.Sprintf("%d=%d*2", i*2, i)
	})
	for i := range dest.Iterator {
		fmt.Println(i)
	}
	// Unordered output:
	// 6=3*2
	// 2=1*2
	// 4=2*2
}

func ExampleMapToSlice() {
	set := NewWith(3, 1, 2)

	mapped := MapToSlice(set, func(i int) string {
		return fmt.Sprintf("%d=%d*2", i*2, i)
	})
	for _, i := range mapped {
		fmt.Println(i)
	}
	// Unordered output:
	// 6=3*2
	// 2=1*2
	// 4=2*2
}

func ExampleFilter() {
	set := NewWith(3, 0, 1, 2, 4)

	filtered := Filter(set, func(i int) bool {
		return i > 2
	})
	for i := range filtered.Iterator {
		fmt.Println(i)
	}
	// Unordered output:
	// 3
	// 4
}

func ExampleReduce() {
	set := NewWith(3, 1, 2)

	sum := Reduce(set, 0, func(agg, v int) int {
		return agg + v
	})
	fmt.Println(sum)
	// Output: 6
}

func ExampleReduceRight() {
	set := NewOrderedWith(3, 1, 2)

	sum := ReduceRight(set, 0, func(agg, v int) int {
		fmt.Println(v)
		return agg + v
	})
	fmt.Println(sum)
	// Output:
	// 2
	// 1
	// 3
	// 6
}

func ExampleForEach() {
	set := NewWith(3, 1, 2)

	ForEach(set, func(i int) {
		fmt.Println(i)
	})
	// Unordered output:
	// 1
	// 2
	// 3
}

func ExampleForEachRight() {
	set := NewOrderedWith(3, 1, 2)

	ForEachRight(set, func(i int) {
		fmt.Println(i)
	})
	// Output:
	// 2
	// 1
	// 3
}

func ExampleAny() {
	set := NewWith(1, 2, 3, 4, 5)

	if Any(set, func(i int) bool { return i > 3 }) {
		fmt.Println("set has element > 3")
	}

	if !Any(set, func(i int) bool { return i > 10 }) {
		fmt.Println("set has no element > 10")
	}
	// Output:
	// set has element > 3
	// set has no element > 10
}

func ExampleAll() {
	set := NewWith(2, 4, 6)

	if All(set, func(i int) bool { return i%2 == 0 }) {
		fmt.Println("all elements are even")
	}

	set.Add(3)
	if !All(set, func(i int) bool { return i%2 == 0 }) {
		fmt.Println("not all elements are even")
	}
	// Output:
	// all elements are even
	// not all elements are even
}

func ExampleContainsAll() {
	set := NewWith(1, 2, 3, 4, 5)

	if ContainsAll(set, 1, 3, 5) {
		fmt.Println("set contains 1, 3, and 5")
	}

	if !ContainsAll(set, 1, 6) {
		fmt.Println("set does not contain both 1 and 6")
	}
	// Output:
	// set contains 1, 3, and 5
	// set does not contain both 1 and 6
}

func ExampleContainsAny() {
	set := NewWith(1, 2, 3)

	if ContainsAny(set, 3, 4, 5) {
		fmt.Println("set contains at least one of 3, 4, 5")
	}

	if !ContainsAny(set, 6, 7, 8) {
		fmt.Println("set contains none of 6, 7, 8")
	}
	// Output:
	// set contains at least one of 3, 4, 5
	// set contains none of 6, 7, 8
}

func ExampleRandom() {
	set := NewWith(42)

	// with a single element, Random always returns that element
	v, ok := Random(set)
	if ok {
		fmt.Println(v)
	}

	empty := New[int]()
	_, ok = Random(empty)
	if !ok {
		fmt.Println("empty set")
	}
	// Output:
	// 42
	// empty set
}

func ExampleFirst() {
	set := NewOrderedWith(5, 3, 1)

	if v, ok := First(set); ok {
		fmt.Println(v)
	}

	empty := NewOrdered[int]()
	if _, ok := First(empty); !ok {
		fmt.Println("empty set")
	}
	// Output:
	// 5
	// empty set
}

func ExampleLast() {
	set := NewOrderedWith(5, 3, 1)

	if v, ok := Last(set); ok {
		fmt.Println(v)
	}

	empty := NewOrdered[int]()
	if _, ok := Last(empty); !ok {
		fmt.Println("empty set")
	}
	// Output:
	// 1
	// empty set
}
