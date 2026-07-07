package stresstest

import (
	"iter"
	"math/rand/v2"
	"testing"

	"github.com/freeformz/sets"
)

// rangeOrderedSet is the surface shared by SortedSet and BitSet that the
// always-sorted differential step/check helpers exercise.
type rangeOrderedSet interface {
	sets.OrderedSet[int]
	Range(lo, hi int) iter.Seq[int]
}

// TestBitSetDifferential runs the same randomized Add/Remove/Sort/Pop/Clear differential as
// TestSortedSetDifferential against BitSet, additionally interleaving Compact and Reserve —
// the memory-management operations must never change the observable contents. The element
// domain (0..49) fits in a single 64-bit word; it is the Reserve span (-64..128), crossing
// the signed-zero word boundary, that forces Compact and regrowth to shuffle whole words
// around the occupied one. (Multi-word add/remove churn is covered by the rapid tests in the
// root package, which draw from ±1024.)
func TestBitSetDifferential(t *testing.T) {
	t.Parallel()

	trials := 250
	if testing.Short() {
		trials = 25
	}

	for trial := range trials {
		rng := rand.New(rand.NewPCG(uint64(trial), 0xb175e7))
		s := sets.NewBitSet[int]()
		var r unsortedModel
		for op := range 300 {
			sortedStep(t, rng, s, &r, trial, op)
			switch rng.IntN(20) {
			case 0:
				s.Compact()
			case 1:
				s.Reserve(-64, 128) // wider than the element domain, crosses zero
			}
			sortedCheck(t, rng, s, &r, trial, op)
		}
	}
}
