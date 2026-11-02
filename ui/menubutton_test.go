package ui

import (
	"testing"

	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
	"github.com/mirzakhany/yoga/theme"
)

func layoutMenuButtonForTest(c *Ctx, n *Node) *layout.Element {
	c.BeginFrame(400, 300, nil, nil)
	el := n.Layout(c)
	el.Frame = render.Rect{X: 10, Y: 10, W: 140, H: 32}
	return el
}

func clickAt(el *layout.Element, x, y float32) {
	m := &input.Mouse{X: x, Y: y, Pressed: true, Down: true}
	el.OnMouse(el, m)
	m.Pressed = false
	m.Down = false
	m.Released = true
	el.OnMouse(el, m)
}

func TestMenuButtonOpensMenuOnClick(t *testing.T) {
	c := New(nil, NewFocusScope(), nil)
	n := MenuButton("mb", "Export", []MenuItem{
		{Label: "CSV", OnSelect: func() {}},
	}).Primary()

	el := layoutMenuButtonForTest(c, n)
	if got := len(c.Overlays()); got != 0 {
		t.Fatalf("closed menu button should register no overlay, got %d", got)
	}

	mst := c.Widget("mb-menu", func() any { return &dropdownState{} }).(*dropdownState)
	if mst.menu == nil {
		t.Fatal("menu not allocated")
	}

	clickAt(el, el.Frame.X+20, el.Frame.Y+el.Frame.H/2)

	if !mst.menu.Open {
		t.Fatal("expected menu open after click")
	}

	c.BeginFrame(400, 300, nil, nil)
	n.Layout(c)
	if got := len(c.Overlays()); got != 1 {
		t.Fatalf("open menu button should register overlay, got %d", got)
	}
}

func TestMenuButtonSplitLabelOnClick(t *testing.T) {
	c := New(nil, NewFocusScope(), nil)
	clicked := false
	n := MenuButton("split", "Save", []MenuItem{
		{Label: "Save As", OnSelect: func() {}},
	}).Primary().OnClick(func() { clicked = true })

	el := layoutMenuButtonForTest(c, n)
	mst := c.Widget("split-menu", func() any { return &dropdownState{} }).(*dropdownState)

	clickAt(el, el.Frame.X+20, el.Frame.Y+el.Frame.H/2)

	if !clicked {
		t.Fatal("expected OnClick on label side")
	}
	if mst.menu.Open {
		t.Fatal("label click should not open menu")
	}
}

func TestMenuButtonSplitChevronOpensMenu(t *testing.T) {
	c := New(nil, NewFocusScope(), nil)
	clicked := false
	n := MenuButton("split2", "Save", []MenuItem{
		{Label: "Save As", OnSelect: func() {}},
	}).Primary().OnClick(func() { clicked = true })

	el := layoutMenuButtonForTest(c, n)
	mst := c.Widget("split2-menu", func() any { return &dropdownState{} }).(*dropdownState)

	th := c.Theme()
	chevronSlot := th.Metrics.IconSizeSM + th.Spacing.M
	chevronX := el.Frame.X + el.Frame.W - chevronSlot/2
	clickAt(el, chevronX, el.Frame.Y+el.Frame.H/2)

	if clicked {
		t.Fatal("chevron click should not run OnClick")
	}
	if !mst.menu.Open {
		t.Fatal("expected menu open after chevron click")
	}
}

func TestMenuButtonShrinkCutsLabel(t *testing.T) {
	text, err := shape.NewEngine(1, false)
	if err != nil {
		t.Fatal(err)
	}
	c := New(text, NewFocusScope(), nil)
	items := []MenuItem{{Label: "a"}}
	label := "a rather long model name that will not fit"
	for _, tc := range []struct {
		name   string
		shrink bool
	}{{"fixed", false}, {"shrink", true}} {
		c.BeginFrame(400, 300, nil, nil)
		n := MenuButton("mb-"+tc.name, label, items)
		if tc.shrink {
			n = n.Shrink(1)
		}
		el := n.Layout(c)
		row := layout.New(layout.Box().Direction(layout.Row).W(120), el)
		row.Calculate(120, 300)
		lw, _ := text.MeasureAt(label, theme.Current().Typography.Body.Size)
		if tc.shrink && (el.Frame.W > 120 || el.Frame.W <= 0) {
			t.Errorf("shrinking button width %v, want within 120", el.Frame.W)
		}
		if !tc.shrink && el.Frame.W < lw {
			t.Errorf("fixed button width %v narrower than its label %v", el.Frame.W, lw)
		}
	}
}
