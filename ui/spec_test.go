package ui

import (
	"testing"

	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/theme"
)

func TestSpecWhenPressedOverridesHover(t *testing.T) {
	s := Background(TokenAccent).
		When(Hovered, Background(TokenAccentHover)).
		When(Pressed, Background(TokenAccentPressed))
	th := theme.Current()
	hover := s.resolve(th, interactState{hovered: true})
	press := s.resolve(th, interactState{hovered: true, pressed: true})
	if hover.bg != th.AccentHover {
		t.Fatalf("hover bg = %v want AccentHover", hover.bg)
	}
	if press.bg != th.AccentPressed {
		t.Fatalf("pressed bg = %v want AccentPressed", press.bg)
	}
}

func TestTokenResolveFollowsThemeUse(t *testing.T) {
	orig := theme.Current().Name
	t.Cleanup(func() { theme.Use(orig) })

	dark := TokenAccent.Resolve(theme.Current())
	if !theme.Use("yoga-light") {
		t.Fatal("yoga-light missing")
	}
	light := TokenAccent.Resolve(theme.Current())
	if dark == light {
		t.Fatal("Accent should change between yoga-dark and yoga-light")
	}
}

func TestWidgetStorePersistsHover(t *testing.T) {
	c := New(nil, NewFocusScope(), nil)
	mouse := &input.Mouse{}
	clicked := 0
	body := func(c *Ctx) View {
		return Button("save", Text("Save")).OnClick(func() { clicked++ })
	}
	root := BuildFrame(c, body, 400, 200, mouse, nil)
	if len(root.Children) == 0 && root.OnMouse == nil {
		// button is the root
	}
	// Find the button element.
	btn := findMouse(root)
	if btn == nil {
		t.Fatal("no OnMouse handler")
	}
	btn.OnMouse(btn, &input.Mouse{X: btn.Frame.X + 1, Y: btn.Frame.Y + 1, Pressed: true, Down: true})
	st := c.Widget("save", func() any { return &buttonState{} }).(*buttonState)
	if !st.pressed && !st.hovered {
		t.Fatal("expected hover or press after mouse")
	}
	c.EndFrame()
	_ = BuildFrame(c, body, 400, 200, mouse, nil)
	st2 := c.Widget("save", func() any { return &buttonState{} }).(*buttonState)
	if st2 != st {
		t.Fatal("store should keep the same button state across frames")
	}
}

func TestWidgetStoreGCsUnusedIDs(t *testing.T) {
	c := New(nil, NewFocusScope(), nil)
	BuildFrame(c, func(c *Ctx) View {
		return Button("a", Text("A"))
	}, 200, 100, nil, nil)
	c.EndFrame()
	BuildFrame(c, func(c *Ctx) View {
		return Button("b", Text("B"))
	}, 200, 100, nil, nil)
	c.EndFrame()
	if _, ok := c.store.items["a"]; ok {
		t.Fatal("unused id a should be GC'd")
	}
	if _, ok := c.store.items["b"]; !ok {
		t.Fatal("id b should remain")
	}
}

func findMouse(e *layout.Element) *layout.Element {
	if e == nil {
		return nil
	}
	if e.OnMouse != nil {
		return e
	}
	for _, ch := range e.Children {
		if found := findMouse(ch); found != nil {
			return found
		}
	}
	return nil
}

func TestColumnGapGrow(t *testing.T) {
	c := New(nil, NewFocusScope(), nil)
	el := Column(Text("a"), Text("b")).Gap(12).Grow(1).Layout(c)
	if el.Style.RowGap != 12 {
		t.Fatalf("gap = %v", el.Style.RowGap)
	}
	if el.Style.Grow != 1 {
		t.Fatalf("grow = %v", el.Style.Grow)
	}
}

func TestSplitterInitialSizes(t *testing.T) {
	c := New(nil, NewFocusScope(), nil)
	root := BuildFrame(c, func(_ *Ctx) View {
		return Splitter("main", Horizontal, Text("left"), Text("right")).Sizes(240, 0)
	}, 800, 400, nil, nil)
	split := root
	if len(split.Children) == 0 {
		t.Fatal("empty splitter")
	}
	// Body may be wrapped with overlays; find the row with 3 children (pane, handle, pane).
	row := findSplitRow(root)
	if row == nil || len(row.Children) != 3 {
		t.Fatalf("want pane/handle/pane, got %v children", childCount(row))
	}
	if row.Children[0].Style.Width != 240 {
		t.Fatalf("left pane width = %v want 240", row.Children[0].Style.Width)
	}
}

func findSplitRow(e *layout.Element) *layout.Element {
	if e == nil {
		return nil
	}
	if len(e.Children) == 3 {
		return e
	}
	for _, ch := range e.Children {
		if found := findSplitRow(ch); found != nil {
			return found
		}
	}
	return nil
}

func childCount(e *layout.Element) int {
	if e == nil {
		return 0
	}
	return len(e.Children)
}
