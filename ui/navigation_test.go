package ui

import (
	"testing"

	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
)

func TestNavigationSelection(t *testing.T) {
	text, err := shape.NewEngine(1, false)
	if err != nil {
		t.Fatal(err)
	}
	sheet := render.NewSpriteSheet(text.Atlas)
	SetFrameResources(text, sheet, nil)

	nav := NewNavigation(NavVertical, NavIconLeft)

	i0 := nav.Add(NavItem{ID: "editor", Label: "Editor", Icon: "edit"})
	i1 := nav.Add(NavItem{ID: "components", Label: "Components", Icon: "code"})
	if i0 != 0 || i1 != 1 {
		t.Fatalf("indices: %d %d", i0, i1)
	}
	if nav.IndexOfID("components") != 1 {
		t.Fatal("IndexOfID failed")
	}

	var gotIndex int
	var gotID string
	nav.OnSelect = func(index int, id string) {
		gotIndex = index
		gotID = id
	}
	nav.Select(1)
	if nav.Selected != 1 || nav.SelectedID() != "components" {
		t.Fatalf("selected: %d %q", nav.Selected, nav.SelectedID())
	}
	if gotIndex != 1 || gotID != "components" {
		t.Fatalf("OnSelect: %d %q", gotIndex, gotID)
	}

	nav.Select(1) // no duplicate callback
	if gotIndex != 1 {
		t.Fatal("Select same index should not fire again")
	}

	if !nav.SelectID("editor") {
		t.Fatal("SelectID not found")
	}
	if nav.Selected != 0 {
		t.Fatalf("SelectID: got %d", nav.Selected)
	}
}

func TestNavigationVerticalLayout(t *testing.T) {
	text, err := shape.NewEngine(1, false)
	if err != nil {
		t.Fatal(err)
	}
	sheet := render.NewSpriteSheet(text.Atlas)
	SetFrameResources(text, sheet, nil)

	nav := NewNavigation(NavVertical, NavIconLeft)
	nav.El.Style = nav.El.Style.W(160)
	nav.Add(NavItem{Label: "One", Icon: "edit"})
	nav.Add(NavItem{Label: "Two", Icon: "code"})
	nav.El.Calculate(160, 200)

	if len(nav.itemEls) != 2 {
		t.Fatalf("item count: %d", len(nav.itemEls))
	}
	a, b := nav.itemEls[0], nav.itemEls[1]
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

	left := NewNavigation(NavHorizontal, NavIconLeft)
	top := NewNavigation(NavHorizontal, NavIconTop)
	left.Add(NavItem{Label: "Item", Icon: "edit"})
	top.Add(NavItem{Label: "Item", Icon: "edit"})

	hLeft := left.itemHeight()
	hTop := top.itemHeight()
	if hTop <= hLeft {
		t.Fatalf("icon-top should be taller: left=%.1f top=%.1f", hLeft, hTop)
	}
}
