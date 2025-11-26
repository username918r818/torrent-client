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

func TestRemoveRange(t *testing.T) {

	t.Run("remove from empty list", func(t *testing.T) {
		var list *list
		list = util.RemoveRange(list, 1, 5)
		if list != nil {
			t.Errorf("List should remain nil when removing from an empty list")
		}
	})

	t.Run("remove range from single element list", func(t *testing.T) {
		var list *list
		list = util.InsertRange(list, 1, 5)
		list = util.RemoveRange(list, 1, 5)
		if list != nil {
			t.Errorf("List should be empty after removing the only element {1, 5}")
		}
	})

	t.Run("remove range from middle", func(t *testing.T) {
		var list *list
		list = util.InsertRange(list, 1, 5)
		list = util.InsertRange(list, 5, 15)
		list = util.RemoveRange(list, 6, 9) // No range matches
		if list == nil {
			t.Errorf("newList is nill")
			return
		}

		if !comparePairs(list.Value, util.Pair[int]{1, 6}) {
			t.Errorf("Expected {1, 6}, got {%v, %v}", list.Value.First, list.Value.Second)
			// return
		}

		list = list.Next

		if list == nil {
			t.Errorf("newList is nill")
			return
		}

		if !comparePairs(list.Value, util.Pair[int]{9, 15}) {
			t.Errorf("Expected {9, 15}, got {%v, %v}", list.Value.First, list.Value.Second)
			// return
		}

		list = list.Next
	})

	t.Run("remove range that matches exactly behind", func(t *testing.T) {
		var list *list
		list = util.InsertRange(list, 1, 5)
		list = util.InsertRange(list, 10, 15)
		list = util.RemoveRange(list, 1, 5)
		if list == nil {
			t.Errorf("newList is nill")
			return
		}
		if !comparePairs(list.Value, util.Pair[int]{10, 15}) {
			t.Errorf("Expected list {10, 15}, got {%v, %v}", list.Value.First, list.Value.Second)
		}
	})

	t.Run("remove range that matches exactly in the end", func(t *testing.T) {
		var list *list
		list = util.InsertRange(list, 1, 5)
		list = util.InsertRange(list, 10, 15)
		list = util.RemoveRange(list, 10, 15)
		if list == nil {
			t.Errorf("newList is nill")
			return
		}
		if !comparePairs(list.Value, util.Pair[int]{1, 5}) {
			t.Errorf("Expected list {1, 5}, got {%v, %v}", list.Value.First, list.Value.Second)
		}
	})

	t.Run("remove and split range", func(t *testing.T) {
		var list *list
		list = util.InsertRange(list, 1, 5)   // {1, 5}
		list = util.InsertRange(list, 10, 15) // {1, 5} -> {10, 15}
		list = util.RemoveRange(list, 10, 13) // Remove range {3, 13}
		if list == nil {
			t.Errorf("newList is nill")
			return
		}

		if !comparePairs(list.Value, util.Pair[int]{1, 5}) {
			t.Errorf("Expected {1, 5}, got {%v, %v}", list.Value.First, list.Value.Second)
			// return
		}

		list = list.Next
		if list == nil {
			t.Errorf("newList is nill")
			return
		}

		if !comparePairs(list.Value, util.Pair[int]{13, 15}) {
			t.Errorf("Expected {13, 15}, got {%v, %v}", list.Value.First, list.Value.Second)
			// return
		}
	})

	t.Run("remove middle range", func(t *testing.T) {
		var list *list
		list = util.InsertRange(list, 1, 5)   // {1, 5}
		list = util.InsertRange(list, 10, 15) // {1, 5} -> {10, 15}
		list = util.InsertRange(list, 6, 9)   // {1, 5} -> {6, 9} -> {10, 15}
		list = util.RemoveRange(list, 6, 9)   // Remove {6, 9}
		if list == nil || !comparePairs(list.Value, util.Pair[int]{1, 5}) || !comparePairs(list.Next.Value, util.Pair[int]{10, 15}) {
			t.Errorf("Expected list {1, 5} -> {10, 15}, got {%v, %v} -> {%v, %v}",
				list.Value.First, list.Value.Second,
				list.Next.Value.First, list.Next.Value.Second)
		}
	})

	t.Run("remove range from the beginning", func(t *testing.T) {
		var list *list
		list = util.InsertRange(list, 1, 5)   // {1, 5}
		list = util.InsertRange(list, 10, 15) // {1, 5} -> {10, 15}
		list = util.RemoveRange(list, 1, 3)   // Remove {1, 3}, leave {3, 5} -> {10, 15}
		if list == nil || !comparePairs(list.Value, util.Pair[int]{3, 5}) || !comparePairs(list.Next.Value, util.Pair[int]{10, 15}) {
			t.Errorf("Expected list {3, 5} -> {10, 15}, got {%v, %v} -> {%v, %v}",
				list.Value.First, list.Value.Second,
				list.Next.Value.First, list.Next.Value.Second)
		}
	})

	t.Run("remove range from the end", func(t *testing.T) {
		var list *list
		list = util.InsertRange(list, 1, 5)   // {1, 5}
		list = util.InsertRange(list, 10, 15) // {1, 5} -> {10, 15}
		list = util.RemoveRange(list, 12, 15) // Remove {12, 15}, leave {1, 5} -> {10, 12}
		if list == nil || !comparePairs(list.Value, util.Pair[int]{1, 5}) || !comparePairs(list.Next.Value, util.Pair[int]{10, 12}) {
			t.Errorf("Expected list {1, 5} -> {10, 12}, got {%v, %v} -> {%v, %v}",
				list.Value.First, list.Value.Second,
				list.Next.Value.First, list.Next.Value.Second)
		}
	})
}
