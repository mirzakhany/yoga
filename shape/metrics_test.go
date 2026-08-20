package shape

import (
	"math"
	"testing"

	"github.com/mirzakhany/yoga/render"
)

func TestMeasureAtEmpty(t *testing.T) {
	eng, err := NewEngine(1, false)
	if err != nil {
		t.Fatal(err)
	}
	for _, size := range []float32{12, 14, 20} {
		w, h := eng.MeasureAt("", size)
		if w != 0 {
			t.Fatalf("size %v: empty width got %v want 0", size, w)
		}
		wantH := eng.Fonts.MetricsAt(size).LineHeight
		if h != wantH {
			t.Fatalf("size %v: empty height got %v want %v", size, h, wantH)
		}
	}
}

func TestMeasureAtMatchesLine(t *testing.T) {
	eng, err := NewEngine(1, false)
	if err != nil {
		t.Fatal(err)
	}
	const s = "Hello"
	for _, size := range []float32{12, 14, 20, 28} {
		w, h := eng.MeasureAt(s, size)
		ln := eng.LineAt(s, size)
		if w != ln.Width {
			t.Fatalf("size %v: MeasureAt width %v != LineAt.Width %v", size, w, ln.Width)
		}
		wantH := eng.Fonts.MetricsAt(size).LineHeight
		if h != wantH {
			t.Fatalf("size %v: MeasureAt height %v != MetricsAt %v", size, h, wantH)
		}
	}
}

func TestMeasureAtWidthIncreasesWithGlyphs(t *testing.T) {
	eng, err := NewEngine(1, false)
	if err != nil {
		t.Fatal(err)
	}
	wi, _ := eng.MeasureAt("i", 14)
	wii, _ := eng.MeasureAt("ii", 14)
	wiiii, _ := eng.MeasureAt("iiii", 14)
	if !(wi > 0 && wii > wi && wiiii > wii) {
		t.Fatalf("expected i < ii < iiii widths, got %v %v %v", wi, wii, wiiii)
	}
}

func TestMeasureAtNativeSizeScale(t *testing.T) {
	eng, err := NewEngine(1, false)
	if err != nil {
		t.Fatal(err)
	}
	w14, _ := eng.MeasureAt("Hello", 14)
	w20, _ := eng.MeasureAt("Hello", 20)
	w28, _ := eng.MeasureAt("Hello", 28)
	if w14 <= 0 {
		t.Fatal("14px width should be positive")
	}
	ratio := w20 / w14
	// Hinting means not exactly 20/14; accept a band around the linear scale.
	if ratio < 1.3 || ratio > 1.6 {
		t.Fatalf("20/14 width ratio %v outside [1.3, 1.6] (w14=%v w20=%v)", ratio, w14, w20)
	}
	if w28 <= w20 {
		t.Fatalf("28px should be wider than 20px: 20=%v 28=%v", w20, w28)
	}
}

func TestMetricsAtDefaultMatchesMetrics(t *testing.T) {
	eng, err := NewEngine(1, false)
	if err != nil {
		t.Fatal(err)
	}
	base := eng.Metrics()
	at14 := eng.Fonts.MetricsAt(14)
	if at14 != base {
		t.Fatalf("MetricsAt(14)=%+v Metrics()=%+v", at14, base)
	}
	at20 := eng.Fonts.MetricsAt(20)
	if at20.LineHeight <= at14.LineHeight {
		t.Fatalf("MetricsAt(20).LineHeight %v should exceed MetricsAt(14) %v", at20.LineHeight, at14.LineHeight)
	}
}

func TestByteForXRoundTripAtSizes(t *testing.T) {
	eng, err := NewEngine(1, false)
	if err != nil {
		t.Fatal(err)
	}
	for _, size := range []float32{12, 20} {
		ln := eng.LineAt("abcdef", size)
		for _, off := range []int{0, 1, 3, 6} {
			x := ln.XForByte(off)
			got := ln.ByteForX(x)
			if got != off && off < 6 {
				t.Fatalf("size %v off %d: x=%v round-trip=%d", size, off, x, got)
			}
		}
	}
}

func TestDrawStringTopAtWidthMatchesMeasure(t *testing.T) {
	eng, err := NewEngine(1, false)
	if err != nil {
		t.Fatal(err)
	}
	const s = "Hello"
	for _, size := range []float32{14, 20} {
		mw, _ := eng.MeasureAt(s, size)
		dl := &render.DrawList{}
		gotW := eng.DrawStringTopAt(dl, s, 10, 20, render.RGBA8(255, 255, 255, 255), size)
		if math.Abs(float64(gotW-mw)) > 0.01 {
			t.Fatalf("size %v: draw width %v != measure %v", size, gotW, mw)
		}
		ln := eng.LineAt(s, size)
		if len(ln.Glyphs) == 0 {
			t.Fatal("expected glyphs")
		}
		last := ln.Glyphs[len(ln.Glyphs)-1]
		layoutEnd := last.X + last.Advance
		if math.Abs(float64(layoutEnd-mw)) > 0.5 {
			t.Fatalf("size %v: last glyph X+Advance %v != measure width %v", size, layoutEnd, mw)
		}
		// Drawn quads use baked atlas sizes at the same ppem (no stretch).
		if len(dl.Vertices) < 4 {
			t.Fatalf("size %v: expected glyph quads in draw list", size)
		}
	}
}
