# Changelog

## Unreleased

## v0.7.0

* Added `set.Pop() (M, bool)`, which returns and removes a random element from the set.
  The second return value is false if nothing was removed.

## v0.6.1

* Added `sets.IsEmpty(set)` which is a convenience wrapper for `set.Cardinality() == 0`

## v0.6.0

* `NewXWith`: Convenience wrappers for `NewXFrom(slices.Values(slice))` and allows for easy set construction
  when the items are not already a slice. This did require a small breaking API change as there were already
  locking constructors using With, those were renamed to `NewLockingXWrapping`.
* Added examples for the constructors.
