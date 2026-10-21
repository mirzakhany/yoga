package ui

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
)

func TestFuzzyMatchRanges(t *testing.T) {
	got, ok := FuzzyMatch("List todos", "lstd")
	want := [][2]int{{0, 1}, {2, 4}, {7, 8}} // l, st, d
	if !ok || !reflect.DeepEqual(got, want) {
		t.Fatalf("FuzzyMatch = %v, %v; want %v", got, ok, want)
	}
	if _, ok := FuzzyMatch("List todos", "xyz"); ok {
		t.Fatal("non-match reported as a match")
	}
	got, _ = FuzzyMatch("Ünïcode", "ÏC") // ï is bytes 3-5, c is 5
	if want := [][2]int{{3, 6}}; !reflect.DeepEqual(got, want) {
		t.Fatalf("multi-byte ranges = %v, want %v", got, want)
	}
	// Ranking is unchanged by the refactor.
	a, _ := subsequenceScore("List todos", "lis")
	b, _ := subsequenceScore("Close all lists", "lis")
	if a <= b {
		t.Fatalf("prefix match should rank higher: %d vs %d", a, b)
	}
}

// paletteFrame builds one frame with the palette open over 12 entries.
func paletteFrame(t *testing.T, c *Ctx, setup func(h *CommandsHost)) *layout.Element {
	t.Helper()
	return BuildFrame(c, func(c *Ctx) View {
		h := c.Commands()
		setup(h)
		return Column()
	}, 900, 560, &input.Mouse{X: -1, Y: -1}, &input.Keyboard{})
}

func TestCommandsCompactFitsTenRows(t *testing.T) {
	c := newTestCtx(t)
	setup := func(h *CommandsHost) {
		h.Density(DensityCompact).HintFooter(true)
		for i := range 12 {
			h.Register(Item(fmt.Sprintf("r%d", i)).Title(fmt.Sprintf("Request %d", i)).Trailing("Todos").
				Leading(Badge("GET")))
		}
	}
	c.Commands().Show()
	paletteFrame(t, c, setup)
	c.EndFrame()
	paletteFrame(t, c, setup)
	h := c.Commands()
	th := c.Theme()
	if rh := h.rowHeight(th); rh != th.Metrics.ControlHeight {
		t.Fatalf("compact row %v, want %v", rh, th.Metrics.ControlHeight)
	}
	if n := int(h.listView / h.rowHeight(th)); n < 10 {
		t.Fatalf("compact palette shows %d rows in %v px, want >= 10", n, h.listView)
	}
	if h.panel.Frame.H > 560 {
		t.Fatalf("panel %v taller than the window", h.panel.Frame.H)
	}

	h.Density(DensityDefault)
	if n := int(h.listView / commandRowHeight(th)); n >= 10 {
		t.Fatalf("default rows unexpectedly fit %d", n)
	}
}

func TestCommandsAltRunsAndCloses(t *testing.T) {
	c := New(nil, NewFocusScope(), nil)
	c.BeginFrame(800, 600, nil, nil)
	ran := ""
	h := c.Commands()
	h.Register(
		Item("a").Title("List todos").Run(func() { ran = "open" }).Alt("⌘↵", func() { ran = "send" }),
		Item("b").Title("Create todo").Run(func() { ran = "b" }),
	)
	h.Show()
	h.Layout(c)
	h.FilterKeys([]input.KeyEvent{{Key: input.KeyEnter, Mods: input.ModCtrl}})
	if ran != "send" || h.Open {
		t.Fatalf("ran=%q open=%v; want the alt action and a closed palette", ran, h.Open)
	}

	// An entry without Alt ignores the combo rather than running it.
	ran = ""
	h.Show()
	h.Layout(c)
	h.FilterKeys([]input.KeyEvent{{Key: input.KeyDown}})
	h.FilterKeys([]input.KeyEvent{{Key: input.KeyEnter, Mods: input.ModSuper}})
	if ran != "b" {
		t.Fatalf("ran=%q, want plain Enter behavior for an entry without Alt", ran)
	}
}

func TestCommandsTitleHighlightsMatches(t *testing.T) {
	text, err := shape.NewEngine(1, false)
	if err != nil {
		t.Skip(err)
	}
	c := New(text, NewFocusScope(), nil)
	c.BeginFrame(800, 600, nil, nil)
	v := matchTitle("List todos", "lstd", false)
	mt, ok := v.(*matchText)
	if !ok {
		t.Fatalf("matching title is %T, want *matchText", v)
	}
	el := mt.Layout(c)
	el.Frame = render.Rect{W: 400, H: 30}
	var dl render.DrawList
	el.Paint(&dl, text)
	if len(dl.Vertices) == 0 {
		t.Fatal("highlighted title painted nothing")
	}
	if _, ok := matchTitle("List todos", "", false).(*Node); !ok {
		t.Fatal("an empty query should render a plain title")
	}
}
