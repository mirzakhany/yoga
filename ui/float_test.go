package ui

import (
	"testing"

	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/layout"
)

// TestFloatPinsToCorner checks that a floating node sits at its corner of the
// parent's content box and takes no room from its siblings.
func TestFloatPinsToCorner(t *testing.T) {
	c := newTestCtx(t)
	cases := []struct {
		corner     Corner
		wantRight  bool
		wantBottom bool
	}{
		{CornerTopLeft, false, false},
		{CornerTopRight, true, false},
		{CornerBottomLeft, false, true},
		{CornerBottomRight, true, true},
	}
	for _, tc := range cases {
		var fill, float *layout.Element
		body := func(cc *Ctx) View {
			fill = Column().Grow(1).Layout(cc)
			float = Text("hi").Width(40).Height(20).Float(tc.corner, 10, 6).Layout(cc)
			return Column(Raw(fill), Raw(float)).Padding(4).Grow(1)
		}
		BuildFrame(c, body, 400, 300, &input.Mouse{}, &input.Keyboard{})
		c.EndFrame()

		if fill.Frame.H < 290 {
			t.Errorf("corner %d: sibling lost room to the float: h=%v", tc.corner, fill.Frame.H)
		}
		f := float.Frame
		wantX := float32(4 + 10)
		if tc.wantRight {
			wantX = 400 - 4 - 10 - 40
		}
		wantY := float32(4 + 6)
		if tc.wantBottom {
			wantY = 300 - 4 - 6 - 20
		}
		if f.X != wantX || f.Y != wantY || f.W != 40 || f.H != 20 {
			t.Errorf("corner %d: frame %+v, want x=%v y=%v 40x20", tc.corner, f, wantX, wantY)
		}
	}
}
