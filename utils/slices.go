package utils

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
