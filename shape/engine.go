package shape

import (
	"github.com/mirzakhany/yoga/render"
)

// Engine bundles fonts, atlas, shaping, and line cache for UI text.
type Engine struct {
	Fonts  *FontSystem
	Atlas  *render.FontAtlas
	Shaper *Shaper
	Cache  *LineCache
}

// NewEngine creates a text engine at the given device scale.
func NewEngine(scale float32, useSystemFonts bool) (*Engine, error) {
	fonts, err := NewFontSystem(scale, useSystemFonts)
	if err != nil {
		return nil, err
	}
	atlas := render.NewAtlasScale(scale)
	shaper := NewShaper(fonts)
	return &Engine{
		Fonts:  fonts,
		Atlas:  atlas,
		Shaper: shaper,
		Cache:  NewLineCache(shaper),
	}, nil
}

// Metrics returns line metrics in logical pixels.
func (e *Engine) Metrics() Metrics { return e.Fonts.Metrics() }

// Line shapes and caches a single line.
func (e *Engine) Line(text string) Line { return e.Cache.Get(text) }

// Measure returns width and height for a single-line string.
func (e *Engine) Measure(s string) (w, h float32) {
	return e.Shaper.Measure(s)
}

// LineAt returns a shaped line at logicalSize (0 = default UI size).
func (e *Engine) LineAt(text string, logicalSize float32) Line {
	return e.Cache.GetAt(text, logicalSize)
}

// MeasureAt returns width and height for a single-line string at logicalSize.
func (e *Engine) MeasureAt(s string, logicalSize float32) (w, h float32) {
	return e.Shaper.MeasureAt(s, logicalSize)
}

// DrawStringTop draws with top-left y (convenience for UI chrome).
func (e *Engine) DrawStringTop(dl *render.DrawList, s string, x, topY float32, c render.Color) float32 {
	return e.DrawString(dl, s, x, topY+e.Metrics().Ascent, c)
}

// DrawStringTopAt draws at logicalSize with top-left y.
func (e *Engine) DrawStringTopAt(dl *render.DrawList, s string, x, topY float32, c render.Color, logicalSize float32) float32 {
	m := e.Fonts.MetricsAt(logicalSize)
	return e.DrawStringAt(dl, s, x, topY+m.Ascent, c, logicalSize)
}

// DrawStringAt draws a single line at baseline y and logicalSize.
func (e *Engine) DrawStringAt(dl *render.DrawList, s string, x, baselineY float32, c render.Color, logicalSize float32) float32 {
	ln := e.LineAt(s, logicalSize)
	topY := baselineY - e.Fonts.MetricsAt(logicalSize).Ascent
	for _, g := range ln.Glyphs {
		face := e.Fonts.Face(g.FaceID)
		entry := e.Atlas.EnsureGlyph(g.FaceID, face, g.GID)
		w, h := entry.W, entry.H
		if logicalSize > 0 && logicalSize != DefaultLogicalSize() {
			scale := logicalSize / DefaultLogicalSize()
			w *= scale
			h *= scale
		}
		dst := render.Rect{X: x + g.X, Y: topY + g.Y, W: w, H: h}
		if entry.Color {
			dl.AddGlyphQuad(dst, entry.UV, render.PageColor, c)
		} else {
			dl.AddGlyphQuad(dst, entry.UV, render.PageMono, c)
		}
	}
	return ln.Width
}

// DrawString draws a single line at baseline y.
func (e *Engine) DrawString(dl *render.DrawList, s string, x, baselineY float32, c render.Color) float32 {
	ln := e.Line(s)
	topY := baselineY - e.Metrics().Ascent
	for _, g := range ln.Glyphs {
		face := e.Fonts.Face(g.FaceID)
		entry := e.Atlas.EnsureGlyph(g.FaceID, face, g.GID)
		dst := render.Rect{X: x + g.X, Y: topY + g.Y, W: entry.W, H: entry.H}
		if entry.Color {
			dl.AddGlyphQuad(dst, entry.UV, render.PageColor, c)
		} else {
			dl.AddGlyphQuad(dst, entry.UV, render.PageMono, c)
		}
	}
	return ln.Width
}

// DrawLineGlyphs draws an already-shaped line with per-glyph colors from tintAt byte offset.
func (e *Engine) DrawLineGlyphs(dl *render.DrawList, ln Line, x, topY float32, tint func(byteOff int) render.Color) {
	for _, g := range ln.Glyphs {
		face := e.Fonts.Face(g.FaceID)
		entry := e.Atlas.EnsureGlyph(g.FaceID, face, g.GID)
		dst := render.Rect{X: x + g.X, Y: topY + g.Y, W: entry.W, H: entry.H}
		col := tint(g.ClusterByte)
		if entry.Color {
			dl.AddGlyphQuad(dst, entry.UV, render.PageColor, col)
		} else {
			dl.AddGlyphQuad(dst, entry.UV, render.PageMono, col)
		}
	}
}

// FlushAtlas uploads dirty atlas regions to the GPU renderer when present.
func (e *Engine) FlushAtlas(r *render.Renderer) error {
	if r == nil {
		return nil
	}
	for _, d := range e.Atlas.DirtyRects() {
		if err := r.UpdateAtlasRegion(d); err != nil {
			return err
		}
	}
	if e.Atlas.NeedsFullRebuild() {
		return r.UpdateAtlas(e.Atlas)
	}
	e.Atlas.ClearDirty()
	return nil
}
