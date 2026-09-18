package shape

import (
	"math"
	"testing"

	"github.com/mirzakhany/yoga/render"
)

func TestEngineSetScaleRebakesGlyphs(t *testing.T) {
	eng, err := NewEngine(2, false)
	if err != nil {
		t.Fatal(err)
	}
	draw := func() *render.DrawList {
		dl := &render.DrawList{}
		eng.DrawStringTopAt(dl, "Hello", 10.3, 5.7, render.RGBA8(255, 255, 255, 255), 14)
		return dl
	}
	hi := draw()
	hiH := hi.Vertices[2].Pos[1] - hi.Vertices[0].Pos[1]
	width2x := eng.LineAt("Hello", 14).Width

	if !eng.SetScale(1) {
		t.Fatal("SetScale(1) should report a change")
	}
	if eng.SetScale(1) {
		t.Fatal("repeated SetScale should be a no-op")
	}
	if eng.Atlas.Scale() != 1 || eng.Fonts.Scale() != 1 {
		t.Fatalf("scale not applied: atlas %v fonts %v", eng.Atlas.Scale(), eng.Fonts.Scale())
	}
	lo := draw()
	for i, v := range lo.Vertices {
		for _, p := range v.Pos {
			if p != float32(math.Round(float64(p))) {
				t.Fatalf("vertex %d at %v not on a 1x device pixel", i, v.Pos)
			}
		}
	}
	// A 2x bake has half-pixel padding in logical px; a 1x bake is whole-pixel
	// and so its quads differ in height.
	loH := lo.Vertices[2].Pos[1] - lo.Vertices[0].Pos[1]
	if loH == hiH {
		t.Fatalf("glyph quad height unchanged after scale switch (%v); glyphs not re-baked", loH)
	}
	if w := eng.LineAt("Hello", 14).Width; math.Abs(float64(w-width2x)) > 2 {
		t.Fatalf("logical width drifted across scales: 2x %v, 1x %v", width2x, w)
	}
	// Switching back restores the original logical font size exactly.
	eng.SetScale(2)
	if got := eng.Fonts.uiPixelSize.Round(); got != 28 {
		t.Fatalf("ui ppem at 2x = %d, want 28", got)
	}
}
