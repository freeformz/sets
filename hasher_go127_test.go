//go:build go1.27

package sets

import "hash/maphash"

var _ maphash.Hasher[Set[int]] = SetHasher[int]{}
