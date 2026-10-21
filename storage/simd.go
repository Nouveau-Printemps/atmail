package storage

import (
	"iter"
	"strings"
)

func simdSearch(haystack, needle string) iter.Seq[uint32] {
	return func(yield func(uint32) bool) {
		last := uint32(0)
		for {
			i := strings.Index(haystack[last:], needle)
			if i < 0 || !yield(uint32(i)) {
				return
			}
			last = uint32(i)
		}
	}
}
