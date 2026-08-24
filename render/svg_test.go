package render

import (
	"testing"
)

const testRectSVG = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 10 20">
  <rect width="10" height="20" fill="#ff0000"/>
</svg>`

const testTintSVG = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 8 8">
  <rect width="8" height="8" fill="currentColor"/>
</svg>`

func TestParseSVGDocSize(t *testing.T) {
	doc, err := ParseSVGDoc([]byte(testRectSVG), Color{})
	if err != nil {
		t.Fatal(err)
	}
	w, h := doc.Size()
	if absf(w-10) > 0.5 || absf(h-20) > 0.5 {
		t.Fatalf("css size: got %.2fx%.2f want 10x20", w, h)
	}
}

func TestRasterizeSVGRGBAColor(t *testing.T) {
	img, err := RasterizeSVGRGBA([]byte(testRectSVG), 10, 20, Color{})
	if err != nil {
		t.Fatal(err)
	}
	b := img.Bounds()
	if b.Dx() < 8 || b.Dy() < 16 {
		t.Fatalf("raster size: got %dx%d", b.Dx(), b.Dy())
	}
	r, g, bl, a := img.At(b.Min.X+b.Dx()/2, b.Min.Y+b.Dy()/2).RGBA()
	if a>>8 < 200 || r>>8 < 200 || g>>8 > 40 || bl>>8 > 40 {
		t.Fatalf("center pixel not red: r=%d g=%d b=%d a=%d", r>>8, g>>8, bl>>8, a>>8)
	}
}

func TestRasterizeSVGRGBACurrentColor(t *testing.T) {
	tint := Color{R: 0, G: 1, B: 0, A: 1}
	img, err := RasterizeSVGRGBA([]byte(testTintSVG), 8, 8, tint)
	if err != nil {
		t.Fatal(err)
	}
	b := img.Bounds()
	r, g, bl, a := img.At(b.Min.X+b.Dx()/2, b.Min.Y+b.Dy()/2).RGBA()
	if a>>8 < 200 || g>>8 < 200 || r>>8 > 40 || bl>>8 > 40 {
		t.Fatalf("center pixel not green: r=%d g=%d b=%d a=%d", r>>8, g>>8, bl>>8, a>>8)
	}
}

func TestRasterizeSVGRGBAInvalid(t *testing.T) {
	_, err := RasterizeSVGRGBA([]byte("not svg"), 16, 16, Color{})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestClampSVGRaster(t *testing.T) {
	w, h := clampSVGRaster(4000, 2000, 4000, 2000)
	if w > maxSVGRaster || h > maxSVGRaster {
		t.Fatalf("exceeds cap: %dx%d", w, h)
	}
	if w < 1 || h < 1 {
		t.Fatalf("zero size: %dx%d", w, h)
	}
}

func TestColorHex(t *testing.T) {
	if got := colorHex(Color{}); got != "#000000" {
		t.Fatalf("zero: %s", got)
	}
	if got := colorHex(Color{R: 1, A: 1}); got != "#ff0000" {
		t.Fatalf("red: %s", got)
	}
}

func absf(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}

func TestRasterizeSVGRGBANonEmptyAlpha(t *testing.T) {
	img, err := RasterizeSVGRGBA([]byte(testRectSVG), 16, 32, Color{})
	if err != nil {
		t.Fatal(err)
	}
	nz := 0
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			_, _, _, a := img.At(x, y).RGBA()
			if a > 0 {
				nz++
			}
		}
	}
	if nz == 0 {
		t.Fatal("empty raster")
	}
}
