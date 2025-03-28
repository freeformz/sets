# Sets

![ci status](https://github.com/freeformz/sets/actions/workflows/ci.yaml/badge.svg?branch=main)
[![Go Report Card](https://goreportcard.com/badge/github.com/freeformz/sets)](https://goreportcard.com/report/github.com/freeformz/sets)
[![GoDoc](https://godoc.org/github.com/freeformz/sets?status.svg)](http://godoc.org/github.com/freeformz/sets)

A generics based go set package that supports modern go features like iterators.

NOTE: This is currently a WIP. I don't expect to make any breaking API changes atm, but am not able to rule it out yet.

## Install

Use go get to install this package.

```console
go get github.com/freeformz/sets
```

## Features

* [Generics](https://go.dev/doc/tutorial/generics) based implementation.
* Common, minimal interface based Set type.
* Iterator support in the Set type and set methods.
* Multiple set implementations:
  * `New()` -> Map based set;
  * `NewLocked()` -> Map based that uses a lock to be concurrency safe;
  * `NewSync()` -> sync.Map based (concurrency safe);
  * `NewOrdered()` -> ordered set (uses a map for indexes and a slice for order);
  * `NewLockedOrdered()` -> ordered set that is concurrency safe.
* `set` package functions align with standard lib packages like `slices` and `maps`.
* Implement as much as possible as package functions, not Set methods.
* Exhaustive unit tests via [rapid](https://github.com/flyingmutant/rapid).
* Somewhat exhaustive examples.

## Usage

[Package Level Examples](https://pkg.go.dev/github.com/freeformz/sets#pkg-examples)

[Set Example](https://pkg.go.dev/github.com/freeformz/sets#example-Set)

[OrderedSet Example](https://pkg.go.dev/github.com/freeformz/sets#example-OrderedSet)

## JSON

Sets marshal to/from JSON as JSON arrays.
A JSON array with repeated values unmarshaled to a Set will not preserve duplicates.
An empty Set marshals to `[]`.
OrderedSets preserve order when {un,}marshaling, while Sets do not.

Sets of types that don't have a JSON equivalent can't be marshaled to and/or from JSON w/o an error. For instance a Set of an interface type can marshal to json, but can't then un-marshal back to Go w/o an error.

## Set Helpers

These helpers work on all Set types, including OrderedSets.

* `sets.Elements(aSet)` : Elements of the set as a slice.
* `sets.AppendSeq(aSet,sequence)` : Append the items in the sequence (an iterator) to the set.
* `sets.RemoveSeq(aSet,sequence)` : Remove the items in the sequence (an iterator) from the set.
* `sets.Union(aSet,bSet)` : Returns a new set (of the same underling type as aSet) with all elements from both sets.
* `sets.Intersection(aSet,bSet)` : Returns a new set (of the same underlying type as aSet) with elements that are in both sets.
* `sets.Difference(aSet,bSet)` : Returns a new set (of the same underlying type as aSet) with elements that are in the first set but not in the second set.
* `sets.SymmetricDifference(aSet,bSet)` : Returns a new set (of the same underlying type as aSet) with elements that are not in both sets.
* `sets.Subset(aSet,bSet)` : Returns true if all elements in the first set are also in the second set.
* `sets.Superset(aSet, bSet)` : Returns true if all elements in the second set are also in the first set.
* `sets.Equal(aSet, bSet)` : Returns true if the two sets contain the same elements.
* `sets.Disjoint(aSet, bSet)` : Returns true if the two sets have no elements in common.
* `sets.ContainsSeq(aSet, sequence)` : Returns true if the set contains all elements in the sequence. Empty sets are considered to contain only empty sequences.
* `sets.Iter2(sequence)` : Returns a (int,V) iterator where the int represents a "pseudo" index.
* `sets.Max(aSet)` : Returns the max element in the set as determined by the max builtin.
* `sets.Min(aSet)` : Returns the min element in the set as determined by the min builtin.
* `sets.Chunk(aSet,n)` : Chunks the set into n sets of equal size. The last set will have fewer elements if the cardinality of the set is not a multiple of n.
* `sets.IsEmpty(aSet)` : Returns true if the set is empty, otherwise false.
* `sets.MapBy(aSet, func(v V) X { return ... }) bSet` : Maps the elements of the set to a new set.
* `sets.MapTo(aSet, bSet, func(v V) X { return ... })` : Maps the elements of aSet into bSet.
* `sets.MapToSlice(aSet, func(v V) X { return ... }) aSlice` : Maps the elements of the set to a new slice.
* `sets.Filter(aSet, func(v V) bool { return true/false }) bSet` : Filters the elements of the set and returns a new set.
* `sets.Reduce(aSet, X, func(X, K) X { return ... }) X` : Reduces the set to a single value.
* `sets.ForEach(aSet, func(v V))` : calls the provided function with each set member.

## OrderedSet Helpers

These helpers work on all OrderedSet types.

* `sets.EqualOrdered(aOrderedSet, bOrderedSet)` : Returns true if the two OrderedSets contain the same elements in the same order.
* `sets.IsSorted(aOrderedSet)` : Returns true if the OrderedSet is sorted in ascending order.
* `sets.Reverse(aOrderedSet)` :  Returns a new OrderedSet with the elements in the reverse order of the original OrderedSet.
* `sets.Sorted(aOrderedSet)` : Return a copy of aOrderedSet with the elements sorted in ascending order. Does not modify the original set.
* `sets.ReduceRight(aSet, X, func(X, K) X { return ... }) X` : Reduces the set to a single value in reverse order.
* `sets.ForEachRight(aSet, func(K) { ... })` : calls the provided function with each set member in reverse order.

## Custom Set Types

You can implement your own set types as long as they conform to the interfaces and can use the package level functions
as they do not rely on any internal implementation details.

## TODOs

* Ordered rapid tests that test the OrderedSet bits like the normal Set bits are tested.
