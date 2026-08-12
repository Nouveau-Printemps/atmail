package utils

import "iter"

func Map[T, V any](in []T, fn func(T) V) []V {
	out := make([]V, 0, len(in))
	for _, v := range in {
		out = append(out, fn(v))
	}
	return out
}

func ReduceMapToSlice[A comparable, B, V any](in map[A]B, fn func(A, B) V) []V {
	out := make([]V, 0, len(in))
	for a, b := range in {
		out = append(out, fn(a, b))
	}
	return out
}

func Zip[A, B any](as iter.Seq[A], bs iter.Seq[B]) iter.Seq2[A, B] {
	return func(yield func(A, B) bool) {
		for a := range as {
			for b := range bs {
				if !yield(a, b) {
					return
				}
			}
		}
	}
}

func GroupBy[A any, B comparable](in []A, fn func(A) B) map[B][]A {
	mp := make(map[B][]A)
	for _, a := range in {
		key := fn(a)
		arr, ok := mp[key]
		if !ok {
			mp[key] = []A{a}
		} else {
			mp[key] = append(arr, a)
		}
	}
	return mp
}

func Any(in []bool) bool {
	for _, v := range in {
		if v {
			return true
		}
	}
	return false
}

func All(in []bool) bool {
	for _, v := range in {
		if !v {
			return false
		}
	}
	return true
}
