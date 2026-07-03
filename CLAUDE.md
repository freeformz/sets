# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Go generics-based set library (`github.com/freeformz/sets`) supporting iterators (`iter.Seq`). Requires Go 1.25+ (see go.mod). No runtime dependencies — only test dependencies (`go-cmp`, `rapid`).

## Commands

```bash
# Run all tests with race detector
go test -v -race ./...

# Run a specific test
go test -v -run TestMap ./...

# Lint
go vet ./...
go install honnef.co/go/tools/cmd/staticcheck@latest && staticcheck ./...
```

CI runs checks (staticcheck, vet) on Ubuntu and tests (with `-race`) across Ubuntu/macOS/Windows on multiple Go versions.

## Architecture

**Interfaces** (`set.go`, `ordered_set.go`):
- `Set[M comparable]` — base interface all set types implement
- `OrderedSet[M cmp.Ordered]` — extends Set with index-based access and ordered iteration

**Implementations**:
- `Map[M]` (`map.go`) — default map-based set, created via `New()`
- `SyncMap[M]` (`sync.go`) — `sync.Map`-based, concurrent-safe via `NewSyncMap()`
- `Locked[M]` (`locked.go`) — RWMutex wrapper around a Set via `NewLocked()`
- `Ordered[M]` (`ordered.go`) — insertion-ordered set via `NewOrdered()`
- `LockedOrdered[M]` (`locked_ordered.go`) — RWMutex wrapper around OrderedSet via `NewLockedOrdered()`

**Design philosophy**: Functionality lives in package-level generic functions (in `set.go` and `ordered_set.go`), not methods. This aligns with stdlib `slices`/`maps` style. Locked types use composition, wrapping an inner set with mutex protection.

All types implement `json.Marshaler`/`json.Unmarshaler` and `sql.Scanner`. The `Locker` interface (`locker.go`) is a marker for concurrent-safe implementations.

## Versioning

When bumping the minimum Go version, the commit message should include `#minor` to trigger a minor version bump.

## Conventions

- **Zero-value style**: Use `var x T` instead of `x := T{}`; refer to this as "T's zero value" in prose.
- **Tests are the spec**: When modifying implementations, do not change tests. Treat test failures as implementation bugs.
- **Mathematical correctness**: Prefer mathematically correct semantics (e.g., vacuous truth for empty predicates).
- **Commit messages**: Use conventional commits (`feat:`, `fix:`, `docs:`, etc.).

## Testing

Tests use property-based state machine testing via `pgregory.net/rapid`. The state machine in `set_test.go` validates invariants across all set implementations. Tests run in parallel.

Heavier randomized stress tests live in the test-only `stresstest/` subpackage (differential testing against reference models, concurrency regression tests). New unit tests belong in the root package; add to `stresstest/` only for randomized/differential or concurrency stress coverage.
