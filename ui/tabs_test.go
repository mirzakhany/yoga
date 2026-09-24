package ui

import (
	"fmt"
	"testing"

	"github.com/mirzakhany/yoga/icons"
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

func manyTabs(n int) []TabModel {
	tabs := make([]TabModel, n)
	for i := range tabs {
		tabs[i] = TabModel{Title: fmt.Sprintf("request %d", i)}
	}
	return tabs
}

func layoutTabsW(c *Ctx, n *Node, w float32) *layout.Element {
	el := n.Layout(c)
	el.Calculate(w, 32)
	return el
}

func tabsStateOf(c *Ctx, id string) *tabsState {
	return c.Widget(id, func() any { return nil }).(*tabsState)
}

func TestTabsFitNoOverflow(t *testing.T) {
	c, cleanup := setupTabsTest(t)
	defer cleanup()

	tabs := manyTabs(2)
	el := layoutTabsW(c, Tabs("t", tabs), 800)
	g := tabsStateOf(c, "t").fit(el, tabs, 0, true)
	if g.overflows() {
		t.Fatalf("two tabs in 800px should not overflow: %+v", g.overflow)
	}
	if g.view.W != el.Frame.W {
		t.Fatalf("viewport should span the strip: got %v want %v", g.view.W, el.Frame.W)
	}
}

func TestTabsOverflowCountsHidden(t *testing.T) {
	c, cleanup := setupTabsTest(t)
	defer cleanup()

	tabs := manyTabs(12)
	el := layoutTabsW(c, Tabs("t", tabs), 300)
	g := tabsStateOf(c, "t").fit(el, tabs, 0, true)
	if !g.overflows() {
		t.Fatal("12 tabs in 300px should overflow")
	}
	if g.view.X+g.view.W != g.overflow.X {
		t.Fatalf("overflow button should sit right of the viewport: view %v button %v", g.view, g.overflow)
	}
	visible := len(tabs) - g.hidden()
	if visible < 1 || g.hidden() < 1 {
		t.Fatalf("expected some visible and some hidden tabs, hidden=%d", g.hidden())
	}
	// A click past the viewport lands on the button, not on a clipped tab.
	if i := g.tabAt(g.overflow.X+2, g.overflow.Y+4); i != -1 {
		t.Fatalf("tabAt over the overflow button: got %d want -1", i)
	}
}

func TestTabsRevealsActiveTab(t *testing.T) {
	c, cleanup := setupTabsTest(t)
	defer cleanup()

	tabs := manyTabs(12)
	el := layoutTabsW(c, Tabs("t", tabs).Selected(11), 300)
	g := tabsStateOf(c, "t").fit(el, tabs, 11, true)
	last := g.ext[11]
	if last.x < g.view.X || last.x+last.w > g.view.X+g.view.W+0.5 {
		t.Fatalf("active tab not in view: tab %v..%v view %v", last.x, last.x+last.w, g.view)
	}
	if g.ext[0].x >= g.view.X {
		t.Fatal("first tab should be scrolled out of view")
	}
}

func TestTabsWheelScrollsAndKeepsOffset(t *testing.T) {
	c, cleanup := setupTabsTest(t)
	defer cleanup()

	tabs := manyTabs(12)
	el := layoutTabsW(c, Tabs("t", tabs), 300)
	st := tabsStateOf(c, "t")
	y := el.Frame.Y + el.Frame.H/2
	m := &input.Mouse{X: el.Frame.X + 20, Y: y, ScrollY: -1}
	el.OnMouse(el, m)
	if !m.Consumed || m.ScrollY != 0 {
		t.Fatal("wheel over an overflowing strip should be consumed")
	}
	if st.scrollX <= 0 {
		t.Fatalf("wheel down should scroll right, scrollX=%v", st.scrollX)
	}
	// A later frame with the same active tab keeps the wheel offset.
	before := st.scrollX
	st.fit(el, tabs, 0, true)
	if st.scrollX != before {
		t.Fatalf("fit reset the wheel offset: %v -> %v", before, st.scrollX)
	}
	// Scrolling far right clamps to the end of the strip.
	el.OnMouse(el, &input.Mouse{X: el.Frame.X + 20, Y: y, ScrollY: -1000})
	g := st.fit(el, tabs, 0, true)
	if want := g.content - g.view.W; st.scrollX != want {
		t.Fatalf("scroll not clamped: got %v want %v", st.scrollX, want)
	}
}

func TestTabsWheelIgnoredWhenFitting(t *testing.T) {
	c, cleanup := setupTabsTest(t)
	defer cleanup()

	tabs := manyTabs(2)
	el := layoutTabsW(c, Tabs("t", tabs), 800)
	m := &input.Mouse{X: el.Frame.X + 20, Y: el.Frame.Y + 4, ScrollY: -1}
	el.OnMouse(el, m)
	if m.Consumed || m.ScrollY == 0 {
		t.Fatal("wheel over a fitting strip should pass through to the page")
	}
}

func TestTabsOverflowMenuListsHiddenTabs(t *testing.T) {
	c, cleanup := setupTabsTest(t)
	defer cleanup()

	tabs := manyTabs(12)
	var selected = -1
	el := layoutTabsW(c, Tabs("t", tabs).Selected(6).OnSelectItem(func(i int, _ string) { selected = i }), 300)
	st := tabsStateOf(c, "t")
	g := st.fit(el, tabs, 6, true)
	el.OnMouse(el, &input.Mouse{X: g.overflow.X + g.overflow.W/2, Y: g.overflow.Y + 4, Pressed: true, Down: true})
	if !st.menu.Open {
		t.Fatal("clicking the overflow button should open the hidden tab list")
	}
	// Expect the left-hidden tabs, a separator, then the right-hidden tabs,
	// in strip order.
	var want []string
	for i := range tabs {
		if g.hiddenLeft(i) {
			want = append(want, tabs[i].Title)
		}
	}
	if len(want) == 0 {
		t.Fatal("test needs tabs hidden on the left")
	}
	want = append(want, "---")
	right := 0
	for i := range tabs {
		if !g.hiddenLeft(i) && g.hiddenRight(i) {
			want = append(want, tabs[i].Title)
			right++
		}
	}
	if right == 0 {
		t.Fatal("test needs tabs hidden on the right")
	}
	var got []string
	for _, it := range st.menu.items {
		if it.Separator {
			got = append(got, "---")
			continue
		}
		if it.Label == tabs[6].Title {
			t.Fatal("the active tab is in view and should not be listed")
		}
		got = append(got, it.Label)
	}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("menu items:\n got %v\nwant %v", got, want)
	}
	if len(got)-1 != g.hidden() {
		t.Fatalf("list has %d tabs, button counts %d", len(got)-1, g.hidden())
	}
	st.menu.items[len(st.menu.items)-1].OnSelect()
	if selected != 11 {
		t.Fatalf("choosing the last hidden tab: got %d want 11", selected)
	}
}

func TestTabsOverflowMenuNoSeparatorOnOneSide(t *testing.T) {
	c, cleanup := setupTabsTest(t)
	defer cleanup()

	tabs := manyTabs(12)
	el := layoutTabsW(c, Tabs("t", tabs), 300)
	st := tabsStateOf(c, "t")
	g := st.fit(el, tabs, 0, true)
	el.OnMouse(el, &input.Mouse{X: g.overflow.X + g.overflow.W/2, Y: g.overflow.Y + 4, Pressed: true, Down: true})
	if len(st.menu.items) != g.hidden() {
		t.Fatalf("items: got %d want %d", len(st.menu.items), g.hidden())
	}
	for _, it := range st.menu.items {
		if it.Separator || it.Label == tabs[0].Title {
			t.Fatalf("unexpected item %+v", it)
		}
	}
}

func TestTabsContextMenu(t *testing.T) {
	c, cleanup := setupTabsTest(t)
	defer cleanup()

	tabs := manyTabs(3)
	asked, selected := -1, -1
	el := layoutTabsW(c, Tabs("t", tabs).
		OnSelectItem(func(i int, _ string) { selected = i }).
		OnTabContextMenu(func(i int) []MenuItem {
			asked = i
			return []MenuItem{{Label: "Close"}}
		}), 800)
	st := tabsStateOf(c, "t")
	ext := tabExtents(el, tabs, true)
	m := &input.Mouse{X: ext[1].x + 4, Y: el.Frame.Y + 4, RightPressed: true, RightDown: true}
	el.OnMouse(el, m)
	if asked != 1 {
		t.Fatalf("context menu asked for tab %d want 1", asked)
	}
	if !st.menu.Open || !m.Consumed {
		t.Fatal("right-click should open the tab menu")
	}
	if selected != -1 {
		t.Fatal("right-click should not activate the tab")
	}
}

func TestTabsContextMenuEmptyShowsNothing(t *testing.T) {
	c, cleanup := setupTabsTest(t)
	defer cleanup()

	tabs := manyTabs(3)
	el := layoutTabsW(c, Tabs("t", tabs).OnTabContextMenu(func(int) []MenuItem { return nil }), 800)
	ext := tabExtents(el, tabs, true)
	el.OnMouse(el, &input.Mouse{X: ext[0].x + 4, Y: el.Frame.Y + 4, RightPressed: true, RightDown: true})
	if tabsStateOf(c, "t").menu.Open {
		t.Fatal("no items should mean no menu")
	}
}

// A tab icon widens the tab by the icon and its gap, so the title never runs
// under it.
func TestTabsIconWidensTab(t *testing.T) {
	c, cleanup := setupTabsTest(t)
	defer cleanup()

	plain := []TabModel{{Title: "Users"}}
	withIcon := []TabModel{{Title: "Users", Icon: icons.Globe}}
	a := tabExtents(layoutTabsEl(c, Tabs("a", plain)), plain, true)
	b := tabExtents(layoutTabsEl(c, Tabs("b", withIcon)), withIcon, true)
	th := theme.Current()
	if got, want := b[0].w-a[0].w, th.Metrics.IconSizeSM+th.Spacing.S; got != want {
		t.Fatalf("icon added %v px, want %v", got, want)
	}
}
