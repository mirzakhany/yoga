package ui

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
)

func encodePNG(w, h int, c color.RGBA) []byte {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetRGBA(x, y, c)
		}
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

func encodeJPEG(w, h int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetRGBA(x, y, color.RGBA{G: 200, A: 255})
		}
	}
	var buf bytes.Buffer
	_ = jpeg.Encode(&buf, img, nil)
	return buf.Bytes()
}

func setupImageTest(t *testing.T) *Ctx {
	t.Helper()
	text, err := shape.NewEngine(1, false)
	if err != nil {
		t.Fatal(err)
	}
	sheet := render.NewSpriteSheet(text.Atlas)
	c := New(text, NewFocusScope(), nil)
	c.SetIcons(sheet)
	SetFrameResources(text, sheet, nil)
	return c
}

func layoutImageForTest(c *Ctx, n *Node) *layout.Element {
	c.BeginFrame(400, 300, nil, nil)
	el := n.Layout(c)
	root := layout.New(layout.Box(), el)
	root.Calculate(400, 300)
	return el
}

func TestImageIntrinsicSize(t *testing.T) {
	c := setupImageTest(t)
	pngData := encodePNG(120, 80, color.RGBA{R: 255, A: 255})
	n := Image("img-intrinsic", pngData)
	el := layoutImageForTest(c, n)
	if el == nil {
		t.Fatal("nil element")
	}
	w, h := el.LayoutSize()
	if w != 120 || h != 80 {
		t.Fatalf("intrinsic size: got %.0fx%.0f want 120x80", w, h)
	}
}

func TestImageWidthPreservesAspect(t *testing.T) {
	c := setupImageTest(t)
	pngData := encodePNG(100, 50, color.RGBA{B: 255, A: 255})
	n := Image("img-aspect", pngData).Width(200)
	el := layoutImageForTest(c, n)
	w, h := el.LayoutSize()
	if w != 200 || h != 100 {
		t.Fatalf("aspect size: got %.0fx%.0f want 200x100", w, h)
	}
}

func TestImageJPEGDecode(t *testing.T) {
	c := setupImageTest(t)
	jpg := encodeJPEG(64, 48)
	n := Image("img-jpeg", jpg)
	el := layoutImageForTest(c, n)
	w, h := el.LayoutSize()
	if w != 64 || h != 48 {
		t.Fatalf("jpeg size: got %.0fx%.0f want 64x48", w, h)
	}
}

func TestImageFS(t *testing.T) {
	c := setupImageTest(t)
	data := encodePNG(24, 24, color.RGBA{A: 255})
	fsys := fstest.MapFS{"logo.png": {Data: data}}
	n := ImageFS("img-fs", fsys, "logo.png")
	el := layoutImageForTest(c, n)
	w, h := el.LayoutSize()
	if w != 24 || h != 24 {
		t.Fatalf("fs size: got %.0fx%.0f want 24x24", w, h)
	}
	// Second frame should not re-read (still works).
	c.BeginFrame(400, 300, nil, nil)
	el2 := n.Layout(c)
	root2 := layout.New(layout.Box(), el2)
	root2.Calculate(400, 300)
	w2, h2 := el2.LayoutSize()
	if w2 != w || h2 != h {
		t.Fatalf("second layout changed size: %.0fx%.0f", w2, h2)
	}
}

func TestImageFile(t *testing.T) {
	c := setupImageTest(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "photo.png")
	if err := os.WriteFile(path, encodePNG(40, 30, color.RGBA{R: 128, A: 255}), 0o644); err != nil {
		t.Fatal(err)
	}
	n := ImageFile("img-file", path)
	el := layoutImageForTest(c, n)
	w, h := el.LayoutSize()
	if w != 40 || h != 30 {
		t.Fatalf("file size: got %.0fx%.0f want 40x30", w, h)
	}
}

func TestImageInvalidBytesNoPanic(t *testing.T) {
	c := setupImageTest(t)
	n := Image("img-bad", []byte("not-an-image"))
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

func TestImageDestRectContain(t *testing.T) {
	frame := render.Rect{X: 0, Y: 0, W: 200, H: 100}
	dst := imageDestRect(frame, 100, 100, FitContain)
	if dst.W != 100 || dst.H != 100 {
		t.Fatalf("contain: got %.0fx%.0f want 100x100", dst.W, dst.H)
	}
	if dst.X != 50 || dst.Y != 0 {
		t.Fatalf("contain center: got x=%v y=%v", dst.X, dst.Y)
	}
}

func TestImageDestRectCover(t *testing.T) {
	frame := render.Rect{X: 0, Y: 0, W: 200, H: 100}
	dst := imageDestRect(frame, 100, 100, FitCover)
	if dst.W != 200 || dst.H != 200 {
		t.Fatalf("cover: got %.0fx%.0f want 200x200", dst.W, dst.H)
	}
}

func TestImageDestRectFill(t *testing.T) {
	frame := render.Rect{X: 10, Y: 20, W: 200, H: 100}
	dst := imageDestRect(frame, 50, 50, FitFill)
	if dst != frame {
		t.Fatalf("fill: got %+v want %+v", dst, frame)
	}
}
