package util

type Pair[T any] struct {
	first, second T
}

type List[T comparable] struct {
	prev, next *List[T]
	value T
}