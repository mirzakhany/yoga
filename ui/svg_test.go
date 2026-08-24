package ui

import (
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
)

const testSVG = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 40 20">
  <rect width="40" height="20" fill="#3366ff"/>
</svg>`

const testTintSVG = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16 16">
  <rect width="16" height="16" fill="currentColor"/>
</svg>`

func TestSVGIntrinsicSize(t *testing.T) {
	c := setupImageTest(t)
	n := SVG("svg-intrinsic", []byte(testSVG))
	el := layoutImageForTest(c, n)
	w, h := el.LayoutSize()
	if w != 40 || h != 20 {
		t.Fatalf("intrinsic size: got %.0fx%.0f want 40x20", w, h)
	}
}

func TestSVGWidthPreservesAspect(t *testing.T) {
	c := setupImageTest(t)
	n := SVG("svg-aspect", []byte(testSVG)).Width(80)
	el := layoutImageForTest(c, n)
	w, h := el.LayoutSize()
	if w != 80 || h != 40 {
		t.Fatalf("aspect size: got %.0fx%.0f want 80x40", w, h)
	}
}

func TestSVGFS(t *testing.T) {
	c := setupImageTest(t)
	fsys := fstest.MapFS{"logo.svg": {Data: []byte(testSVG)}}
	n := SVGFS("svg-fs", fsys, "logo.svg")
	el := layoutImageForTest(c, n)
	w, h := el.LayoutSize()
	if w != 40 || h != 20 {
		t.Fatalf("fs size: got %.0fx%.0f want 40x20", w, h)
	}
	c.BeginFrame(400, 300, nil, nil)
	el2 := n.Layout(c)
	root2 := layout.New(layout.Box(), el2)
	root2.Calculate(400, 300)
	w2, h2 := el2.LayoutSize()
	if w2 != w || h2 != h {
		t.Fatalf("second layout changed size: %.0fx%.0f", w2, h2)
	}
}

func TestSVGFile(t *testing.T) {
	c := setupImageTest(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "mark.svg")
	if err := os.WriteFile(path, []byte(testSVG), 0o644); err != nil {
		t.Fatal(err)
	}
	n := SVGFile("svg-file", path)
	el := layoutImageForTest(c, n)
	w, h := el.LayoutSize()
	if w != 40 || h != 20 {
		t.Fatalf("file size: got %.0fx%.0f want 40x20", w, h)
	}
}

func TestSVGInvalidBytesNoPanic(t *testing.T) {
	c := setupImageTest(t)
	n := SVG("svg-bad", []byte("not-an-svg"))
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("panicked: %v", r)
		}
	}()
	el := layoutImageForTest(c, n)
	if el == nil {
		t.Fatal("nil element")
	}
	w, h := el.LayoutSize()
	if w <= 0 || h <= 0 {
		t.Fatalf("expected fallback size, got %.0fx%.0f", w, h)
	}
}

func TestSVGCurrentColor(t *testing.T) {
	c := setupImageTest(t)
	n := SVG("svg-tint", []byte(testTintSVG)).
		Width(16).
		Style(Spec{}.TextColorLit(render.Color{G: 1, A: 1}))
	el := layoutImageForTest(c, n)
	w, h := el.LayoutSize()
	if w != 16 || h != 16 {
		t.Fatalf("size: got %.0fx%.0f want 16x16", w, h)
	}
	if el.Paint == nil {
		t.Fatal("missing paint")
	}
}

func TestSVGFit(t *testing.T) {
	c := setupImageTest(t)
	n := SVG("svg-fit", []byte(testSVG)).Frame(80, 80).Fit(FitContain)
	el := layoutImageForTest(c, n)
	w, h := el.LayoutSize()
	if w != 80 || h != 80 {
		t.Fatalf("frame: got %.0fx%.0f want 80x80", w, h)
	}
}
