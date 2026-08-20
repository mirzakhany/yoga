package shape

import (
	"testing"

	"github.com/mirzakhany/yoga/render"
)

func TestSemiBoldUsesDifferentFace(t *testing.T) {
	eng, err := NewEngine(1, false)
	if err != nil {
		t.Fatal(err)
	}
	const s = "Hello"
	reg := eng.LineAtWeight(s, 14, WeightRegular)
	strong := eng.LineAtWeight(s, 14, WeightSemiBold)
	if len(reg.Glyphs) == 0 || len(strong.Glyphs) == 0 {
		t.Fatal("expected glyphs")
	}
	if reg.Glyphs[0].FaceID == strong.Glyphs[0].FaceID {
		t.Fatalf("expected different face IDs for Regular vs SemiBold: both %d", reg.Glyphs[0].FaceID)
	}
	if eng.Fonts.Face(reg.Glyphs[0].FaceID) != eng.Fonts.Primary() {
		t.Fatal("regular glyphs should use Primary")
	}
	if eng.Fonts.Face(strong.Glyphs[0].FaceID) != eng.Fonts.PrimaryStrong() {
		t.Fatal("semibold glyphs should use PrimaryStrong")
	}
}

func TestSemiBoldMeasureWidthDiffers(t *testing.T) {
	eng, err := NewEngine(1, false)
	if err != nil {
		t.Fatal(err)
	}
	const s = "Hello"
	w400, _ := eng.MeasureAtWeight(s, 14, WeightRegular)
	w600, _ := eng.MeasureAtWeight(s, 14, WeightSemiBold)
	if w400 <= 0 || w600 <= 0 {
		t.Fatalf("expected positive widths: 400=%v 600=%v", w400, w600)
	}
	if w400 == w600 {
		t.Fatalf("expected SemiBold width to differ from Regular: both %v", w400)
	}
}

func TestMeasureAtDefaultsToRegular(t *testing.T) {
	eng, err := NewEngine(1, false)
	if err != nil {
		t.Fatal(err)
	}
	const s = "Yoga"
	a, _ := eng.MeasureAt(s, 14)
	b, _ := eng.MeasureAtWeight(s, 14, WeightRegular)
	if a != b {
		t.Fatalf("MeasureAt should match WeightRegular: %v vs %v", a, b)
	}
}

func TestDrawStringTopAtWeightUsesStrongFace(t *testing.T) {
	eng, err := NewEngine(1, false)
	if err != nil {
		t.Fatal(err)
	}
	dl := &render.DrawList{}
	eng.DrawStringTopAtWeight(dl, "Hi", 0, 0, render.RGBA8(255, 255, 255, 255), 14, WeightSemiBold)
	if len(dl.Vertices) < 4 {
		t.Fatal("expected drawn quads")
	}
	ln := eng.LineAtWeight("Hi", 14, WeightSemiBold)
	if ln.Weight < WeightSemiBold {
		t.Fatalf("line weight: got %d want >= %d", ln.Weight, WeightSemiBold)
	}
}
