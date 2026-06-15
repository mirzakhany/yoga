package shape

import "testing"

func TestLetterSpacingWidensLine(t *testing.T) {
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
	if len(base.Glyphs) != len(spaced.Glyphs) {
		t.Fatalf("glyph count changed: %d vs %d", len(base.Glyphs), len(spaced.Glyphs))
	}
	for i := range base.Glyphs {
		if spaced.Glyphs[i].Advance <= base.Glyphs[i].Advance {
			t.Fatalf("glyph %d advance not widened: base=%v spaced=%v", i,
				base.Glyphs[i].Advance, spaced.Glyphs[i].Advance)
		}
	}
}

func TestSpacedRoundTrip(t *testing.T) {
	fs, err := NewFontSystem(1, false)
	if err != nil {
		t.Fatal(err)
	}
	if err := fs.SetFont(FontConfig{Mono: FaceConfig{LetterSpacing: 3}}); err != nil {
		t.Fatal(err)
	}
	s := NewShaper(fs)
	ln := s.ShapeLineMono("abcdef")
	for _, off := range []int{0, 1, 3, 6} {
		x := ln.XForByte(off)
		got := ln.ByteForX(x)
		if got != off && off < 6 {
			t.Fatalf("off %d: x=%v round-trip=%d", off, x, got)
		}
	}
}

func TestLineHeightMultiplier(t *testing.T) {
	fs, err := NewFontSystem(1, false)
	if err != nil {
		t.Fatal(err)
	}
	base := fs.MonoMetrics().LineHeight
	if err := fs.SetFont(FontConfig{Mono: FaceConfig{LineHeight: 2.0}}); err != nil {
		t.Fatal(err)
	}
	tall := fs.MonoMetrics().LineHeight
	if tall <= base {
		t.Fatalf("line-height multiplier should increase line height: base=%v tall=%v", base, tall)
	}
}

func TestFontSizeWidensAndHeightens(t *testing.T) {
	fs, err := NewFontSystem(1, false)
	if err != nil {
		t.Fatal(err)
	}
	s := NewShaper(fs)
	baseW := s.ShapeLineMono("abcdef").Width
	baseH := fs.MonoMetrics().LineHeight

	if err := fs.SetFont(FontConfig{Mono: FaceConfig{Size: 28}}); err != nil {
		t.Fatal(err)
	}
	if w := s.ShapeLineMono("abcdef").Width; w <= baseW {
		t.Fatalf("larger size should widen text: base=%v big=%v", baseW, w)
	}
	if h := fs.MonoMetrics().LineHeight; h <= baseH {
		t.Fatalf("larger size should increase line height: base=%v big=%v", baseH, h)
	}
}

func TestSetFontBadFileLeavesStateIntact(t *testing.T) {
	fs, err := NewFontSystem(1, false)
	if err != nil {
		t.Fatal(err)
	}
	genBefore := fs.FontGen()
	monoBefore := fs.mono

	err = fs.SetFont(FontConfig{Mono: FaceConfig{File: "/no/such/font.ttf"}})
	if err == nil {
		t.Fatal("expected error for missing font file")
	}
	if fs.FontGen() != genBefore {
		t.Fatalf("font generation changed after failed SetFont: %d -> %d", genBefore, fs.FontGen())
	}
	if fs.mono != monoBefore {
		t.Fatal("mono face replaced despite failed SetFont")
	}
}

func TestTabWidthConfigurable(t *testing.T) {
	fs, err := NewFontSystem(1, false)
	if err != nil {
		t.Fatal(err)
	}
	s := NewShaper(fs)
	if err := fs.SetFont(FontConfig{TabWidth: 2}); err != nil {
		t.Fatal(err)
	}
	narrow := s.Width("\tx")
	if err := fs.SetFont(FontConfig{TabWidth: 8}); err != nil {
		t.Fatal(err)
	}
	wide := s.Width("\tx")
	if wide <= narrow {
		t.Fatalf("wider tab columns should advance further: narrow=%v wide=%v", narrow, wide)
	}
}
