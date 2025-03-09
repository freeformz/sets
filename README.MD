# Sets

![ci status](https://github.com/freeformz/sets/actions/workflows/ci.yaml/badge.svg?branch=main)
[![Go Report Card](https://goreportcard.com/badge/github.com/freeformz/sets)](https://goreportcard.com/report/github.com/freeformz/sets)
[![GoDoc](https://godoc.org/github.com/freeformz/sets?status.svg)](http://godoc.org/github.com/freeformz/sets)

A generics based go set package that supports modern go features like iterators.

NOTE: This is currently a WIP. I don't expect to make any breaking API changes atm, but I not able to rule it out yet.

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

See the package level [examples](https://pkg.go.dev/github.com/freeformz/sets#pkg-examples) for a more exhaustive set of examples.

```go
  a := New[string]()
  a.Add("hi")
  a.Add("there")

  a.Cardinality() // 2
  a.Contains("bob") // false
  a.Contains("hi") // true

  a.Remove("bob") // false
  a.Remove("hi") // true

  b := New[string]()
  b.Add("there")

  set.Equal(a,b) // false
  b.Add("hi")
  set.Equal(a,b) // true

  ...
```

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

## OrderedSet Helpers

These helpers work on all OrderedSet types.

* `sets.EqualOrdered(aOrderedSet, bOrderedSet)` : Returns true if the two OrderedSets contain the same elements in the same order.
* `sets.IsSorted(aOrderedSet)` : Returns true if the OrderedSet is sorted in ascending order.
* `sets.Reverse(aOrderedSet)` :  Returns a new OrderedSet with the elements in the reverse order of the original OrderedSet.

## TODOs

* Ordered rapid tests that test the OrderedSet bits like the normal Set bits are tested.
* More examples?
