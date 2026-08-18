package ui

import (
	"testing"

	"github.com/mirzakhany/yoga/layout"
)

func TestFluentModifierChain(t *testing.T) {
	a := Text("a")
	b := Text("b")

	root := Row(a, b).
		Gap(5).
		Grow(1).
		PaddingLeft(4).
		Margin(7)

	el := root.Layout(New(nil, NewFocusScope(), nil))
	if got := el.Style.RowGap; got != 5 {
		t.Fatalf("Gap not applied: RowGap=%v", got)
	}
	if got := el.Style.Grow; got != 1 {
		t.Fatalf("Grow not applied: Grow=%v", got)
	}
	if got := el.Style.Padding.Left; got != 4 {
		t.Fatalf("PaddingLeft not applied: %v", got)
	}
	if el.Style.Margin.Top != 7 || el.Style.Margin.Left != 7 {
		t.Fatalf("Margin not applied uniformly: %+v", el.Style.Margin)
	}
	if len(el.Children) != 2 {
		t.Fatalf("Row should hold 2 children, got %d", len(el.Children))
	}
}

func TestStackDirections(t *testing.T) {
	c := New(nil, NewFocusScope(), nil)
	if Column().Layout(c).Style.Dir != layout.Column {
		t.Fatal("Column should be a column")
	}
	if Row().Layout(c).Style.Dir != layout.Row {
		t.Fatal("Row should be a row")
	}
	if Stack().Layout(c).Style.Disp != layout.DisplayStack {
		t.Fatal("Stack should use stack display")
	}
}
