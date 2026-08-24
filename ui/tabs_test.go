package ui

import (
	"testing"

	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
	"github.com/mirzakhany/yoga/theme"
)

func setupTabsTest(t *testing.T) (*Ctx, func()) {
	t.Helper()
	text, err := shape.NewEngine(1, false)
	if err != nil {
		t.Fatal(err)
	}
	sheet := render.NewSpriteSheet(text.Atlas)
	SetFrameResources(text, sheet, nil)
	c := New(text, NewFocusScope(), nil)
	c.SetIcons(sheet)
	return c, func() { SetFrameResources(nil, nil, nil) }
}

func layoutTabsEl(c *Ctx, n *Node) *layout.Element {
	el := n.Layout(c)
	el.Calculate(800, 32)
	return el
}

func TestTabsClosableDefaultHasCloseWidth(t *testing.T) {
	c, cleanup := setupTabsTest(t)
	defer cleanup()

	tabs := []TabModel{{Title: "Body"}, {Title: "Headers"}}
	el := layoutTabsEl(c, Tabs("t", tabs))
	ext := tabExtents(el, tabs, true)
	if ext[0].close.W <= 0 {
		t.Fatalf("expected close rect width, got %v", ext[0].close)
	}
}

func TestTabsClosableFalseNoCloseWidth(t *testing.T) {
	c, cleanup := setupTabsTest(t)
	defer cleanup()

	tabs := []TabModel{{Title: "Body"}, {Title: "Headers"}}
	el := layoutTabsEl(c, Tabs("t", tabs).Closable(false))
	extClosable := tabExtents(el, tabs, true)
	ext := tabExtents(el, tabs, false)
	if ext[0].close.W != 0 {
		t.Fatalf("expected empty close rect, got %v", ext[0].close)
	}
	closeW := theme.Current().Metrics.IconSizeMD
	if ext[0].w+closeW != extClosable[0].w {
		t.Fatalf("non-closable tab should be narrower by closeW: got %v vs %v", ext[0].w, extClosable[0].w)
	}
}

func TestTabsCloseCallback(t *testing.T) {
	c, cleanup := setupTabsTest(t)
	defer cleanup()

	tabs := []TabModel{{Title: "Body"}, {Title: "Headers"}}
	var closed int
	el := layoutTabsEl(c, Tabs("t", tabs).OnTabClose(func(i int) { closed = i }))
	ext := tabExtents(el, tabs, true)
	cx := ext[0].close.X + ext[0].close.W/2
	cy := ext[0].close.Y + ext[0].close.H/2
	el.OnMouse(el, &input.Mouse{X: cx, Y: cy, Pressed: true, Down: true})
	if closed != 0 {
		t.Fatalf("OnTabClose: got %d want 0", closed)
	}
}

func TestTabsClosableFalseSelectsOnTrailingEdge(t *testing.T) {
	c, cleanup := setupTabsTest(t)
	defer cleanup()

	tabs := []TabModel{{Title: "Body"}, {Title: "Headers"}}
	var closed, selected int
	el := layoutTabsEl(c, Tabs("t", tabs).
		Closable(false).
		OnTabClose(func(i int) { closed = i }).
		OnSelectItem(func(i int, _ string) { selected = i }))
	ext := tabExtents(el, tabs, false)
	x := ext[0].x + ext[0].w - 1
	y := el.Frame.Y + el.Frame.H/2
	el.OnMouse(el, &input.Mouse{X: x, Y: y, Pressed: true, Down: true})
	if closed != 0 {
		t.Fatalf("OnTabClose should not fire when Closable(false), got %d", closed)
	}
	if selected != 0 {
		t.Fatalf("OnSelectItem: got %d want 0", selected)
	}
	x = ext[1].x + ext[1].w - 1
	el.OnMouse(el, &input.Mouse{X: x, Y: y, Pressed: true, Down: true})
	if selected != 1 {
		t.Fatalf("OnSelectItem on tab 1: got %d want 1", selected)
	}
}
