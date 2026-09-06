package render

import (
	"os"
	"testing"

	"github.com/mirzakhany/yoga/icons"
)

func TestRasterizeSVGLucide(t *testing.T) {
	data, err := os.ReadFile(".cache/lucide-1.33.0/icons/search.svg")
	if err != nil {
		t.Skip("Lucide cache missing; run go run ./cmd/generate-lucide")
	}
	mask, err := RasterizeSVG(data, 40)
	if err != nil {
		t.Fatal(err)
	}
	nz := 0
	for _, p := range mask.Pix {
		if p > 0 {
			nz++
		}
	}
	if nz == 0 {
		t.Fatal("empty mask")
	}
}

func TestEnsureIconLazyPack(t *testing.T) {
	a := NewAtlasScale(2)
	uv, ok := a.EnsureIcon(icons.Search)
	if !ok {
		t.Fatal("EnsureIcon failed")
	}
	if uv.W <= 0 || uv.H <= 0 {
		t.Fatalf("bad uv: %+v", uv)
	}
	uv2, ok := a.IconUV(icons.Search.Name)
	if !ok || uv2 != uv {
		t.Fatalf("cache miss: %+v vs %+v", uv, uv2)
	}
}

func TestEnsureIconEmpty(t *testing.T) {
	a := NewAtlasScale(1)
	if _, ok := a.EnsureIcon(icons.Icon{}); ok {
		t.Fatal("expected miss for empty icon")
	}
}

func TestGrowMonoPreservesIcon(t *testing.T) {
	a := NewAtlasScale(2)
	uvBefore, ok := a.EnsureIcon(icons.Check)
	if !ok {
		t.Fatal("pack check")
	}
	eBefore := a.icons[icons.Check.Name]
	oldH := a.monoH
	a.growMono(a.monoH * 2)
	uvAfter, ok := a.IconUV(icons.Check.Name)
	if !ok {
		t.Fatal("check missing after growMono")
	}
	eAfter := a.icons[icons.Check.Name]
	if eAfter.physX != eBefore.physX || eAfter.physY != eBefore.physY ||
		eAfter.physW != eBefore.physW || eAfter.physH != eBefore.physH {
		t.Fatalf("phys coords changed: %+v vs %+v", eBefore, eAfter)
	}
	want := insetUV(eBefore.physX, eBefore.physY, eBefore.physW, eBefore.physH, a.monoW, a.monoH)
	if uvAfter != want {
		t.Fatalf("UV not recomputed for new height: got %+v want %+v (before %+v oldH=%d)", uvAfter, want, uvBefore, oldH)
	}
	if a.monoH != oldH*2 {
		t.Fatalf("expected monoH=%d, got %d", oldH*2, a.monoH)
	}
}

func TestGrowMonoRemapsDrawListUVs(t *testing.T) {
	a := NewAtlasScale(1)
	uv, ok := a.EnsureIcon(icons.Check)
	if !ok {
		t.Fatal("pack check")
	}
	var dl DrawList
	a.BindDrawList(&dl)
	defer a.BindDrawList(nil)

	dl.AddTexQuad(Rect{X: 0, Y: 0, W: 20, H: 20}, uv, Color{R: 1, G: 1, B: 1, A: 1})
	oldH := a.monoH
	vBefore := dl.Vertices[0].UV[1]

	a.growMono(a.monoH * 2)
	scale := float32(oldH) / float32(a.monoH)
	wantV := vBefore * scale
	if dl.Vertices[0].UV[1] != wantV {
		t.Fatalf("vertex V not remapped: got %v want %v", dl.Vertices[0].UV[1], wantV)
	}
	// Map UV must match remapped verts (same physical cell, new page height).
	mapUV, ok := a.IconUV(icons.Check.Name)
	if !ok {
		t.Fatal("icon missing after grow")
	}
	if dl.Vertices[0].UV[0] != mapUV.X || dl.Vertices[0].UV[1] != mapUV.Y {
		t.Fatalf("vert UV %+v != map UV top-left (%v,%v)", dl.Vertices[0].UV, mapUV.X, mapUV.Y)
	}
}
