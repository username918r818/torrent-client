package util

import "sync"

type Range = Pair[int64, int64]

type RangeSet interface {
	Lock()
	Unlock()
	Contains(Range) bool
	IsEmpty(Range) bool
	Insert(Range)
	Extract(Range)
	Find(int64) *Range
}

type naiveImpl struct {
	sync.Mutex
	list *List[Range]
}

func GetNaiveRangeSet() RangeSet {
	return &naiveImpl{}
}

func (n *naiveImpl) Contains(r Range) bool {
	return Contains(n.list, r.First, r.Second)
}

func (n *naiveImpl) IsEmpty(r Range) bool {
	tmp := n.list

	for tmp != nil {
		if tmp.Value.First < r.Second && r.First < tmp.Value.Second {
			return false
		}
		tmp = tmp.Next
	}

	return true
}

func (n *naiveImpl) Insert(r Range) {
	n.Extract(r)

	InsertRange(n.list, r.First, r.Second)
}

func (n *naiveImpl) Extract(r Range) {
	tmp := n.list

	for tmp != nil {
		if tmp.Value.First < r.Second && r.First < tmp.Value.Second {
			switch {
			case tmp.Value.First < r.First && r.Second < tmp.Value.Second:
				tmp.Value.Second = r.First

				newTmp := &List[Range]{Prev: tmp, Next: tmp.Next}
				newTmp.Value.First = r.Second
				newTmp.Value.Second = tmp.Value.Second
				tmp.Next = newTmp
				newTmp.Next.Prev = newTmp
				return
			case tmp.Value.First < r.First && tmp.Value.Second <= r.Second:
				tmp.Value.Second = r.First

			case r.First <= tmp.Value.First && r.Second < tmp.Value.Second:
				tmp.Value.First = r.Second

			case r.First <= tmp.Value.First && tmp.Value.Second <= r.Second:
				cur := tmp
				tmp = tmp.Next
				tmp.Prev = cur.Prev
				tmp.Prev.Next = tmp
				continue
			}
		}
		tmp = tmp.Next
	}
}

func (n *naiveImpl) Find(i int64) *Range {
	tmp := n.list

	for tmp != nil {
		if tmp.Value.Second > i {
			return &Range{tmp.Value.First, tmp.Value.Second}
		}
		tmp = tmp.Next
	}
	return nil
}
