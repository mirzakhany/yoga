package render

import (
	_ "embed"
	"image"
	"image/color"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

// Source Code Pro (OFL) is embedded so the framework ships a high-quality
// monospace face with no runtime font dependency.
//
//go:embed assets/SourceCodePro-Regular.ttf
var sourceCodeProTTF []byte

// FontAtlas is a CPU-side monospace glyph cache. At construction it rasterizes
// every printable ASCII glyph (plus a small set of vector icons) from a real
// TrueType face into a single 8-bit coverage bitmap arranged as a uniform grid,
// and records the 0..1 UV rectangle of each cell. The pixel data is later
// uploaded once by the renderer as an R8 texture; text is then drawn purely by
// emitting textured quads that index into this atlas.
//
// HiDPI: the atlas bakes at (logical size * scale) pixels, while CellW/CellH are
// reported in *logical* pixels. The UI lays out and emits quads in logical
// units; because the renderer rasterizes onto the full framebuffer (which is
// `scale` times larger), each logical glyph quad covers exactly its baked pixel
// footprint, so text stays crisp on Retina/HiDPI displays.
type FontAtlas struct {
	Pixels []byte // pixelRows x pixelCols, 1 coverage byte per pixel
	W, H   int    // atlas dimensions in pixels

	CellW float32 // monospace advance width in LOGICAL pixels
	CellH float32 // line height in LOGICAL pixels

	cellPxW int // baked cell width in physical pixels
	cellPxH int // baked cell height in physical pixels

	glyphs map[rune]Rect // rune -> UV rect in 0..1 atlas space
	icons  map[string]Rect
}

const (
	atlasCols     = 16   // cells per row
	asciiRows     = 8    // rows 0..7 hold code points 0..127
	iconRow       = 8    // row 8 holds baked vector icons
	atlasRows     = 9    // ascii rows + icon row
	logicalFontPx = 14.0 // logical font height
)

// NewMonoAtlas bakes Source Code Pro at 1x (used by headless/non-HiDPI paths).
func NewMonoAtlas() *FontAtlas { return NewMonoAtlasScale(1) }

// NewMonoAtlasScale bakes Source Code Pro at the given device pixel scale
// (e.g. 2 for a typical Retina display).
func NewMonoAtlasScale(scale float32) *FontAtlas {
	if scale < 1 {
		scale = 1
	}
	fnt, err := opentype.Parse(sourceCodeProTTF)
	if err != nil {
		panic("render: cannot parse embedded font: " + err.Error())
	}
	pixelSize := logicalFontPx * float64(scale)
	face, err := opentype.NewFace(fnt, &opentype.FaceOptions{
		Size:    pixelSize,
		DPI:     72, // with DPI 72, 1pt == 1px, so Size is the pixel height
		Hinting: font.HintingFull,
	})
	if err != nil {
		panic("render: cannot create face: " + err.Error())
	}
	defer face.Close()

	metrics := face.Metrics()
	ascent := metrics.Ascent.Ceil()
	cellPxH := metrics.Height.Ceil()
	if h := metrics.Ascent.Ceil() + metrics.Descent.Ceil(); h > cellPxH {
		cellPxH = h
	}
	advance, _ := face.GlyphAdvance('m')
	cellPxW := advance.Ceil()
	if cellPxW < 1 {
		cellPxW = 1
	}

	a := &FontAtlas{
		cellPxW: cellPxW,
		cellPxH: cellPxH,
		W:       atlasCols * cellPxW,
		H:       atlasRows * cellPxH,
		CellW:   float32(cellPxW) / scale,
		CellH:   float32(cellPxH) / scale,
		glyphs:  make(map[rune]Rect, 128),
		icons:   make(map[string]Rect, 8),
	}
	a.Pixels = make([]byte, a.W*a.H)

	a.bakeASCII(face, ascent)
	a.bakeIcons()
	return a
}

// cellUV returns the 0..1 UV rectangle for the grid cell at (col,row).
func (a *FontAtlas) cellUV(col, row int) Rect {
	return Rect{
		X: float32(col*a.cellPxW) / float32(a.W),
		Y: float32(row*a.cellPxH) / float32(a.H),
		W: float32(a.cellPxW) / float32(a.W),
		H: float32(a.cellPxH) / float32(a.H),
	}
}

// bakeASCII rasterizes code points 32..126 with x/image's software drawer.
func (a *FontAtlas) bakeASCII(face font.Face, ascent int) {
	for r := rune(32); r < 127; r++ {
		cell := image.NewAlpha(image.Rect(0, 0, a.cellPxW, a.cellPxH))
		d := &font.Drawer{
			Dst:  cell,
			Src:  image.NewUniform(color.Alpha{A: 255}),
			Face: face,
			Dot:  fixed.P(0, ascent),
		}
		d.DrawString(string(r))

		col := int(r) % atlasCols
		row := int(r) / atlasCols
		a.blit(col, row, cell)
		a.glyphs[r] = a.cellUV(col, row)
	}
}

// blit copies an 8-bit alpha glyph image into the atlas cell at (col,row).
func (a *FontAtlas) blit(col, row int, src *image.Alpha) {
	ox := col * a.cellPxW
	oy := row * a.cellPxH
	for y := 0; y < a.cellPxH && y < src.Rect.Dy(); y++ {
		for x := 0; x < a.cellPxW && x < src.Rect.Dx(); x++ {
			a.Pixels[(oy+y)*a.W+(ox+x)] = src.Pix[y*src.Stride+x]
		}
	}
}

// bakeIcons procedurally rasterizes a few coverage-mask icons into the icon row,
// sized relative to the (scale-aware) cell so they stay sharp at any density.
// Real apps would compile a higher-resolution sprite sheet; baking icons into
// the same texture keeps the whole UI on one bind group / draw call.
func (a *FontAtlas) bakeIcons() {
	w, h := a.cellPxW, a.cellPxH
	fw, fh := float64(w), float64(h)

	defs := []struct {
		name string
		draw func(set func(x, y int))
	}{
		{"chevron-down", func(set func(x, y int)) {
			top := fh * 0.35
			bot := fh * 0.65
			for y := top; y <= bot; y++ {
				t := (y - top) / (bot - top) // 0 at apex base.. widen downward inverted
				half := (fw * 0.4) * (1 - t)
				cx := fw / 2
				for x := cx - half; x <= cx+half; x++ {
					set(int(x), int(y))
				}
			}
		}},
		{"circle", func(set func(x, y int)) {
			cx, cy := fw/2, fh/2
			rad := fh * 0.28
			for y := 0.0; y < fh; y++ {
				for x := 0.0; x < fw; x++ {
					dx, dy := x-cx, y-cy
					if dx*dx+dy*dy <= rad*rad {
						set(int(x), int(y))
					}
				}
			}
		}},
		{"dot", func(set func(x, y int)) {
			cx, cy := fw/2, fh/2
			rad := fh * 0.12
			for y := 0.0; y < fh; y++ {
				for x := 0.0; x < fw; x++ {
					dx, dy := x-cx, y-cy
					if dx*dx+dy*dy <= rad*rad {
						set(int(x), int(y))
					}
				}
			}
		}},
		{"square", func(set func(x, y int)) {
			x0, y0 := int(fw*0.2), int(fh*0.3)
			x1, y1 := int(fw*0.8), int(fh*0.7)
			for y := y0; y <= y1; y++ {
				for x := x0; x <= x1; x++ {
					set(x, y)
				}
			}
		}},
	}

	for i, def := range defs {
		col := i % atlasCols
		ox := col * a.cellPxW
		oy := iconRow * a.cellPxH
		def.draw(func(x, y int) {
			if x >= 0 && x < a.cellPxW && y >= 0 && y < a.cellPxH {
				a.Pixels[(oy+y)*a.W+(ox+x)] = 255
			}
		})
		a.icons[def.name] = a.cellUV(col, iconRow)
	}
}

// GlyphUV returns the atlas UV rect for a rune, falling back to '?' for any
// glyph that was not baked (e.g. non-ASCII).
func (a *FontAtlas) GlyphUV(r rune) Rect {
	if uv, ok := a.glyphs[r]; ok {
		return uv
	}
	return a.glyphs['?']
}

// IconUV returns the atlas UV rect for a named icon (see bakeIcons), or the
// zero rect if unknown.
func (a *FontAtlas) IconUV(name string) (Rect, bool) {
	uv, ok := a.icons[name]
	return uv, ok
}

// Measure returns the pixel size of a single-line string in this monospace font.
func (a *FontAtlas) Measure(s string) (w, h float32) {
	return float32(len([]rune(s))) * a.CellW, a.CellH
}

// DrawText appends textured quads for s with its top-left at (x, y), tinted by
// c. It returns the advance width. Newlines are not handled here; the editor
// performs its own per-line layout for virtualization.
func (a *FontAtlas) DrawText(dl *DrawList, s string, x, y float32, c Color) float32 {
	penX := x
	for _, r := range s {
		if r == ' ' {
			penX += a.CellW
			continue
		}
		dst := Rect{X: penX, Y: y, W: a.CellW, H: a.CellH}
		dl.AddTexQuad(dst, a.GlyphUV(r), c)
		penX += a.CellW
	}
	return penX - x
}
