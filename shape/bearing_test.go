package shape

import (
	"math"
	"testing"

	"github.com/mirzakhany/yoga/render"
)

func TestUIGlyphPenAtOriginAndBearingForInk(t *testing.T) {
	eng, err := NewEngine(1, false)
	if err != nil {
		t.Fatal(err)
	}
	ln := eng.LineAt("To", 14)
	if len(ln.Glyphs) < 2 {
		t.Fatalf("expected >=2 glyphs, got %d", len(ln.Glyphs))
	}
	first := ln.Glyphs[0]
	if math.Abs(float64(first.X)) > 0.01 {
		t.Fatalf("first glyph pen X should be ~0, got %v", first.X)
	}
	// Inter's 'T' has a non-trivial left side bearing; ink is not at the pen.
	if first.BearingX == 0 {
		t.Fatal("expected non-zero BearingX for Inter 'T'")
	}

	dl := &render.DrawList{}
	eng.DrawStringTopAt(dl, "To", 0, 0, render.RGBA8(255, 255, 255, 255), 14)
	if len(dl.Vertices) < 4 {
		t.Fatal("expected drawn glyph quads")
	}
	face := eng.Fonts.Face(first.FaceID)
	ppem := eng.glyphPpem(ln)
	entry := eng.Atlas.EnsureGlyph(first.FaceID, face, first.GID, ppem)
	wantInk := first.X + first.BearingX - entry.Pad
	gotQuadX := dl.Vertices[0].Pos[0]
	if math.Abs(float64(gotQuadX-wantInk)) > 0.05 {
		t.Fatalf("drawn quad X=%v want ink-left %v (bearing=%v pad=%v)",
			gotQuadX, wantInk, first.BearingX, entry.Pad)
	}
	if entry.Pad <= 0 {
		t.Fatal("outline glyphs should record atlas Pad")
	}
}

func TestUIBearingDiffersAcrossLetters(t *testing.T) {
	eng, err := NewEngine(1, false)
	if err != nil {
		t.Fatal(err)
	}
	to := eng.LineAt("To", 14)
	nn := eng.LineAt("nn", 14)
	if len(to.Glyphs) < 2 || len(nn.Glyphs) < 2 {
		t.Fatal("expected shaped glyphs")
	}
	// Different left side bearings → different ink offsets relative to the pen.
	if to.Glyphs[0].BearingX == nn.Glyphs[0].BearingX {
		t.Fatalf("expected distinct BearingX for 'T' vs 'n': both %v", to.Glyphs[0].BearingX)
	}
	// Pen advances still drive layout width (caret/selection), independent of ink.
	if to.Glyphs[1].X <= to.Glyphs[0].X {
		t.Fatal("second glyph pen should advance past the first")
	}
}

func TestMonoLetterSpacingStillWidens(t *testing.T) {
	fs, err := NewFontSystem(1, false)
	if err != nil {
		t.Fatal(err)
	}
	s := NewShaper(fs)
	base := s.ShapeLineMono("abcdef")
	if err := fs.SetFont(FontConfig{Mono: FaceConfig{LetterSpacing: 4}}); err != nil {
		t.Fatal(err)
	}
	spaced := s.ShapeLineMono("abcdef")
	if spaced.Width <= base.Width {
		t.Fatalf("letter spacing should widen line: base=%v spaced=%v", base.Width, spaced.Width)
	}
}

func TestDrawInkUsesBearingNotPen(t *testing.T) {
	eng, err := NewEngine(1, false)
	if err != nil {
		t.Fatal(err)
	}
	const originX float32 = 40
	dl := &render.DrawList{}
	eng.DrawStringTopAt(dl, "A", originX, 10, render.RGBA8(255, 255, 255, 255), 14)
	ln := eng.LineAt("A", 14)
	if len(ln.Glyphs) == 0 || len(dl.Vertices) == 0 {
		t.Fatal("expected glyph and vertices")
	}
	g := ln.Glyphs[0]
	face := eng.Fonts.Face(g.FaceID)
	entry := eng.Atlas.EnsureGlyph(g.FaceID, face, g.GID, eng.glyphPpem(ln))
	want := originX + g.X + g.BearingX - entry.Pad
	if math.Abs(float64(dl.Vertices[0].Pos[0]-want)) > 0.05 {
		t.Fatalf("quad X=%v want %v (pen=%v bearing=%v pad=%v)",
			dl.Vertices[0].Pos[0], want, g.X, g.BearingX, entry.Pad)
	}
	// Quad must not sit at the bare pen (unless bearing and pad cancel).
	barePen := originX + g.X
	if math.Abs(float64(g.BearingX-entry.Pad)) > 0.05 &&
		math.Abs(float64(dl.Vertices[0].Pos[0]-barePen)) < 0.05 {
		t.Fatalf("quad landed on pen without bearing/pad correction")
	}
}
