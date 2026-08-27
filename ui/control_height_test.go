package ui

import (
	"testing"

	"github.com/mirzakhany/yoga/icons"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
	"github.com/mirzakhany/yoga/theme"
)

func TestCompactControlsShareControlHeight(t *testing.T) {
	text, err := shape.NewEngine(1, false)
	if err != nil {
		t.Fatal(err)
	}
	sheet := render.NewSpriteSheet(text.Atlas)
	SetFrameResources(text, sheet, nil)

	th := theme.Current()
	want := th.Metrics.ControlHeight
	c := New(text, NewFocusScope(), nil)
	c.SetIcons(sheet)
	c.BeginFrame(900, 80, nil, nil)

	opts := []SelectOption{
		{Label: "GET", Value: "GET"},
		{Label: "POST", Value: "POST"},
	}
	row := Row(
		TextField("tf", "").Placeholder("url").Width(200),
		Select("sel", opts).Width(100).Selected(0),
		Button("btn", Text("Send")).Primary(),
		IconButton("ib", icons.Settings),
		Segmented("seg",
			SegmentItem{Icon: icons.LayoutPanelLeft, Value: "h"},
			SegmentItem{Icon: icons.LayoutPanelTop, Value: "v"},
		).Selected(0),
		Dropdown("dd", "Actions", []MenuItem{{Label: "Copy"}}),
	).Gap(th.Spacing.S)
	el := row.Layout(c)
	root := layout.New(layout.Box(), el)
	root.Calculate(900, 80)

	if len(el.Children) != 6 {
		t.Fatalf("row children: got %d want 6", len(el.Children))
	}
	names := []string{"TextField", "Select", "Button", "IconButton", "Segmented", "Dropdown"}
	for i, child := range el.Children {
		if child.Frame.H != want {
			t.Errorf("%s height: got %v want %v (ControlHeight)", names[i], child.Frame.H, want)
		}
	}
}
