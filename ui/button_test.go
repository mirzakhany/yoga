package ui

import (
	"testing"

	"github.com/mirzakhany/yoga/icons"
	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
	"github.com/mirzakhany/yoga/theme"
)

func layoutButtonForTest(c *Ctx, n *Node) *layout.Element {
	c.BeginFrame(400, 300, nil, nil)
	el := n.Layout(c)
	root := layout.New(layout.Box(), el)
	root.Calculate(400, 300)
	return el
}

func clickButton(el *layout.Element) {
	m := &input.Mouse{X: el.Frame.X + el.Frame.W/2, Y: el.Frame.Y + el.Frame.H/2, Pressed: true, Down: true}
	el.OnMouse(el, m)
	m.Pressed = false
	m.Down = false
	m.Released = true
	el.OnMouse(el, m)
}

func TestGhostButtonCompactMetrics(t *testing.T) {
	text, err := shape.NewEngine(1, false)
	if err != nil {
		t.Fatal(err)
	}
	c := New(text, NewFocusScope(), nil)
	th := theme.Current()
	c.BeginFrame(400, 300, nil, nil)

	ghost := Button("ghost", Text("Ln 12, Col 4")).Ghost().Layout(c)
	secondary := Button("secondary", Text("Ln 12, Col 4")).Layout(c)
	row := layout.New(layout.Box().Direction(layout.Row).Gap(th.Spacing.S).AlignItems(layout.AlignCenter), ghost, secondary)
	row.Calculate(400, 300)

	if ghost.Frame.H >= secondary.Frame.H {
		t.Fatalf("ghost height should be smaller than secondary: ghost=%v secondary=%v", ghost.Frame.H, secondary.Frame.H)
	}
	if ghost.Frame.W >= secondary.Frame.W {
		t.Fatalf("ghost width should be smaller than secondary: ghost=%v secondary=%v", ghost.Frame.W, secondary.Frame.W)
	}
	wantH := th.Typography.Body.LineHeight
	if ghost.Frame.H != wantH {
		t.Fatalf("ghost height: got %v want %v", ghost.Frame.H, wantH)
	}
}

func TestGhostButtonOnClick(t *testing.T) {
	c := New(nil, NewFocusScope(), nil)
	clicked := false
	el := layoutButtonForTest(c, Button("ghost-click", Text("Action")).Ghost().OnClick(func() { clicked = true }))
	clickButton(el)
	if !clicked {
		t.Fatal("expected OnClick on ghost button")
	}
}

func TestGhostHoverFillSpec(t *testing.T) {
	c := New(nil, NewFocusScope(), nil)
	th := theme.Current()

	plain := Button("plain", Text("x")).Ghost().buttonSpec(c)
	hover := Button("hover", Text("x")).Ghost().HoverFill().buttonSpec(c)

	rest := plain.resolve(th, interactState{})
	if rest.hasBg && rest.bg.A > 0 {
		t.Fatal("plain ghost rest should have no background")
	}
	hov := hover.resolve(th, interactState{hovered: true})
	if !hov.hasBg || hov.bg.A == 0 {
		t.Fatal("ghost HoverFill should paint background on hover")
	}
}

func TestGhostHoverFillMetrics(t *testing.T) {
	text, err := shape.NewEngine(1, false)
	if err != nil {
		t.Fatal(err)
	}
	c := New(text, NewFocusScope(), nil)
	th := theme.Current()
	c.BeginFrame(400, 300, nil, nil)

	plain := Button("plain", Text("Action")).Ghost().Layout(c)
	fill := Button("fill", Text("Action")).Ghost().HoverFill().Layout(c)
	row := layout.New(layout.Box().Direction(layout.Row).Gap(th.Spacing.S).AlignItems(layout.AlignCenter), plain, fill)
	row.Calculate(400, 300)

	if fill.Frame.H <= plain.Frame.H {
		t.Fatalf("HoverFill ghost should be taller than plain ghost: fill=%v plain=%v", fill.Frame.H, plain.Frame.H)
	}
	if fill.Frame.W <= plain.Frame.W {
		t.Fatalf("HoverFill ghost should be wider than plain ghost: fill=%v plain=%v", fill.Frame.W, plain.Frame.W)
	}
}

func TestGhostButtonIconAndTooltip(t *testing.T) {
	c := New(nil, NewFocusScope(), nil)
	sheet := render.NewSpriteSheet(nil)
	c.SetIcons(sheet)

	n := Button("ghost-icon", Caption("UTF-8")).
		Ghost().
		IconStart(icons.ChevronDown).
		Tooltip("Select encoding")
	el := layoutButtonForTest(c, n)
	if el == nil {
		t.Fatal("nil element")
	}
	if el.Frame.W <= 0 || el.Frame.H <= 0 {
		t.Fatalf("expected positive size, got %v", el.Frame)
	}
}
