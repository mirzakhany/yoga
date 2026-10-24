package ui

import (
	"testing"

	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
)

// A FitContent select showing a short option opens a menu wide enough for
// its longest one, while the trigger stays sized to the selection.
func TestSelectMenuFitsLongestOption(t *testing.T) {
	text, err := shape.NewEngine(1, false)
	if err != nil {
		t.Fatal(err)
	}
	SetFrameResources(text, render.NewSpriteSheet(text.Atlas), nil)

	c := New(text, NewFocusScope(), nil)
	c.BeginFrame(400, 80, nil, nil)

	opts := []SelectOption{{Label: "GET", Value: "GET"}, {Label: "CONNECT", Value: "CONNECT"}}
	el := Select("m", opts).FitContent().Selected(0).Layout(c)
	layout.New(layout.Box(), el).Calculate(400, 80)

	th := c.Theme()
	longW, _ := text.MeasureAt("CONNECT", th.Typography.Body.Size)
	st := c.Widget("m", func() any { return &selectState{} }).(*selectState)
	if el.Frame.W >= longW+2*th.Spacing.MNudge {
		t.Fatalf("trigger %v should fit GET, not CONNECT (%v)", el.Frame.W, longW)
	}
	if need := longW + 2*th.Spacing.MNudge; st.menu.width < need {
		t.Fatalf("menu width %v, want at least %v to show CONNECT", st.menu.width, need)
	}
}
