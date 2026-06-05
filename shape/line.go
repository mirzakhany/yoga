package shape

import (
	"sort"
	"unicode/utf8"

	"github.com/go-text/typesetting/di"
	"github.com/go-text/typesetting/font"
	"github.com/go-text/typesetting/shaping"
)

const tabWidthCols = 4

// Glyph is one visually-placed glyph in a shaped line.
type Glyph struct {
	FaceID      uint32
	GID         font.GID
	X, Y        float32 // top-left of quad in line coordinates (logical px)
	W, H        float32
	Advance     float32
	ClusterByte int // byte offset in line of cluster start
	ClusterLen  int // byte length of cluster in line
	Color       bool
}

// Line is a fully shaped, visually ordered text line.
type Line struct {
	Glyphs []Glyph
	Width  float32
	Runes  []rune // original line runes (without trailing newline)
}

// Shaper shapes single logical lines (no soft wrap).
type Shaper struct {
	fs         *FontSystem
	lastGlyphs []Glyph
}

// NewShaper returns a shaper bound to fonts.
func NewShaper(fs *FontSystem) *Shaper { return &Shaper{fs: fs} }

// ShapeLine shapes s as one LTR/RTL-aware line with tab expansion.
func (s *Shaper) ShapeLine(text string) Line {
	if text == "" {
		return Line{}
	}
	runes := []rune(text)
	var out Line
	out.Runes = runes
	x := float32(0)
	tabCol := 0

	segStart := 0
	flush := func(end int, tabStop bool) {
		if end <= segStart {
			return
		}
		segRunes := runes[segStart:end]
		byteBase := len(string(runes[:segStart]))
		w := s.shapeSegment(segRunes, byteBase, x)
		x += w
		out.Glyphs = append(out.Glyphs, s.lastGlyphs...)
		if tabStop {
			tabCol = int(x/s.cellWidth()) + tabWidthCols
			tabCol = (tabCol / tabWidthCols) * tabWidthCols
			x = float32(tabCol) * s.cellWidth()
		}
		segStart = end
	}

	for i, r := range runes {
		if r == '\t' {
			flush(i, true)
			segStart = i + 1
		}
	}
	flush(len(runes), false)

	out.Width = x
	return out
}

func (s *Shaper) cellWidth() float32 {
	// approximate monospace tab width from 'm' advance
	in := s.fs.baseInput([]rune("m"))
	out := s.fs.shaper.Shape(in)
	return toLogical(s.fs, out.Advance)
}

func (s *Shaper) shapeSegment(runes []rune, byteBase int, startX float32) float32 {
	if len(runes) == 0 {
		s.lastGlyphs = nil
		return 0
	}
	in := s.fs.baseInput(runes)
	runs := s.fs.segment.Split(in, s.fs)
	if len(runs) == 0 {
		s.lastGlyphs = nil
		return 0
	}

	outputs := make([]shaping.Output, len(runs))
	for i, run := range runs {
		run.Size = s.fs.pixelSize
		outputs[i] = s.fs.shaper.Shape(run)
	}
	computeBidiOrdering(di.DirectionLTR, outputs)

	sort.Slice(outputs, func(i, j int) bool {
		return outputs[i].VisualIndex < outputs[j].VisualIndex
	})

	var glyphs []Glyph
	x := startX
	for _, out := range outputs {
		faceID := s.fs.FaceID(out.Face)
		for _, g := range out.Glyphs {
			gx := x + toLogical(s.fs, g.XOffset)
			gy := s.fs.metrics.Ascent + toLogical(s.fs, g.YOffset)
			w := toLogical(s.fs, g.Width)
			h := toLogical(s.fs, g.Height)
			if w <= 0 {
				w = toLogical(s.fs, g.Advance)
			}
			if h <= 0 {
				h = s.fs.metrics.Ascent + s.fs.metrics.Descent
			}
			runeIdx := g.TextIndex()
			if runeIdx < 0 || runeIdx >= len(runes) {
				runeIdx = 0
			}
			clusterByte := byteBase + len(string(runes[:runeIdx]))
			clusterLen := utf8.RuneLen(runes[runeIdx])
			if clusterLen < 1 {
				clusterLen = 1
			}
			color := isColorGlyph(out.Face, g.GlyphID)
			glyphs = append(glyphs, Glyph{
				FaceID: faceID, GID: g.GlyphID,
				X: gx, Y: gy - toLogical(s.fs, g.YBearing),
				W: w, H: h,
				Advance:     toLogical(s.fs, g.Advance),
				ClusterByte: clusterByte,
				ClusterLen:  clusterLen,
				Color:       color,
			})
			x += toLogical(s.fs, g.Advance)
		}
	}
	s.lastGlyphs = glyphs
	return x - startX
}

func isColorGlyph(face *font.Face, gid font.GID) bool {
	if data := face.GlyphData(gid); data != nil {
		_, isColor := data.(font.GlyphColor)
		return isColor
	}
	return false
}

// Width returns shaped width of text.
func (s *Shaper) Width(text string) float32 {
	return s.ShapeLine(text).Width
}

// Measure returns width and line height for a single line string.
func (s *Shaper) Measure(text string) (w, h float32) {
	return s.ShapeLine(text).Width, s.fs.metrics.LineHeight
}

// XForByte returns the x coordinate for a byte offset within the line text.
func (l Line) XForByte(byteOff int) float32 {
	if byteOff <= 0 {
		return 0
	}
	for _, g := range l.Glyphs {
		end := g.ClusterByte + g.ClusterLen
		if byteOff <= g.ClusterByte {
			return g.X
		}
		if byteOff < end {
			return g.X + g.W*0.5
		}
	}
	return l.Width
}

// ByteForX maps a pixel x to the nearest byte offset (cluster boundary).
func (l Line) ByteForX(x float32) int {
	if x <= 0 || len(l.Glyphs) == 0 {
		return 0
	}
	for _, g := range l.Glyphs {
		mid := g.X + g.Advance*0.5
		if x < mid {
			return g.ClusterByte
		}
	}
	last := l.Glyphs[len(l.Glyphs)-1]
	return last.ClusterByte + last.ClusterLen
}

// PrevCluster returns the byte offset before off (grapheme/cluster aware).
func (l Line) PrevCluster(off int) int {
	if off <= 0 {
		return 0
	}
	prev := 0
	for _, g := range l.Glyphs {
		if g.ClusterByte >= off {
			break
		}
		prev = g.ClusterByte
	}
	return prev
}

// NextCluster returns the byte offset after off.
func (l Line) NextCluster(off int, lineLen int) int {
	for _, g := range l.Glyphs {
		end := g.ClusterByte + g.ClusterLen
		if end > off {
			return end
		}
	}
	return lineLen
}

// SelectionRects returns highlight rectangles for byte range [a,b) in line coords.
func (l Line) SelectionRects(a, b int, y, lineH float32) [][4]float32 {
	if a >= b {
		return nil
	}
	var rects [][4]float32
	for _, g := range l.Glyphs {
		gStart := g.ClusterByte
		gEnd := g.ClusterByte + g.ClusterLen
		if gEnd <= a || gStart >= b {
			continue
		}
		rects = append(rects, [4]float32{g.X, y, g.Advance, lineH})
	}
	return rects
}

// computeBidiOrdering resolves VisualIndex for shaped runs (from go-text).
func computeBidiOrdering(dir di.Direction, finalLine []shaping.Output) {
	bidiStart := -1
	for idx, run := range finalLine {
		basePosition := idx
		if dir.Progression() == di.TowardTopLeft {
			basePosition = len(finalLine) - 1 - idx
		}
		finalLine[idx].VisualIndex = int32(basePosition)
		if run.Direction == dir {
			if bidiStart != -1 {
				swapVisualOrder(finalLine[bidiStart:idx])
				bidiStart = -1
			}
		} else if bidiStart == -1 {
			bidiStart = idx
		}
	}
	if bidiStart != -1 {
		swapVisualOrder(finalLine[bidiStart:])
	}
}

func swapVisualOrder(subline []shaping.Output) {
	L := len(subline)
	for i := 0; i < L/2; i++ {
		j := (L - i) - 1
		subline[i].VisualIndex, subline[j].VisualIndex = subline[j].VisualIndex, subline[i].VisualIndex
	}
}
