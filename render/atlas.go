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
	atlasCols     = 16   // glyph cells per row
	asciiRows     = 8    // rows 0..7 hold code points 0..127
	logicalFontPx = 14.0 // logical font height
	logicalIconPx = 20.0 // logical icon cell size (square)
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

	// Atlas layout: rows 0..asciiRows-1 are the glyph grid; the icon region is
	// appended below it. We must know the final height before recording any UV
	// rect (cellUV divides by a.H), so the icon region is sized up front.
	iconPx := int(logicalIconPx*scale + 0.5)
	if iconPx < 8 {
		iconPx = 8
	}
	atlasW := atlasCols * cellPxW
	asciiH := asciiRows * cellPxH

	names := iconNames()
	perRow := atlasW / iconPx
	if perRow < 1 {
		perRow = 1
	}
	iconRows := (len(names) + perRow - 1) / perRow
	iconRegionH := iconRows * iconPx

	a := &FontAtlas{
		cellPxW: cellPxW,
		cellPxH: cellPxH,
		W:       atlasW,
		H:       asciiH + iconRegionH,
		CellW:   float32(cellPxW) / scale,
		CellH:   float32(cellPxH) / scale,
		glyphs:  make(map[rune]Rect, 128),
		icons:   make(map[string]Rect, len(names)),
	}
	a.Pixels = make([]byte, a.W*a.H)

	a.bakeASCII(face, ascent)
	a.packIcons(names, iconPx, asciiH, perRow)
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

// packIcons rasterizes each registered SVG icon to an iconPx coverage mask and
// shelf-packs them into the icon region that begins at pixel row regionY. Each
// icon occupies an iconPx-by-iconPx cell; perRow icons fit across the atlas
// width. Icon coverage is tinted by the vertex color at draw time, so a single
// monochrome mask renders in any theme color.
func (a *FontAtlas) packIcons(names []string, iconPx, regionY, perRow int) {
	for i, name := range names {
		mask, err := rasterizeIcon(iconRegistry[name], iconPx)
		if err != nil {
			continue
		}
		col := i % perRow
		row := i / perRow
		ox := col * iconPx
		oy := regionY + row*iconPx
		for y := 0; y < iconPx; y++ {
			for x := 0; x < iconPx; x++ {
				a.Pixels[(oy+y)*a.W+(ox+x)] = mask.Pix[y*mask.Stride+x]
			}
		}
		a.icons[name] = Rect{
			X: float32(ox) / float32(a.W),
			Y: float32(oy) / float32(a.H),
			W: float32(iconPx) / float32(a.W),
			H: float32(iconPx) / float32(a.H),
		}
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

// IconUV returns the atlas UV rect for a named icon (see packIcons), or the
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
