package ui

import (
	"testing"

	"github.com/mirzakhany/yoga/layout"
)

func TestScrollViewContentHeight(t *testing.T) {
	content := layout.New(layout.Box().Direction(layout.Column).Gap(8))
	for i := 0; i < 10; i++ {
		content.Children = append(content.Children, layout.New(layout.Box().H(40)))
	}
	sv := NewScrollView(content)
	root := layout.New(layout.Box().FlexGrow(1), sv.host)
	root.Calculate(400, 200)

	if content.Frame.H < 470 {
		t.Fatalf("scroll content height: got %v want >= 470", content.Frame.H)
	}
	if content.Children[0].Frame.H < 39 {
		t.Fatalf("block height shrunk: got %v", content.Children[0].Frame.H)
	}
}

func TestScrollDSLDoesNotShrinkChildren(t *testing.T) {
	c := New(nil, NewFocusScope(), nil)
	blocks := make([]View, 0, 10)
	for i := 0; i < 10; i++ {
		blocks = append(blocks, Raw(layout.New(layout.Box().H(40).FlexShrink(0))))
	}
	root := BuildFrame(c, func(_ *Ctx) View {
		return Scroll("s", Column(blocks...).Gap(8)).Grow(1)
	}, 400, 200, nil, nil)

	var col *layout.Element
	var find func(*layout.Element)
	find = func(e *layout.Element) {
		if e == nil || col != nil {
			return
		}
		if len(e.Children) == 10 {
			col = e
			return
		}
		for _, ch := range e.Children {
			find(ch)
		}
	}
	find(root)
	if col == nil {
		t.Fatal("scroll content column not found")
	}
	if col.Frame.H < 470 {
		t.Fatalf("content height: got %v want >= 470", col.Frame.H)
	}
	for i := 1; i < len(col.Children); i++ {
		prev := col.Children[i-1]
		ch := col.Children[i]
		if ch.Frame.Y < prev.Frame.Y+prev.Frame.H-0.5 {
			t.Fatalf("children overlap at %d: prev=%v child=%v", i, prev.Frame, ch.Frame)
		}
	}
}
