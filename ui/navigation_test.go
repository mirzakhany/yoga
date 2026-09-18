package ui

import (
	"testing"

	"github.com/mirzakhany/yoga/icons"
	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
)

func TestNavigationVerticalLayout(t *testing.T) {
	text, err := shape.NewEngine(1, false)
	if err != nil {
		t.Fatal(err)
	}
	sheet := render.NewSpriteSheet(text.Atlas)
	SetFrameResources(text, sheet, nil)

	c := New(text, NewFocusScope(), nil)
	c.SetIcons(sheet)
	el := Nav("n", NavVertical, NavIconLeft,
		NavItem{Label: "One", Icon: icons.Pencil},
		NavItem{Label: "Two", Icon: icons.Code},
	).Width(160).Layout(c)
	el.Calculate(160, 200)

	if len(el.Children) != 2 {
		t.Fatalf("item count: %d", len(el.Children))
	}
	a, b := el.Children[0], el.Children[1]
	if b.Frame.Y <= a.Frame.Y {
		t.Fatalf("items overlap vertically: a=%v b=%v", a.Frame, b.Frame)
	}
}

func TestNavigationHorizontalIconTopHeight(t *testing.T) {
	text, err := shape.NewEngine(1, false)
	if err != nil {
		t.Fatal(err)
	}
	sheet := render.NewSpriteSheet(text.Atlas)
	SetFrameResources(text, sheet, nil)

	hLeft := navItemHeight(NavIconLeft)
	hTop := navItemHeight(NavIconTop)
	if hTop <= hLeft {
		t.Fatalf("icon-top should be taller: left=%.1f top=%.1f", hLeft, hTop)
	}
}

func TestNavigationSelectCallback(t *testing.T) {
	text, err := shape.NewEngine(1, false)
	if err != nil {
		t.Fatal(err)
	}
	sheet := render.NewSpriteSheet(text.Atlas)
	SetFrameResources(text, sheet, nil)
	c := New(text, NewFocusScope(), nil)
	c.SetIcons(sheet)

	var got int
	var gotID string
	el := Nav("n", NavVertical, NavIconLeft,
		NavItem{ID: "editor", Label: "Editor"},
		NavItem{ID: "gallery", Label: "Gallery"},
	).Selected(0).OnSelectItem(func(i int, id string) {
		got, gotID = i, id
	}).Width(160).Layout(c)
	el.Calculate(160, 200)
	item := el.Children[1]
	m := &input.Mouse{X: item.Frame.X + 1, Y: item.Frame.Y + 1, Released: true}
	item.OnMouse(item, m)
	if got != 1 || gotID != "gallery" {
		t.Fatalf("OnSelectItem: %d %q", got, gotID)
	}
}

func TestNavigationRadiusModifier(t *testing.T) {
	text, err := shape.NewEngine(1, false)
	if err != nil {
		t.Fatal(err)
	}
	sheet := render.NewSpriteSheet(text.Atlas)
	SetFrameResources(text, sheet, nil)

	// hasCornerVertex reports whether the selected item's highlight reaches its
	// top-left corner, i.e. whether it was painted square.
	hasCornerVertex := func(n *Node) bool {
		c := New(text, NewFocusScope(), nil)
		c.SetIcons(sheet)
		el := n.Selected(0).Width(160).Layout(c)
		el.Calculate(160, 200)
		item := el.Children[0]
		dl := &render.DrawList{}
		item.Paint(dl, text)
		for _, v := range dl.Vertices {
			if v.Pos[0] == item.Frame.X && v.Pos[1] == item.Frame.Y {
				return true
			}
		}
		return false
	}
	nav := func() *Node { return Nav("n", NavVertical, NavIconLeft, NavItem{Label: "One"}) }

	if hasCornerVertex(nav()) {
		t.Fatal("default nav item should be rounded")
	}
	if !hasCornerVertex(nav().Radius(0)) {
		t.Fatal("Radius(0) should paint a square item highlight")
	}
}
