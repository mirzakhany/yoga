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
