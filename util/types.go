package util

type Pair[T any] struct {
	First, Second T
}

type List[T comparable] struct {
	Prev, Next *List[T]
	Value      T
}

func InsertRange[T int | int64 | uint64](list *List[Pair[T]], a, b T) *List[Pair[T]] {
	if a > b {
		a, b = b, a
	}

	if list == nil {
		return &List[Pair[T]]{nil, nil, Pair[T]{a, b}}
	}

	if list.Value.First > b {
		newHead := &List[Pair[T]]{nil, list, Pair[T]{a, b}}
		list.Prev = newHead
		return newHead
	}

	if list.Value.First == b {
		list.Value.First = a
		return list
	}

	node := list

	for node.Value.Second <= a && node.Next != nil {
		node = node.Next
	}

	if node.Prev != nil {
		if node.Prev.Value.Second <= a && node.Value.Second > a {
			node = node.Prev
		}
	}

	newNode := &List[Pair[T]]{node, node.Next, Pair[T]{a, b}}

	if node.Value.Second == a {
		node.Value.Second = b
		newNode = node
	} else {
		node.Next = newNode
	}

	if newNode.Next != nil {
		if newNode.Next.Value.First == newNode.Value.Second {
			newNode.Value.Second = newNode.Next.Value.Second
			newNode.Next = newNode.Next.Next
			if newNode.Next != nil {
				newNode.Next.Prev = newNode
			}
		} else {
			newNode.Next.Prev = newNode
		}
	}

	return list
}

func RemoveRange[T int | int64 | uint64](list *List[Pair[T]], a, b T) *List[Pair[T]] {
	if a > b {
		a, b = b, a
	}

	if list == nil {
		return nil
	}

	node := list

	for node != nil && (node.Value.First > a || node.Value.Second < b) {
		node = node.Next
	}

	if node == nil {
		return list
	}

	left, right := node.Prev, node.Next

	if node.Value.First == a {
		node.Value.First = b
	}

	if node.Value.Second == b {
		node.Value.Second = a
	}

	if node.Value.First > node.Value.Second {
		if left == nil {
			if right == nil {
				return nil
			}
			right.Prev = nil
			return right
		}
		left.Next = right
		if right != nil {
			right.Prev = left
		}
		return list
	}

	if node.Value.First == b || node.Value.Second == a {
		return list
	}

	next := &List[Pair[T]]{node, node.Next, Pair[T]{b, node.Value.Second}}
	if node.Next != nil {
		node.Next.Prev = next
	}
	node.Next = next
	node.Value.Second = a
	return list
}

func Contains[T int | int64 | uint64](list *List[Pair[T]], a, b T) bool {
	if a > b {
		a, b = b, a
	}

	if list == nil {
		return false
	}

	node := list

	for node != nil && (node.Value.First > a || node.Value.Second < b) {
		node = node.Next
	}

	if node == nil {
		return false
	}

	if node.Value.First <= a && node.Value.Second >= b {
		return true
	}
	return false
}
