package sets

import (
	"encoding/json"
	"fmt"
	"slices"
)

func ExampleSet() {
	ints := New[int]()
	ints.Add(5)
	ints.Add(1)

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

	if x := ints.Clear(); x == 2 {
		fmt.Println("Clear removed all elements")
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
	// ints has 3 elements
	// Cloned set has 3 elements
	// 5 is present
	// 2 is not present
	// 2 was not removed
	// 5 was removed
	// Clear removed all elements
	// 1
	// 5
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
	ints := New[int]()
	ints.Add(5)
	ints.Add(3)
	ints.Add(2)

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
	ints := New[int]()
	ints.Add(5)
	ints.Add(3)
	// adds 2,4,1 to the set since 5 and 3 already exist
	added := AppendSeq(ints, slices.Values([]int{5, 3, 2, 4, 1}))
	fmt.Println(added)
	// Output: 3
}

func ExampleRemoveSeq() {
	ints := New[int]()
	ints.Add(5)
	ints.Add(3)
	ints.Add(2)
	// removes 2 from the set since 5 and 3 exist
	removed := RemoveSeq(ints, slices.Values([]int{2, 4, 1}))
	fmt.Println(removed)
	// Output: 1
}

func ExampleUnion() {
	a := New[int]()
	a.Add(5)
	a.Add(3)

	b := New[int]()
	b.Add(3)
	b.Add(2)

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
	a := New[int]()
	a.Add(5)
	a.Add(3)

	b := New[int]()
	b.Add(3)
	b.Add(2)

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
	a := New[int]()
	a.Add(5)
	a.Add(3)

	b := New[int]()
	b.Add(3)
	b.Add(2)

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
	a := New[int]()
	a.Add(5)
	a.Add(3)

	b := New[int]()
	b.Add(3)
	b.Add(2)

	c := SymmetricDifference(a, b)
	for i := range c.Iterator {
		fmt.Println(i)
	}
	// Unordered output:
	// 2
	// 5
}

func ExampleSubset() {
	a := New[int]()
	a.Add(5)
	a.Add(3)

	b := New[int]()
	b.Add(5)
	b.Add(3)
	b.Add(2)

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
	a := New[int]()
	a.Add(5)
	a.Add(3)

	b := New[int]()
	b.Add(5)
	b.Add(3)
	b.Add(2)

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
	a := New[int]()
	a.Add(5)
	a.Add(3)

	b := New[int]()
	b.Add(5)
	b.Add(3)

	if Equal(a, b) {
		fmt.Println("a and b are equal")
	}

	b.Add(2)
	if !Equal(a, b) {
		fmt.Println("a and b are not equal now")
	}
	// Output:
	// a and b are equal
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
	a := New[int]()
	a.Add(5)
	a.Add(3)

	b := New[int]()
	b.Add(2)
	b.Add(4)

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
	a := NewOrdered[int]()
	a.Add(5)
	a.Add(3)
	a.Add(1)

	b := NewOrdered[int]()
	b.Add(5)
	b.Add(3)
	b.Add(1)

	if EqualOrdered(a, b) {
		fmt.Println("a and b are equal")
	}

	b.Add(2)
	if !EqualOrdered(a, b) {
		fmt.Println("a and b are not equal now")
	}
	// Output:
	// a and b are equal
	// a and b are not equal now
}

func ExampleMin() {
	ints := New[int]()
	ints.Add(3)
	ints.Add(2)
	ints.Add(5)

	min := Min(ints)
	fmt.Println(min)
	// Output: 2
}

func ExampleMax() {
	ints := New[int]()
	ints.Add(3)
	ints.Add(5)
	ints.Add(2)

	max := Max(ints)
	fmt.Println(max)
	// Output: 5
}

func ExampleIsSorted() {
	ints := NewOrdered[int]()
	ints.Add(2)
	ints.Add(3)
	ints.Add(5)

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
	ints := NewOrdered[int]()
	ints.Add(2)
	ints.Add(3)
	ints.Add(5)

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
	ints := NewOrdered[int]()
	ints.Add(2)
	ints.Add(5)
	ints.Add(3)

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
	ints := NewOrdered[int]()
	AppendSeq(ints, slices.Values([]int{1, 2, 3, 4, 5}))

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
	ints := NewOrdered[int]()
	AppendSeq(ints, slices.Values([]int{1, 2, 3, 4, 5}))

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
	set := NewOrderedFrom(slices.Values([]float32{1.0, 1.2, 1.3, 1.4, 1.5}))
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
	m := []string{"a", "b", "c", "b"}
	set := NewWith(m...)
	fmt.Println(set.Cardinality())

	// Output: 3
}

func ExampleNewLockedWith() {
	m := []string{"a", "b", "c", "b"}
	set := NewLockedWith(m...)
	fmt.Println(set.Cardinality())

	// Output: 3
}

func ExampleNewOrderedWith() {
	m := []string{"a", "b", "c", "b"}
	set := NewOrderedWith(m...)
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
	m := []string{"a", "b", "c", "b"}
	set := NewLockedOrderedWith(m...)
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

func ExampleNewSyncWith() {
	m := []string{"a", "b", "c", "b"}
	set := NewSyncWith(m...)
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

func ExampleNewSync() {
	set := NewSync[string]()
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

func ExampleNewSyncFrom() {
	m := []string{"a", "b", "c", "b"}
	set := NewSyncFrom(slices.Values(m))
	fmt.Println(set.Cardinality())

	// Output: 3
}

func ExampleNewLockedWrapping() {
	set := New[string]()
	set.Add("a")
	set.Add("b")
	set.Add("c")
	set.Add("b")

	wrapped := NewLockedWrapping(set)
	// wrapped is safe for concurrent use
	fmt.Println(wrapped.Cardinality())

	// Output: 3
}

func ExampleNewLockedOrderedWrapping() {
	set := NewOrdered[string]()
	set.Add("a")
	set.Add("b")
	set.Add("c")
	set.Add("b")

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
