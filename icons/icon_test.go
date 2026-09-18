package icons

import "testing"

func TestSearchIcon(t *testing.T) {
	if Search.Empty() {
		t.Fatal("Search icon missing")
	}
	if _, err := Search.Alpha(BakePx); err != nil {
		t.Fatal(err)
	}
}

func TestYogaIcon(t *testing.T) {
	if Yoga.Empty() {
		t.Fatal("Yoga icon missing")
	}
}

func TestScaleAlphaDownscaleAverages(t *testing.T) {
	// A 1px line in a 4x4 mask halves to 50% coverage, not 0 or 100%.
	src := make([]byte, 16)
	for x := 0; x < 4; x++ {
		src[1*4+x] = 255
	}
	got := scaleAlpha(src, 4, 4, 2, 2)
	if got.Pix[0] != 128 || got.Pix[1] != 128 || got.Pix[2] != 0 || got.Pix[3] != 0 {
		t.Fatalf("area average = %v, want [128 128 0 0]", got.Pix)
	}
}
