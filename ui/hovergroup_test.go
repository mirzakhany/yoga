package ui

import (
	"testing"

	"github.com/mirzakhany/yoga/icons"
	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/render"
)

func TestHoverGroupTracksPointer(t *testing.T) {
	c := New(nil, NewFocusScope(), nil)
	var seen bool
	row := HoverGroup("row", func(h bool) View { seen = h; return Row() })
	c.BeginFrame(400, 300, nil, nil)
	el := row.Layout(c)
	el.Frame = render.Rect{X: 0, Y: 0, W: 200, H: 30}
	if seen {
		t.Fatal("group should start unhovered")
	}

	el.OnMouse(el, &input.Mouse{X: 50, Y: 10})
	c.BeginFrame(400, 300, nil, nil)
	row.Layout(c)
	if !seen {
		t.Fatal("pointer over the group should report hovered")
	}

	el.OnMouse(el, &input.Mouse{X: 50, Y: 100})
	c.BeginFrame(400, 300, nil, nil)
	row.Layout(c)
	if seen {
		t.Fatal("pointer off the group should report unhovered")
	}
}

func TestIconButtonRevealHidesUntilShown(t *testing.T) {
	c := New(nil, NewFocusScope(), nil)
	clicked := false
	el := layoutIconButtonForTest(c, IconButton("act", icons.Ellipsis).Reveal(false).OnClick(func() { clicked = true }))
	clickAt(el, el.Frame.X+16, el.Frame.Y+16)
	if clicked {
		t.Fatal("a hidden button should let clicks through")
	}

	el = layoutIconButtonForTest(c, IconButton("act", icons.Ellipsis).Reveal(true).OnClick(func() { clicked = true }))
	clickAt(el, el.Frame.X+16, el.Frame.Y+16)
	if !clicked {
		t.Fatal("a shown button should take clicks")
	}
}

func TestIconButtonRevealKeepsOpenMenu(t *testing.T) {
	c := New(nil, NewFocusScope(), nil)
	items := []MenuItem{{Label: "Delete"}}
	el := layoutIconButtonForTest(c, IconButton("more", icons.Ellipsis).Reveal(true).Menu(items))
	clickAt(el, el.Frame.X+16, el.Frame.Y+16)

	layoutIconButtonForTest(c, IconButton("more", icons.Ellipsis).Reveal(false).Menu(items))
	if got := len(c.Overlays()); got != 1 {
		t.Fatalf("an open menu should stay while its button is hidden, got %d overlays", got)
	}
}

func TestIconButtonTooltipHidesWhileMenuOpen(t *testing.T) {
	c := New(nil, NewFocusScope(), nil)
	n := func() *Node {
		return IconButton("more", icons.Ellipsis).Tooltip("More").Menu([]MenuItem{{Label: "Delete"}})
	}
	el := layoutIconButtonForTest(c, n())
	tip := c.Widget("more#tip", nil).(*tooltipState)
	tip.hovered, tip.visible = true, true

	clickAt(el, el.Frame.X+16, el.Frame.Y+16)
	layoutIconButtonForTest(c, n())
	if tip.visible || tip.hovered {
		t.Fatal("the tooltip should hide while the menu is open")
	}
	if got := len(c.Overlays()); got != 1 {
		t.Fatalf("want only the menu overlay, got %d", got)
	}
}
