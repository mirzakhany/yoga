package ui

import (
	"testing"

	"github.com/mirzakhany/yoga/icons"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
)

func layoutIconButtonForTest(c *Ctx, n *Node) *layout.Element {
	c.BeginFrame(400, 300, nil, nil)
	el := n.Layout(c)
	el.Frame = render.Rect{X: 300, Y: 10, W: 32, H: 32}
	return el
}

func TestIconButtonMenuOpensOnClick(t *testing.T) {
	c := New(nil, NewFocusScope(), nil)
	clicked := false
	n := IconButton("more", icons.Ellipsis).
		OnClick(func() { clicked = true }).
		Menu([]MenuItem{{Label: "Duplicate", OnSelect: func() {}}, {Label: "Delete", OnSelect: func() {}}})

	el := layoutIconButtonForTest(c, n)
	if got := len(c.Overlays()); got != 0 {
		t.Fatalf("closed menu should register no overlay, got %d", got)
	}
	mst := c.Widget("more-menu", func() any { return &dropdownState{} }).(*dropdownState)

	clickAt(el, el.Frame.X+16, el.Frame.Y+16)
	if !mst.menu.Open {
		t.Fatal("expected the menu open after a click")
	}
	if clicked {
		t.Fatal("a button with a menu should open it instead of running OnClick")
	}
	// The menu hangs below the button, right edges aligned.
	f := mst.menu.overlay().Frame
	if f.Y != el.Frame.Y+el.Frame.H || f.X+f.W != el.Frame.X+el.Frame.W {
		t.Errorf("menu frame = %+v, want right-aligned below %+v", f, el.Frame)
	}

	c.BeginFrame(400, 300, nil, nil)
	n.Layout(c)
	if got := len(c.Overlays()); got != 1 {
		t.Fatalf("open menu should register an overlay, got %d", got)
	}

	clickAt(el, el.Frame.X+16, el.Frame.Y+16)
	if mst.menu.Open {
		t.Fatal("a second click should close the menu")
	}
}

func TestIconButtonMenuStaysInView(t *testing.T) {
	c := New(nil, NewFocusScope(), nil)
	n := IconButton("left", icons.Ellipsis).Menu([]MenuItem{{Label: "Delete"}})
	c.BeginFrame(400, 300, nil, nil)
	el := n.Layout(c)
	el.Frame = render.Rect{X: 4, Y: 10, W: 32, H: 32}

	clickAt(el, 20, 26)
	mst := c.Widget("left-menu", func() any { return &dropdownState{} }).(*dropdownState)
	if f := mst.menu.overlay().Frame; f.X < 0 {
		t.Errorf("menu opened off the left edge: %+v", f)
	}
}

func TestIconButtonMenuDisabled(t *testing.T) {
	c := New(nil, NewFocusScope(), nil)
	n := IconButton("off", icons.Ellipsis).Disabled(true).Menu([]MenuItem{{Label: "Delete"}})
	el := layoutIconButtonForTest(c, n)
	clickAt(el, el.Frame.X+16, el.Frame.Y+16)
	mst := c.Widget("off-menu", func() any { return &dropdownState{} }).(*dropdownState)
	if mst.menu.Open {
		t.Fatal("a disabled button should not open its menu")
	}
}

func TestIconButtonWithoutMenu(t *testing.T) {
	c := New(nil, NewFocusScope(), nil)
	clicked := false
	n := IconButton("plain", icons.Plus).OnClick(func() { clicked = true })
	el := layoutIconButtonForTest(c, n)
	clickAt(el, el.Frame.X+16, el.Frame.Y+16)
	if !clicked {
		t.Fatal("OnClick should run on a button without a menu")
	}
	if got := len(c.Overlays()); got != 0 {
		t.Fatalf("no menu, so no overlay; got %d", got)
	}
}
