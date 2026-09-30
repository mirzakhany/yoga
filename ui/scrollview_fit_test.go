package ui

import (
	"strings"
	"testing"

	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/render"
)

// fitScrollFrames lays out a header over a FitContent scroll of text kept at
// the bottom, a few frames so the scroll learns its content height, and
// returns the scroll's frame.
func fitScrollFrames(t *testing.T, text string) render.Rect {
	t.Helper()
	c := newTestCtx(t)
	for range 4 {
		BuildFrame(c, func(*Ctx) View {
			return Column(
				Text("Steps"),
				Column(Scroll("s", Paragraph(text)).FitContent()).Grow(1).Justify(JustifyEnd),
			).Grow(1)
		}, 400, 300, &input.Mouse{}, &input.Keyboard{})
		c.EndFrame()
	}
	return c.Widget("s", nil).(*ScrollView).host.Frame
}

func TestScrollFitContentSitsAtBottom(t *testing.T) {
	fr := fitScrollFrames(t, "one\ntwo")
	c := newTestCtx(t)
	_, lineH := c.Text().MeasureAt("Ag", c.Theme().Typography.Body.Size)
	if fr.H != 2*lineH {
		t.Fatalf("height %v, want two lines (%v)", fr.H, 2*lineH)
	}
	if fr.Y+fr.H != 300 {
		t.Fatalf("bottom at %v, want 300", fr.Y+fr.H)
	}
}

func TestScrollFitContentShrinksToRoom(t *testing.T) {
	fr := fitScrollFrames(t, strings.Repeat("line\n", 100))
	if fr.Y+fr.H != 300 || fr.Y <= 0 || fr.H >= 300 {
		t.Fatalf("frame %+v, want it to fill the room below the header and end at 300", fr)
	}
}
