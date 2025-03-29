# Changelog

## v0.10.1

* Retract v0.9.{0,1} as the API change was reverted in v0.10.0

## v0.10.0

* Rename `New{MapWith,MapFrom,}` to `New{With,From,}` for ergonomic reasons. Sorry about the thrash.
* Since the types are exposed, started to add godocs for each of the methods for the exposed types.
* Removed the sync.Cond on the locked types. NFI why I had it there as it's not needed.

## v0.9.1

* Small staticcheck fixes

## v0.9.0

* Renames
  * func Map to func MapBy
  * func New to func NewMap
  * func NewFrom to func NewMapFrom
  * func NewWith to func NewMapWith
* Set types are now exported, so that ...
* Concrete types can be used in structs and json.Unmarshaled to.
  * sets.Map (basic set type)
  * sets.Locked (basic locked set type)
  * sets.Ordered (basic ordred set type)
  * sets.LockedOrdered (basic locked+ordered set type)
  * sets.SyncMap (sync.Map based set type)
* Added more JSON tests, especially for above

## v0.8.0

* Additional helpers:
  * Map - maps the items in the set and returns them as a new set
  * MapTo - maps the items in the set into a provided set
  * MapToSlice - maps the items in the set and returns them as a slice
  * Filter - filters the items in the set based on the function and returns a new set
  * Reduce - reduces the items in the set to a single value
  * ReduceRight - reduces the items in the ordered set, in backwards order, to a single value
  * ForEach - calls the provided function with each member of the set
  * ForEachRight - calls the provided function for each item in the ordered set, in backwards order.
* Cleaned up some Examples so they now use New{Ordered,}With instead of New{Ordered,} and multiple Adds

## v0.7.1

* cmp.Diff examples

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
