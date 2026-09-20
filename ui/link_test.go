package ui

import (
	"testing"

	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
	"github.com/mirzakhany/yoga/theme"
)

// A focused link keeps its label visible: the ring fills its rect, so it has
// to be painted before the glyphs, not over them.
func TestFocusedLinkPaintsRingUnderLabel(t *testing.T) {
	text, err := shape.NewEngine(1, false)
	if err != nil {
		t.Fatal(err)
	}
	c := New(text, NewFocusScope(), nil)
	c.BeginFrame(400, 300, nil, nil)
	th := theme.Current()

	el := Link("link", "chapar.rest").Layout(c)
	if el == nil || el.Paint == nil {
		t.Fatal("link should paint")
	}
	st := c.Widget("link", func() any { return &linkState{} }).(*linkState)
	st.Focus()
	el.Calculate(400, 300)

	dl := &render.DrawList{}
	el.Paint(dl, text)

	ringFill := [4]float32{th.Surface.R, th.Surface.G, th.Surface.B, th.Surface.A}
	lastFill, firstGlyph := -1, -1
	for i, v := range dl.Vertices {
		glyph := v.UV[0] >= 0 && v.UV[1] >= 0
		if glyph {
			if firstGlyph < 0 {
				firstGlyph = i
			}
			continue
		}
		if v.Col == ringFill {
			lastFill = i
		}
	}
	if lastFill < 0 {
		t.Fatal("focused link should fill a focus ring")
	}
	if firstGlyph < 0 {
		t.Fatal("link should paint glyphs for its label")
	}
	if lastFill > firstGlyph {
		t.Fatalf("ring fill paints over the label: fill ends at vertex %d, glyphs start at %d", lastFill, firstGlyph)
	}
}
