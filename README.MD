# Set

![ci status](https://github.com/freeformz/set/actions/workflows/ci.yaml/badge.svg?branch=main)
[![Go Report Card](https://goreportcard.com/badge/github.com/freeformz/set)](https://goreportcard.com/report/github.com/freeformz/set)
[![GoDoc](https://godoc.org/github.com/freeformz/set?status.svg)](http://godoc.org/github.com/freeformz/set)

A generics based go set package that supports modern go features.

NOTE: This is currently a WIP, but I hope to stabilize the API soon.

## Install

Use go get to install this package.

```console
go get github.com/freeformz/set
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
* Exhaustive unit tests via [rapid](https://github.com/flyingmutant/rapid).
* Somewhat exhaustive examples.

## Usage

See the package level [examples](https://pkg.go.dev/github.com/freeformz/set#pkg-examples) for a more exhaustive set of examples.

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

## TODOs

* Ordered rapid tests that test the OrderedSet bits like the normal Set bits are tested.
* More examples?
