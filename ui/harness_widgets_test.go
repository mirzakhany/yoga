package ui

import (
	"testing"

	"github.com/mirzakhany/yoga/icons"
	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
)

func TestTabsBusyAndStatusWiden(t *testing.T) {
	c, cleanup := setupTabsTest(t)
	defer cleanup()

	plain := []TabModel{{Title: "Session"}}
	busy := []TabModel{{Title: "Session", Busy: true, Status: render.Color{R: 1, A: 1}}}
	el := layoutTabsEl(c, Tabs("t", plain))
	w0 := tabExtents(el, plain, true)[0].w
	el = layoutTabsEl(c, Tabs("t2", busy))
	w1 := tabExtents(el, busy, true)[0].w
	if w1 <= w0 {
		t.Fatalf("busy tab with status should be wider: %v <= %v", w1, w0)
	}
}

func TestMenuIconsIndentLabels(t *testing.T) {
	textFieldTestEnv(t)
	plain := NewMenu(0, []MenuItem{{Label: "Remove"}})
	icon := NewMenu(0, []MenuItem{{Label: "Remove", Icon: icons.Trash2, Danger: true}})
	if icon.contentWidth() <= plain.contentWidth() {
		t.Fatalf("icon menu should be wider: %v <= %v", icon.contentWidth(), plain.contentWidth())
	}
}

func TestClickableCardClick(t *testing.T) {
	c, cleanup := setupTabsTest(t)
	defer cleanup()

	clicked := 0
	el := layoutButtonForTest(c, ClickableCard("card", "Title", "Sub", Text("body")).OnClick(func() { clicked++ }).Width(200))
	clickButton(el)
	if clicked != 1 {
		t.Fatalf("card click: got %d want 1", clicked)
	}
}

func TestCardActionClickDoesNotClickCard(t *testing.T) {
	c, cleanup := setupTabsTest(t)
	defer cleanup()

	card, action := 0, 0
	el := layoutButtonForTest(c, ClickableCard("card", "Title", "", nil).
		Actions(IconButton("act", icons.X).OnClick(func() { action++ })).
		OnClick(func() { card++ }).Width(200))
	// The action button is the last child of the title row.
	row := el.Children[0]
	btn := row.Children[len(row.Children)-1]
	m := &input.Mouse{X: btn.Frame.X + btn.Frame.W/2, Y: btn.Frame.Y + btn.Frame.H/2, Pressed: true, Down: true}
	layout.Dispatch(el, m)
	m = &input.Mouse{X: m.X, Y: m.Y, Released: true}
	layout.Dispatch(el, m)
	if action != 1 || card != 0 {
		t.Fatalf("action=%d card=%d, want 1 and 0", action, card)
	}
}
