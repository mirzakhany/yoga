package render

import (
	"testing"

	"github.com/mirzakhany/yoga/icons"
)

func TestSpriteSheetDrawIconPixelAligned(t *testing.T) {
	for _, scale := range []float32{1, 2} {
		a := NewAtlasScale(scale)
		s := NewSpriteSheet(a)
		dl := &DrawList{}
		if !s.Draw(dl, icons.Search, Rect{X: 10.3, Y: 4.6, W: 16, H: 16}, Color{A: 1}) {
			t.Fatalf("scale %v: draw failed", scale)
		}
		v0, v2 := dl.Vertices[0].Pos, dl.Vertices[2].Pos
		for _, p := range []float32{v0[0], v0[1], v2[0], v2[1]} {
			if d := p * scale; d != float32(int(d)) {
				t.Fatalf("scale %v: corner %v not on a device pixel", scale, p)
			}
		}
		if w := v2[0] - v0[0]; w != 16 {
			t.Fatalf("scale %v: icon width %v, want 16", scale, w)
		}
	}
}
