package storage

// #include <stdlib.h>
// #include "../search/search.h"
import "C"
import (
	"iter"
	"unsafe"
)

func simdSearch(haystack, needle string) iter.Seq[uint32] {
	return func(yield func(uint32) bool) {
		last := uint32(0)
		var ok = true
		for ok {
			h := C.CString(haystack[last:])
			n := C.CString(needle)

			resp := C.search(h, n)
			C.free(unsafe.Pointer(h))
			C.free(unsafe.Pointer(n))

			if !yield(uint32(resp.pos)) {
				return
			}
			ok = bool(resp.found)
			last = uint32(resp.rest)
		}
	}
}
