package ui

import (
	"testing"

	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
)

func TestSelectGrowFillsRow(t *testing.T) {
	text, err := shape.NewEngine(1, false)
	if err != nil {
		t.Fatal(err)
	}
	sheet := render.NewSpriteSheet(text.Atlas)
	SetFrameResources(text, sheet, nil)

	c := New(text, NewFocusScope(), nil)
	c.BeginFrame(400, 80, nil, nil)

	opts := []SelectOption{
		{Label: "A", Value: "a"},
		{Label: "B", Value: "b"},
	}
	btnW := float32(80)
	row := Row(
		Select("sel", opts).Selected(0).Grow(1),
		Button("btn", Text("OK")).Width(btnW),
	).Gap(8)
	el := row.Layout(c)
	root := layout.New(layout.Box(), el)
	root.Calculate(400, 80)

	if len(el.Children) != 2 {
		t.Fatalf("row children: got %d want 2", len(el.Children))
	}
	sel := el.Children[0]
	btn := el.Children[1]
	if btn.Frame.W != btnW {
		t.Fatalf("button width: got %v want %v", btn.Frame.W, btnW)
	}
	wantSelW := 400 - btnW - 8
	if sel.Frame.W != wantSelW {
		t.Fatalf("select width: got %v want %v", sel.Frame.W, wantSelW)
	}

	st := c.Widget("sel", func() any { return &selectState{} }).(*selectState)
	if st.menu == nil {
		t.Fatal("menu not allocated")
	}
	st.menu.width = triggerMenuWidth(160, sel.Frame.W)
	if st.menu.width != wantSelW {
		t.Fatalf("menu width: got %v want %v", st.menu.width, wantSelW)
	}
}
