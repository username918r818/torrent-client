package util_test

import (
	"github.com/username918r818/torrent-client/util"
	"testing"
)

type list = util.List[util.Pair[int]]
type pair = util.Pair[int]

// Функция для сравнения пар
func comparePairs(a, b pair) bool {
	if a.First != b.First || a.Second != b.Second {
		return false
	}
	return true
}

func TestInsertRange(t *testing.T) {

	t.Run("insert into empty list", func(t *testing.T) {
		var list *list
		list = util.InsertRange(list, 1, 5)
		if list == nil {
			t.Errorf("lsit is nill")
			return
		}

		if !comparePairs(list.Value, util.Pair[int]{1, 5}) {
			t.Errorf("Expected {1, 5}, got {%v, %v}", list.Value.First, list.Value.Second)
			return
		}
	})

	t.Run("insert into beginning", func(t *testing.T) {
		var list *list
		list = util.InsertRange(list, 2, 5)
		list = util.InsertRange(list, 0, 1)
		if !comparePairs(list.Value, util.Pair[int]{0, 1}) || list.Next == nil || !comparePairs(list.Next.Value, util.Pair[int]{2, 5}) {
			t.Errorf("qwerty")
			return
		}
	})

	t.Run("insert into end", func(t *testing.T) {
		var list *list
		list = util.InsertRange(list, 1, 5)
		list = util.InsertRange(list, 6, 10)
		if list == nil {
			t.Errorf("lsit is nill")
			return
		}
		if !comparePairs(list.Value, util.Pair[int]{1, 5}) {
			t.Errorf("first node is %v, %v", list.Next.Value.First, list.Next.Value.Second)
			return
		}
		list = list.Next

		if list == nil {
			t.Errorf("newList is nill")
			return
		}
		if !comparePairs(list.Value, util.Pair[int]{6, 10}) {
			t.Errorf("first node is %v, %v", list.Next.Value.First, list.Next.Value.Second)
			return
		}

	})

	t.Run("insert and merge at beginning", func(t *testing.T) {
		var list *list
		list = util.InsertRange(list, 1, 5)
		list = util.InsertRange(list, 0, 1)
		if list == nil {
			t.Errorf("newList is nill")
			return
		}
		if !comparePairs(list.Value, util.Pair[int]{0, 5}) {
			t.Errorf("Expected {0, 5}, got {%v, %v}", list.Value.First, list.Value.Second)
			return
		}
	})

	t.Run("insert and merge at end", func(t *testing.T) {
		var list *list
		list = util.InsertRange(list, 1, 5)
		list = util.InsertRange(list, 5, 10)
		list = util.InsertRange(list, 10, 15)
		if list == nil {
			t.Errorf("newList is nill")
			return
		}
		if !comparePairs(list.Value, util.Pair[int]{1, 15}) {
			t.Errorf("Expected {1, 15} got {%v, %v}", list.Value.First, list.Value.Second)
			return
		}
	})

	t.Run("insert into middle", func(t *testing.T) {
		var list *list
		list = util.InsertRange(list, 1, 3)
		list = util.InsertRange(list, 8, 10)
		list = util.InsertRange(list, 4, 7)
		if list == nil {
			t.Errorf("newList is nill")
			return
		}
		if !comparePairs(list.Value, util.Pair[int]{1, 3}) {
			t.Errorf("Expected {1, 3}, got {%v, %v}", list.Value.First, list.Value.Second)
		}
		list = list.Next

		if list == nil {
			t.Errorf("newList is nill")
			return
		}
		if !comparePairs(list.Value, util.Pair[int]{4, 7}) {
			t.Errorf("Expected {4, 7}, got {%v, %v}", list.Value.First, list.Value.Second)
		}
		list = list.Next

		if list == nil {
			t.Errorf("newList is nill")
			return
		}
		if !comparePairs(list.Value, util.Pair[int]{8, 10}) {
			t.Errorf("Expected {8, 10}, got {%v, %v}", list.Value.First, list.Value.Second)
		}
	})

	t.Run("merge in the middle with left", func(t *testing.T) {
		var list *list
		list = util.InsertRange(list, 1, 4)
		list = util.InsertRange(list, 7, 10)
		list = util.InsertRange(list, 4, 6)

		if list == nil {
			t.Errorf("newList is nill")
			return
		}

		if !comparePairs(list.Value, util.Pair[int]{1, 6}) {
			t.Errorf("Expected {1, 6}, got {%v, %v}", list.Value.First, list.Value.Second)
		}

		list = list.Next
		if list == nil {
			t.Errorf("newList is nill")
			return
		}

		if !comparePairs(list.Value, util.Pair[int]{7, 10}) {
			t.Errorf("Expected {7, 10}, got {%v, %v}", list.Value.First, list.Value.Second)
		}
	})

	t.Run("merge in the middle with right", func(t *testing.T) {
		var list *list
		list = util.InsertRange(list, 1, 3)
		list = util.InsertRange(list, 7, 10)
		list = util.InsertRange(list, 4, 7)

		if list == nil {
			t.Errorf("newList is nill")
			return
		}

		if !comparePairs(list.Value, util.Pair[int]{1, 3}) {
			t.Errorf("Expected {1, 3}, got {%v, %v}", list.Value.First, list.Value.Second)
		}

		list = list.Next
		if list == nil {
			t.Errorf("newList is nill")
			return
		}

		if !comparePairs(list.Value, util.Pair[int]{4, 10}) {
			t.Errorf("Expected {4, 10}, got {%v, %v}", list.Value.First, list.Value.Second)
		}
	})

	t.Run("merge in the middle with both left and right", func(t *testing.T) {
		var list *list
		list = util.InsertRange(list, 1, 2)
		list = util.InsertRange(list, 3, 4)  // 1,2  3,4
		list = util.InsertRange(list, 7, 8)  // 1,2  3,4  7,8
		list = util.InsertRange(list, 9, 10) // 1,2  3,4  7,8  9,10
		list = util.InsertRange(list, 4, 7)  // 1,2  3,8  9,10

		if list == nil {
			t.Errorf("newList is nill")
			return
		}

		if !comparePairs(list.Value, util.Pair[int]{1, 2}) {
			t.Errorf("Expected {1, 2}, got {%v, %v}", list.Value.First, list.Value.Second)
			// return
		}

		list = list.Next
		if list == nil {
			t.Errorf("newList is nill")
			return
		}

		if !comparePairs(list.Value, util.Pair[int]{3, 8}) {
			t.Errorf("Expected {3, 8}, got {%v, %v}", list.Value.First, list.Value.Second)
			// return
		}

		list = list.Next
		if list == nil {
			t.Errorf("newList is nill")
			return
		}

		if !comparePairs(list.Value, util.Pair[int]{9, 10}) {
			t.Errorf("Expected {9, 10}, got {%v, %v}", list.Value.First, list.Value.Second)
			// return
		}
	})
}
