package render

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"
)

func makeTestPNG(w, h int, c color.RGBA) []byte {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetRGBA(x, y, c)
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		panic(err)
	}
	return buf.Bytes()
}

func TestEnsureImageCacheHit(t *testing.T) {
	a := NewAtlasScale(1)
	data := makeTestPNG(16, 16, color.RGBA{R: 255, A: 255})
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	rgba := image.NewRGBA(image.Rect(0, 0, 16, 16))
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			rgba.Set(x, y, img.At(x, y))
		}
	}

	e1, ok := a.EnsureImage("test", rgba)
	if !ok {
		t.Fatal("EnsureImage failed")
	}
	e2, ok := a.EnsureImage("test", rgba)
	if !ok {
		t.Fatal("EnsureImage cache miss")
	}
	if e1.UV != e2.UV || e1.W != e2.W || e1.H != e2.H {
		t.Fatalf("cache entry changed: %+v vs %+v", e1, e2)
	}
}

func TestEnsureImageDownscalesWide(t *testing.T) {
	a := NewAtlasScale(1)
	rgba := image.NewRGBA(image.Rect(0, 0, 800, 100))
	e, ok := a.EnsureImage("wide", rgba)
	if !ok {
		t.Fatal("EnsureImage failed")
	}
	maxW := a.colorW - 2*a.colorShelf.pad
	if e.physW > maxW {
		t.Fatalf("expected downscale to <= %d, got physW=%d", maxW, e.physW)
	}
}

func TestGrowColorPreservesUVs(t *testing.T) {
	a := NewAtlasScale(1)
	r1 := image.NewRGBA(image.Rect(0, 0, 32, 32))
	r2 := image.NewRGBA(image.Rect(0, 0, 32, 32))
	e1, ok := a.EnsureImage("a", r1)
	if !ok {
		t.Fatal("first pack failed")
	}
	// Force grow by packing a tall image that won't fit remaining shelf space.
	tall := image.NewRGBA(image.Rect(0, 0, 32, a.colorH))
	_, ok = a.EnsureImage("tall", tall)
	if !ok {
		t.Fatal("tall pack failed")
	}
	e1After, ok := a.ImageUV("a")
	if !ok {
		t.Fatal("lost first image after grow")
	}
	if e1After.physX != e1.physX || e1After.physY != e1.physY {
		t.Fatalf("phys coords changed: %+v vs %+v", e1, e1After)
	}
	if e1After.UV.W <= 0 || e1After.UV.H <= 0 {
		t.Fatalf("bad UV after grow: %+v", e1After.UV)
	}
	// Second small image still packable.
	_, ok = a.EnsureImage("b", r2)
	if !ok {
		t.Fatal("second pack after grow failed")
	}
}

func TestGrowColorRemapsDrawListUVs(t *testing.T) {
	a := NewAtlasScale(1)
	rgba := image.NewRGBA(image.Rect(0, 0, 32, 32))
	e, ok := a.EnsureImage("dot", rgba)
	if !ok {
		t.Fatal("pack failed")
	}
	var dl DrawList
	a.BindDrawList(&dl)
	defer a.BindDrawList(nil)

	dl.AddGlyphQuad(Rect{X: 0, Y: 0, W: 32, H: 32}, e.UV, PageColor, Color{R: 1, G: 1, B: 1, A: 1})
	oldH := a.colorH
	vBefore := dl.Vertices[0].UV[1]

	a.growColor(a.colorH * 2)
	scale := float32(oldH) / float32(a.colorH)
	wantV := vBefore * scale
	if dl.Vertices[0].UV[1] != wantV {
		t.Fatalf("vertex V not remapped: got %v want %v", dl.Vertices[0].UV[1], wantV)
	}
	eAfter, ok := a.ImageUV("dot")
	if !ok {
		t.Fatal("image missing after grow")
	}
	if eAfter.physX != e.physX || eAfter.physY != e.physY {
		t.Fatalf("phys coords changed: %+v vs %+v", e, eAfter)
	}
	if dl.Vertices[0].UV[0] != eAfter.UV.X || dl.Vertices[0].UV[1] != eAfter.UV.Y {
		t.Fatalf("vert UV %+v != map UV top-left (%v,%v)", dl.Vertices[0].UV, eAfter.UV.X, eAfter.UV.Y)
	}
}

func TestSpriteSheetDrawImageEntry(t *testing.T) {
	a := NewAtlasScale(1)
	sheet := NewSpriteSheet(a)
	rgba := image.NewRGBA(image.Rect(0, 0, 8, 8))
	if _, ok := a.EnsureImage("dot", rgba); !ok {
		t.Fatal("pack failed")
	}
	var dl DrawList
	dst := Rect{X: 0, Y: 0, W: 8, H: 8}
	if !sheet.DrawImageEntry(&dl, "dot", dst) {
		t.Fatal("DrawImageEntry failed")
	}
	if len(dl.Vertices) != 4 || len(dl.Indices) != 6 {
		t.Fatalf("expected one quad, got verts=%d indices=%d", len(dl.Vertices), len(dl.Indices))
	}
}
