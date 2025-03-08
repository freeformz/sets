package sets

import (
	"fmt"
	"slices"
)

func ExampleMap_Iterator() {
	ints := New[int]()
	ints.Add(5)
	ints.Add(3)
	ints.Add(2)
	ints.Add(4)
	ints.Add(1)

	out := make([]int, 0, ints.Cardinality())
	for i := range ints.Iterator {
		out = append(out, i)
	}

	// sort the values for consistent output
	slices.Sort(out)
	for _, i := range out {
		fmt.Println(i)
	}
	// Output:
	// 1
	// 2
	// 3
	// 4
	// 5
}

func ExampleOrdered() {
	ints := NewOrdered[int]()
	ints.Add(5)
	ints.Add(3)

	AppendSeq(ints, slices.Values([]int{2, 4, 1}))
	AppendSeq(ints, slices.Values([]int{5, 6, 1}))

	out := make([]int, 0, ints.Cardinality())
	for i := range ints.Iterator {
		out = append(out, i)
	}

	for _, i := range out {
		fmt.Println(i)
	}
	// Output:
	// 5
	// 3
	// 2
	// 4
	// 1
	// 6
}
