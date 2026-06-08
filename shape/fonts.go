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

// FontSystem resolves faces (UI primary + mono editor + system fallback) for shaping.
type FontSystem struct {
	fontMap   *fontscan.FontMap
	primary   *font.Face // UI sans-serif (Source Sans 3)
	mono      *font.Face // editor monospace (Source Code Pro)
	scale     float32
	pixelSize fixed.Int26_6

	faceID   map[*font.Face]uint32
	idFace   map[uint32]*font.Face
	nextID   uint32
	segment  shaping.Segmenter
	shaper   shaping.HarfbuzzShaper

	metrics          Metrics
	metricsCache     map[float32]Metrics
	monoMetrics      Metrics
	monoMetricsCache map[float32]Metrics
}

// Metrics describes line layout in logical pixels.
type Metrics struct {
	Ascent     float32
	Descent    float32
	LineHeight float32
}

// NewFontSystem loads Source Sans 3 (UI) and Source Code Pro (editor mono),
// optionally indexing system fonts for fallback.
func NewFontSystem(scale float32, useSystemFonts bool) (*FontSystem, error) {
	if scale < 1 {
		scale = 1
	}
	px := int(logicalFontPx * float64(scale) + 0.5)
	if px < 1 {
		px = 1
	}

	primary, err := font.ParseTTF(bytes.NewReader(render.SourceSans3TTF))
	if err != nil {
		return nil, err
	}
	primary.SetPpem(uint16(px), uint16(px))

	mono, err := font.ParseTTF(bytes.NewReader(render.SourceCodeProTTF))
	if err != nil {
		return nil, err
	}
	mono.SetPpem(uint16(px), uint16(px))

	fm := fontscan.NewFontMap(log.Default())
	uiMD := primary.Describe()
	fm.AddFace(primary, fontscan.Location{File: "SourceSans3-Regular.ttf"}, uiMD)
	monoMD := mono.Describe()
	fm.AddFace(mono, fontscan.Location{File: "SourceCodePro-Regular.ttf"}, monoMD)
	fm.SetQuery(fontscan.Query{
		Families: []string{"Source Sans 3", "sans-serif", "monospace"},
		Aspect:   font.Aspect{},
	})
	if useSystemFonts {
		_ = fm.UseSystemFonts("") // degrade to embedded faces when unavailable
	}

	fs := &FontSystem{
		fontMap:          fm,
		primary:          primary,
		mono:             mono,
		scale:            scale,
		pixelSize:        fixed.I(px),
		faceID:           make(map[*font.Face]uint32),
		idFace:           make(map[uint32]*font.Face),
		metricsCache:     make(map[float32]Metrics),
		monoMetricsCache: make(map[float32]Metrics),
	}
	fs.registerFace(primary)
	fs.registerFace(mono)
	fs.metrics = fs.computeMetrics(primary)
	fs.monoMetrics = fs.computeMetrics(mono)
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

// Metrics returns UI line metrics in logical pixels at the default size.
func (fs *FontSystem) Metrics() Metrics { return fs.metrics }

// MonoMetrics returns editor mono line metrics at the default size.
func (fs *FontSystem) MonoMetrics() Metrics { return fs.monoMetrics }

// MetricsAt returns UI line metrics scaled to logicalSize (logical px).
func (fs *FontSystem) MetricsAt(logicalSize float32) Metrics {
	return fs.metricsAt(logicalSize, fs.metrics, fs.metricsCache)
}

// MonoMetricsAt returns editor mono line metrics scaled to logicalSize.
func (fs *FontSystem) MonoMetricsAt(logicalSize float32) Metrics {
	return fs.metricsAt(logicalSize, fs.monoMetrics, fs.monoMetricsCache)
}

func (fs *FontSystem) metricsAt(logicalSize float32, base Metrics, cache map[float32]Metrics) Metrics {
	if logicalSize <= 0 {
		return base
	}
	if logicalSize == float32(logicalFontPx) {
		return base
	}
	if m, ok := cache[logicalSize]; ok {
		return m
	}
	scale := logicalSize / float32(logicalFontPx)
	m := Metrics{
		Ascent:     base.Ascent * scale,
		Descent:    base.Descent * scale,
		LineHeight: base.LineHeight * scale,
	}
	cache[logicalSize] = m
	return m
}

// DefaultLogicalSize is the base UI font size in logical pixels.
func DefaultLogicalSize() float32 { return float32(logicalFontPx) }

// Scale returns the device pixel scale.
func (fs *FontSystem) Scale() float32 { return fs.scale }

// PixelSize returns the shaping size in pixels.
func (fs *FontSystem) PixelSize() fixed.Int26_6 { return fs.pixelSize }

// Primary returns the UI sans-serif face.
func (fs *FontSystem) Primary() *font.Face { return fs.primary }

// Mono returns the editor monospace face.
func (fs *FontSystem) Mono() *font.Face { return fs.mono }

func (fs *FontSystem) computeMetrics(face *font.Face) Metrics {
	in := fs.baseInputFace([]rune("Ag"), face)
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

// baseInput builds a shaping input for the UI face.
func (fs *FontSystem) baseInput(text []rune) shaping.Input {
	return fs.baseInputFace(text, fs.primary)
}

// baseInputMono builds a shaping input for the editor mono face.
func (fs *FontSystem) baseInputMono(text []rune) shaping.Input {
	return fs.baseInputFace(text, fs.mono)
}

// baseInputFace builds a shaping input for a rune slice with default LTR paragraph direction.
func (fs *FontSystem) baseInputFace(text []rune, face *font.Face) shaping.Input {
	return shaping.Input{
		Text:      text,
		RunStart:  0,
		RunEnd:    len(text),
		Direction: di.DirectionLTR,
		Face:      face,
		Size:      fs.pixelSize,
		Script:    language.Latin,
		Language:  language.NewLanguage("en"),
	}
}
