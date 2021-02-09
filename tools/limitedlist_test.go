package tools

import (
	"fmt"
	"testing"
)

func TestLimitedList_foreach(t *testing.T) {
	l, _ := NewLimitedSortList(4)

	f1(l)
	lf(l)
	lReverse(l)
}

func f1(l *LimitedList) {
	l.Put(1)
	l.Put(2)
	l.Put(3)
	l.Put(4)
	l.Put(5)

}

func f2(l *LimitedList) {
	l.Put(6)
	l.Put(7)
}

func f3(l *LimitedList) {
	l.Put(8)
	l.Put(9)
}

func lf(l *LimitedList) {
	l.Foreach(func(v interface{}) error {
		fmt.Println(v)
		return nil
	}, false)
	fmt.Println("---")
}

func lReverse(l *LimitedList) {
	l.ForeachReverse(func(v interface{}) error {
		fmt.Println(v)
		return nil
	}, false)
	fmt.Println("---")
}
