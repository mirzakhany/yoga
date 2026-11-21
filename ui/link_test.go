package ui

import (
	"testing"

	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
	"github.com/mirzakhany/yoga/theme"
)

// A focused link strokes a FocusRing ring around its label and paints no fill
// that could cover the text.
func TestFocusedLinkPaintsRingAroundLabel(t *testing.T) {
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

	ring := [4]float32{th.FocusRing.R, th.FocusRing.G, th.FocusRing.B, th.FocusRing.A}
	fill := [4]float32{th.Surface.R, th.Surface.G, th.Surface.B, th.Surface.A}
	sawRing, sawGlyph := false, false
	for _, v := range dl.Vertices {
		if v.UV[0] >= 0 && v.UV[1] >= 0 {
			sawGlyph = true
			continue
		}
		switch v.Col {
		case ring:
			sawRing = true
		case fill:
			// The ring is a stroke outside the label; it must not paint a
			// fill that could cover the text.
			t.Fatal("focused link painted a fill under its ring")
		}
	}
	if !sawRing {
		t.Fatal("focused link should stroke a FocusRing ring")
	}
	if !sawGlyph {
		t.Fatal("link should paint glyphs for its label")
	}
}
