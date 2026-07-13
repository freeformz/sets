package stresstest

import (
	"sync"
	"testing"

	"github.com/freeformz/sets"
)

// TestLockedDelegationConcurrency hammers the locked wrappers' optional-interface delegation
// from both operand orders at once — the classic two-lock deadlock shape — while writers mutate
// both sets. The delegation must stay deadlock-free (operand locks are only try-acquired; under
// contention it declines into the generic one-lock-at-a-time path) and race-free under -race.
// Contents race with the writers, so only type invariants are asserted; the real assertions are
// that the test terminates and the race detector stays quiet.
func TestLockedDelegationConcurrency(t *testing.T) {
	t.Parallel()

	iters := 2000
	if testing.Short() {
		iters = 200
	}

	a := sets.NewLockedWrapping(sets.Set[int](sets.NewBitSetWith(1, 2, 3)))
	b := sets.NewLockedWrapping(sets.Set[int](sets.NewBitSetWith(3, 4, 5)))

	var wg sync.WaitGroup
	for g := range 4 {
		wg.Go(func() {
			x, y := a, b
			if g%2 == 1 { // half the goroutines run the mirror-image operand order
				x, y = y, x
			}
			for i := range iters {
				switch i % 5 {
				case 0:
					if u := sets.Union(x, y); u == nil {
						t.Error("Union returned nil")
						return
					}
				case 1:
					if d := sets.Difference(x, y); d == nil {
						t.Error("Difference returned nil")
						return
					}
				case 2:
					sets.Equal(x, y)
				case 3:
					sets.Subset(x, y)
				case 4:
					sets.Disjoint(x, y)
				}
			}
		})
	}
	for w := range 2 {
		wg.Go(func() {
			for i := range iters {
				if w == 0 {
					a.Add(i % 64)
					b.Remove(i % 64)
				} else {
					b.Add(i % 64)
					a.Remove(i % 64)
				}
			}
		})
	}
	wg.Wait()
}
