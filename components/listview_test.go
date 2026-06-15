package components

import (
	"testing"

	"github.com/mirzakhany/yoga/layout"
)

func TestListViewAdd(t *testing.T) {
	block := layout.New(layout.Box().H(40))
	lv := NewListView(ListViewConfig{Gap: 8})
	for i := 0; i < 10; i++ {
		lv.Add(block)
	}
	if lv.Len() != 10 {
		t.Fatalf("len: got %d want 10", lv.Len())
	}

	root := layout.New(layout.Box().FlexGrow(1), lv.El)
	root.Calculate(400, 200)

	if lv.stack.Frame.H < 470 {
		t.Fatalf("list content height: got %v want >= 470", lv.stack.Frame.H)
	}
}

func TestListViewSetAndClear(t *testing.T) {
	lv := NewListView(ListViewConfig{})
	a := layout.New(layout.Box().H(20))
	b := layout.New(layout.Box().H(20))
	lv.SetItems([]*layout.Element{a, b})
	if lv.Len() != 2 {
		t.Fatalf("set: got %d want 2", lv.Len())
	}
	lv.Clear()
	if lv.Len() != 0 {
		t.Fatalf("clear: got %d want 0", lv.Len())
	}
	if len(lv.stack.Children) != 0 {
		t.Fatal("stack children not cleared")
	}
}

func TestListViewRemove(t *testing.T) {
	lv := NewListView(ListViewConfig{})
	a := layout.New(layout.Box().H(20))
	b := layout.New(layout.Box().H(20))
	lv.Add(a, b)
	if !lv.Remove(0) {
		t.Fatal("remove failed")
	}
	if lv.Len() != 1 || lv.Item(0) != b {
		t.Fatalf("after remove: len=%d item=%p want %p", lv.Len(), lv.Item(0), b)
	}
	if lv.Remove(5) {
		t.Fatal("remove out of range should fail")
	}
}
