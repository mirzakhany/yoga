package shape

import "testing"

func TestShapeLineASCIIWidth(t *testing.T) {
	fs, err := NewFontSystem(1, false)
	if err != nil {
		t.Fatal(err)
	}
	s := NewShaper(fs)
	w := s.Width("hello")
	if w <= 0 {
		t.Fatalf("expected positive width, got %v", w)
	}
}

func TestByteForXRoundTrip(t *testing.T) {
	fs, err := NewFontSystem(1, false)
	if err != nil {
		t.Fatal(err)
	}
	s := NewShaper(fs)
	ln := s.ShapeLine("abcdef")
	for _, off := range []int{0, 1, 3, 6} {
		x := ln.XForByte(off)
		got := ln.ByteForX(x)
		if got != off && off < 6 {
			t.Fatalf("off %d: x=%v round-trip=%d", off, x, got)
		}
	}
}

func TestShapeLineLeadingTab(t *testing.T) {
	fs, err := NewFontSystem(1, false)
	if err != nil {
		t.Fatal(err)
	}
	s := NewShaper(fs)
	plain := s.Width("foo")
	tabbed := s.Width("\tfoo")
	if tabbed <= plain {
		t.Fatalf("leading tab should widen line: plain=%v tabbed=%v", plain, tabbed)
	}
	doubleTab := s.Width("\t\tfoo")
	if doubleTab <= tabbed {
		t.Fatalf("two leading tabs should widen further: one=%v two=%v", tabbed, doubleTab)
	}
}

func TestLineCache(t *testing.T) {
	fs, err := NewFontSystem(1, false)
	if err != nil {
		t.Fatal(err)
	}
	sh := NewShaper(fs)
	c := NewLineCache(sh)
	a := c.Get("same")
	b := c.Get("same")
	if len(a.Glyphs) != len(b.Glyphs) {
		t.Fatal("cache should return equivalent lines")
	}
}
