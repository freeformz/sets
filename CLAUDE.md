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
- `SortedSet[M]` (`sorted.go`) — always-sorted set backed by a sorted slice via `NewSortedSet()`; read-optimized (O(log n) Contains, O(1) At, `Range(lo, hi)` queries), O(n) Add/Remove. Implements the four set-algebra optimization interfaces (O(n+m) linear merge when both operands are SortedSets) and `Maxer`/`Minner` (O(1) from the slice ends)
- `BitSet[M]` (`bitset.go`) — always-sorted dense-bitmap set for integer element types (`Integer` constraint, not `comparable`) via `NewBitSet()`; O(1) Add/Remove/Contains and word-wise set ops between two BitSets (via the exported optional single-method interfaces `Unioner[M]`/`Intersectioner[M]`/`Differencer[M]`/`SymmetricDifferencer[M]` in `set.go`, which any implementation can adopt in any combination to accelerate the package-level algebra functions; the optional `Maxer[M]`/`Minner[M]` interfaces accelerate package-level `Max`/`Min` the same way). Memory ∝ element span, not count — see the type's godoc for the tradeoffs; `Reserve`/`Compact` manage the backing array
- `LockedOrdered[M]` (`locked_ordered.go`) — RWMutex wrapper around OrderedSet via `NewLockedOrdered()`

**Design philosophy**: Functionality lives in package-level generic functions (in `set.go` and `ordered_set.go`), not methods. This aligns with stdlib `slices`/`maps` style. Locked types use composition, wrapping an inner set with mutex protection.

All types implement `json.Marshaler`/`json.Unmarshaler` and `sql.Scanner`. The `Locker` interface (`locker.go`) is a marker for concurrent-safe implementations.

## Versioning

Releases are fully automated by `.github/workflows/release.yaml` ("Bump version"), which runs on every PR merge to `main`:

1. `anothrNick/github-tag-action` computes the next semver tag and pushes it (`v`-prefixed).
2. A `gh release create --generate-notes` step publishes a matching GitHub Release.

**This repo squash-merges PRs** (`allow_merge_commit: false`), so each merge produces exactly one commit on `main`. The tag action scans that squash commit's *entire* message — subject (the PR title, unless edited at merge time) plus body (which, by GitHub's default squash template, is the concatenated list of every individual commit message in the PR) — for bump tokens. Placing a token in **either** the PR title or any commit message in the branch works.

**Bump rules** (`DEFAULT_BUMP: patch` is configured in the workflow, overriding the action's own default of `minor`):

| Token (anywhere in the PR title or a commit message) | Result |
|---|---|
| *(none)* | patch bump — the default for ordinary fixes |
| `#minor` | minor bump — required for new backwards-compatible functionality, and for bumping the minimum Go version |
| `#major` | major bump — breaking API changes |
| `#none` | no tag/release is created for this merge — use for docs/chore/CI-only changes that shouldn't cut a release |

Match the token to the PR's overall scope, not to every individual commit inside it — a PR mixing a `feat:` with incidental `fix:`/`test:` commits still only needs one `#minor` somewhere for the whole merge to bump minor.

**Landmine**: the scanner does a plain substring match over the full squashed commit text — it has no idea a token appears inside prose rather than as a directive. Never write the literal, hash-prefixed tokens together in a commit message body when merely describing the convention (e.g. "supports #major/#minor/#patch/#none") — an incidental `#major` substring there will out-rank real `#none`/`#minor` tags elsewhere in the same PR and trigger an unintended major release (this happened once: PR #33's docs commit describing this exact convention accidentally cut `v1.0.0` from a `v0.12.0` base). When discussing the tokens in a commit message, drop the `#` prefix (`major`/`minor`/`patch`/`none`) or reference this file instead. The tokens are safe to write in full inside file content (e.g. this doc) — only commit *messages* are scanned, not diffs.

## Conventions

- **Zero-value style**: Use `var x T` instead of `x := T{}`; refer to this as "T's zero value" in prose.
- **Tests are the spec**: When modifying implementations, do not change tests. Treat test failures as implementation bugs.
- **Mathematical correctness**: Prefer mathematically correct semantics (e.g., vacuous truth for empty predicates).
- **Commit messages**: Use conventional commits (`feat:`, `fix:`, `docs:`, etc.). See [Versioning](#versioning) for the bump-token convention (`#minor`/`#major`/`#none`) that controls what release a PR produces.

## Testing

Tests use property-based state machine testing via `pgregory.net/rapid`. The state machine in `set_test.go` validates invariants across all set implementations. Tests run in parallel.

Heavier randomized stress tests live in the test-only `stresstest/` subpackage (differential testing against reference models, concurrency regression tests). New unit tests belong in the root package; add to `stresstest/` only for randomized/differential or concurrency stress coverage.
