# Set

![ci status](https://github.com/freeformz/set/actions/workflows/ci.yaml/badge.svg?branch=main)
[![Go Report Card](https://goreportcard.com/badge/github.com/freeformz/set)](https://goreportcard.com/report/github.com/freeformz/set)
[![GoDoc](https://godoc.org/github.com/freeformz/set?status.svg)](http://godoc.org/github.com/freeformz/set)

A generics based go set package that supports modern go features.

## Install

Use go get to install this package.

```console
go get github.com/freeformx/set
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
* [TODO] Exhaustive examples.

## Usage

`TODO`