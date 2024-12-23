package main

// this DS does not provide thread security
func NewStack[T any]() *Stack[T] {
	return &Stack[T]{
		items: make([]T, 0),
	}
}

type Stack[T any] struct {
	items []T
}

func (s *Stack[T]) Push(item T) {
	s.items = append(s.items, item)
}
func (s *Stack[T]) Pop() T {
	l := len(s.items)
	if l == 0 {
		panic("Stack: pop")
	}
	ret := s.items[l]
	s.items = s.items[:l-1]
	return ret
}
func (s Stack[T]) Peek() T {
	l := len(s.items)
	if l == 0 {
		panic("Stack: peek")
	}
	return s.items[l]
}
func (s Stack[T]) IsEmpty() bool {
	return len(s.items) == 0
}
