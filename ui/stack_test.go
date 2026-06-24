package ui

import (
	"testing"

	"github.com/mirzakhany/yoga/layout"
)

func TestFluentModifierChain(t *testing.T) {
	a := layout.New(layout.Box())
	b := layout.New(layout.Box())

	root := HStack(a, b).
		Gap(5).
		FlexGrow(1).
		PaddingLeft(4).
		Margin(7)

	if got := root.Style.RowGap; got != 5 {
		t.Fatalf("Gap not applied: RowGap=%v", got)
	}
	if got := root.Style.Grow; got != 1 {
		t.Fatalf("FlexGrow not applied: Grow=%v", got)
	}
	if got := root.Style.Padding.Left; got != 4 {
		t.Fatalf("PaddingLeft not applied: %v", got)
	}
	if root.Style.Margin.Top != 7 || root.Style.Margin.Left != 7 {
		t.Fatalf("Margin not applied uniformly: %+v", root.Style.Margin)
	}
	if len(root.Children) != 2 {
		t.Fatalf("HStack should hold 2 children, got %d", len(root.Children))
	}
}

func TestStackDirections(t *testing.T) {
	if VStack().Style.Dir != layout.Column {
		t.Fatal("VStack should be a column")
	}
	if HStack().Style.Dir != layout.Row {
		t.Fatal("HStack should be a row")
	}
	if ZStack().Style.Disp != layout.DisplayStack {
		t.Fatal("ZStack should use stack display")
	}
}
