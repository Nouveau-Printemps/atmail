package utils

import (
	"iter"
	"maps"
)

type Set[V comparable] = map[V]struct{}

func NewSet[V comparable]() Set[V] {
	return make(Set[V])
}

func SetContains[V comparable](s Set[V], v V) bool {
	_, ok := s[v]
	return ok
}

func SetDelete[V comparable](s Set[V], v V) {
	delete(s, v)
}

func SetInsert[V comparable](s Set[V], v V) {
	s[v] = struct{}{}
}

func SetMap[F, T comparable](s Set[F], fn func(F) T) Set[T] {
	res := make(Set[T], len(s))
	for v := range s {
		SetInsert(res, fn(v))
	}
	return res
}

func SetValues[V comparable](s Set[V]) iter.Seq[V] {
	return maps.Keys(s)
}
