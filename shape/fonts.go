package shape

import (
	"bytes"
	"log"

	"github.com/go-text/typesetting/di"
	"github.com/go-text/typesetting/font"
	"github.com/go-text/typesetting/fontscan"
	"github.com/go-text/typesetting/language"
	"github.com/go-text/typesetting/shaping"
	"golang.org/x/image/math/fixed"

	"github.com/mirzakhany/yoga/render"
)

const logicalFontPx = 14.0

// FontSystem resolves faces (primary + system fallback) for shaping.
type FontSystem struct {
	fontMap   *fontscan.FontMap
	primary   *font.Face
	scale     float32
	pixelSize fixed.Int26_6

	faceID   map[*font.Face]uint32
	idFace   map[uint32]*font.Face
	nextID   uint32
	segment  shaping.Segmenter
	shaper   shaping.HarfbuzzShaper

	metrics      Metrics
	metricsCache map[float32]Metrics
}

// Metrics describes line layout in logical pixels.
type Metrics struct {
	Ascent     float32
	Descent    float32
	LineHeight float32
}

// NewFontSystem loads Source Code Pro as primary and optionally indexes system fonts.
func NewFontSystem(scale float32, useSystemFonts bool) (*FontSystem, error) {
	if scale < 1 {
		scale = 1
	}
	px := int(logicalFontPx * float64(scale) + 0.5)
	if px < 1 {
		px = 1
	}

	primary, err := font.ParseTTF(bytes.NewReader(render.SourceCodeProTTF))
	if err != nil {
		return nil, err
	}
	primary.SetPpem(uint16(px), uint16(px))

	fm := fontscan.NewFontMap(log.Default())
	md := primary.Describe()
	fm.AddFace(primary, fontscan.Location{File: "SourceCodePro-Regular.ttf"}, md)
	fm.SetQuery(fontscan.Query{
		Families: []string{"Source Code Pro", "monospace", "sans-serif"},
		Aspect:   font.Aspect{},
	})
	if useSystemFonts {
		_ = fm.UseSystemFonts("") // degrade to primary-only when unavailable
	}

	fs := &FontSystem{
		fontMap:      fm,
		primary:      primary,
		scale:        scale,
		pixelSize:    fixed.I(px),
		faceID:       make(map[*font.Face]uint32),
		idFace:       make(map[uint32]*font.Face),
		metricsCache: make(map[float32]Metrics),
	}
	fs.registerFace(primary)
	fs.metrics = fs.computeMetrics()
	return fs, nil
}

func (fs *FontSystem) registerFace(face *font.Face) uint32 {
	if id, ok := fs.faceID[face]; ok {
		return id
	}
	id := fs.nextID
	fs.nextID++
	fs.faceID[face] = id
	fs.idFace[id] = face
	return id
}

// FaceID returns a stable atlas key for face.
func (fs *FontSystem) FaceID(face *font.Face) uint32 { return fs.registerFace(face) }

// Face returns the face for id.
func (fs *FontSystem) Face(id uint32) *font.Face { return fs.idFace[id] }

// ResolveFace implements shaping.Fontmap.
func (fs *FontSystem) ResolveFace(r rune) *font.Face {
	f := fs.fontMap.ResolveFace(r)
	if f == nil {
		return fs.primary
	}
	fs.registerFace(f)
	return f
}

// SetScript implements shaping.FontmapScript.
func (fs *FontSystem) SetScript(s language.Script) { fs.fontMap.SetScript(s) }

// Metrics returns line metrics in logical pixels at the default UI size.
func (fs *FontSystem) Metrics() Metrics { return fs.metrics }

// MetricsAt returns line metrics scaled to logicalSize (logical px).
func (fs *FontSystem) MetricsAt(logicalSize float32) Metrics {
	if logicalSize <= 0 {
		return fs.metrics
	}
	if logicalSize == float32(logicalFontPx) {
		return fs.metrics
	}
	if m, ok := fs.metricsCache[logicalSize]; ok {
		return m
	}
	scale := logicalSize / float32(logicalFontPx)
	m := Metrics{
		Ascent:     fs.metrics.Ascent * scale,
		Descent:    fs.metrics.Descent * scale,
		LineHeight: fs.metrics.LineHeight * scale,
	}
	fs.metricsCache[logicalSize] = m
	return m
}

// DefaultLogicalSize is the base UI font size in logical pixels.
func DefaultLogicalSize() float32 { return float32(logicalFontPx) }

// Scale returns the device pixel scale.
func (fs *FontSystem) Scale() float32 { return fs.scale }

// PixelSize returns the shaping size in pixels.
func (fs *FontSystem) PixelSize() fixed.Int26_6 { return fs.pixelSize }

// Primary returns the primary monospace face.
func (fs *FontSystem) Primary() *font.Face { return fs.primary }

func (fs *FontSystem) computeMetrics() Metrics {
	in := fs.baseInput([]rune("Ag"))
	out := fs.shaper.Shape(in)
	a := toLogical(fs, out.LineBounds.Ascent)
	d := -toLogical(fs, out.LineBounds.Descent)
	if d < 0 {
		d = -d
	}
	lh := a + d + 3/fs.scale
	return Metrics{Ascent: a, Descent: d, LineHeight: lh}
}

func fixedToFloat(v fixed.Int26_6) float32 { return float32(v) / 64 }

func toLogical(fs *FontSystem, px fixed.Int26_6) float32 {
	return fixedToFloat(px) / fs.scale
}

// baseInput builds a shaping input for a rune slice with default LTR paragraph direction.
func (fs *FontSystem) baseInput(text []rune) shaping.Input {
	return shaping.Input{
		Text:      text,
		RunStart:  0,
		RunEnd:    len(text),
		Direction: di.DirectionLTR,
		Face:      fs.primary,
		Size:      fs.pixelSize,
		Script:    language.Latin,
		Language:  language.NewLanguage("en"),
	}
}
