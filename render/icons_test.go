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

func TestGrowMonoRepacksUsedIcon(t *testing.T) {
	a := NewAtlasScale(2)
	if _, ok := a.EnsureIcon(icons.Check); !ok {
		t.Fatal("pack check")
	}
	a.growMono(a.monoH * 2)
	if _, ok := a.IconUV(icons.Check.Name); !ok {
		t.Fatal("check missing after growMono")
	}
}
