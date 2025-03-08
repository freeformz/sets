# Sets

![ci status](https://github.com/freeformz/sets/actions/workflows/ci.yaml/badge.svg?branch=main)
[![Go Report Card](https://goreportcard.com/badge/github.com/freeformz/sets)](https://goreportcard.com/report/github.com/freeformz/sets)
[![GoDoc](https://godoc.org/github.com/freeformz/sets?status.svg)](http://godoc.org/github.com/freeformz/sets)

A generics based go set package that supports modern go features.

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
  * `NewOrdered()` -> ordred set (uses a map for indexes and a slice for order);
  * `NewLockedOrdered()` -> ordered set that is concurrency safe.
* `set` package functions align with standard lib packages like `slices` and `maps`.
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
