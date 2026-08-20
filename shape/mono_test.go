package shape

import "testing"

// TestMonoShapingUsesMonoFace guards against the editor mono path falling back
// to the proportional UI face. The default font query ranks the UI face ahead
// of mono, so ASCII must be explicitly resolved to the mono face; otherwise
// glyph advances vary and monospace alignment (tab stops, gofmt column padding)
// breaks. See ShapeLineFace / monoFontmap.
func TestMonoShapingUsesMonoFace(t *testing.T) {
	fs, err := NewFontSystem(1, false)
	if err != nil {
		t.Fatal(err)
	}
	s := NewShaper(fs)
	monoID := fs.FaceID(fs.mono)

	ln := s.ShapeLineMono("mWi0 _.{}(),")
	if len(ln.Glyphs) == 0 {
		t.Fatal("no glyphs shaped")
	}
	want := ln.Glyphs[0].Advance
	for i, g := range ln.Glyphs {
		if g.FaceID != monoID {
			t.Fatalf("glyph %d (byte %d) shaped with face %d, want mono face %d",
				i, g.ClusterByte, g.FaceID, monoID)
		}
		if g.Advance != want {
			t.Fatalf("glyph %d advance=%v, want uniform monospace advance %v",
				i, g.Advance, want)
		}
	}
}

// TestMonoNoLigatures guards against ligatures / contextual alternates in the
// editor mono path. Fonts like JetBrains Mono substitute connected glyph forms
// for "//", "==", ":=" etc., which visually merges characters and looks like
// stray spacing in code. With features disabled, repeated operator characters
// must map to the same plain glyph and stay one cell per character.
func TestMonoNoLigatures(t *testing.T) {
	fs, err := NewFontSystem(1, false)
	if err != nil {
		t.Fatal(err)
	}
	s := NewShaper(fs)

	cases := []struct {
		text string
		i, j int // glyph indices expected to share a glyph id
	}{
		{"//", 0, 1},     // both slashes
		{"a == b", 2, 3}, // both '='
	}
	for _, c := range cases {
		ln := s.ShapeLineMono(c.text)
		if len(ln.Glyphs) != len([]rune(c.text)) {
			t.Fatalf("%q: %d glyphs for %d runes (ligature merged clusters?)",
				c.text, len(ln.Glyphs), len([]rune(c.text)))
		}
		if a, b := ln.Glyphs[c.i].GID, ln.Glyphs[c.j].GID; a != b {
			t.Fatalf("%q: glyphs %d,%d differ (gid %d vs %d) — contextual alternate applied",
				c.text, c.i, c.j, a, b)
		}
	}
}

// TestMonoTabStopsAlign verifies that leading-tab indentation and gofmt-style
// space padding land on consistent monospace columns, so struct field types
// in differently-named rows align at the same x.
func TestMonoTabStopsAlign(t *testing.T) {
	fs, err := NewFontSystem(1, false)
	if err != nil {
		t.Fatal(err)
	}
	s := NewShaper(fs)
	cw := s.cellWidthAt(true, 0)

	// gofmt output: leading tab indent, then spaces aligning the type column.
	a := s.ShapeLineMono("\tStartByte, OldEndByte, NewEndByte int")
	b := s.ShapeLineMono("\tStart, OldEnd, NewEnd             Pt")

	// Type token starts: "int" is the last 3 cells of a, "Pt" the last 2 of b.
	intX := a.XForByte(len(string(a.Runes)) - 3)
	ptX := b.XForByte(len(string(b.Runes)) - 2)
	if diff := intX - ptX; diff > 0.5 || diff < -0.5 {
		t.Fatalf("type column misaligned: int@%v pt@%v (diff %v, cell %v)", intX, ptX, diff, cw)
	}
}
