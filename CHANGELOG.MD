# Changelog

## Unreleased

## v0.6.1

* Added `sets.IsEmpty(set)` which is a convenience wrapper for `set.Cardinality() == 0`

## v0.6.0

* `NewXWith`: Convenience wrappers for `NewXFrom(slices.Values(slice))` and allows for easy set construction
  when the items are not already a slice. This did require a small breaking API change as there were already
  locking constructors using With, those were renamed to `NewLockingXWrapping`.
* Added examples for the constructors.
