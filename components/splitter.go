package components

import (
	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/theme"
)

const (
	splitHandleHit = 6
	minPaneSize    = 80
)

// SplitSection is one pane in a Splitter. Size > 0 is a fixed main-axis size in
// pixels (user-resizable via the adjacent handle). Size == 0 means the pane
// flexes to fill remaining space.
type SplitSection struct {
	El   *layout.Element
	Size float32
}

type splitHandle struct {
	el      *layout.Element
	index   int
	hovered bool
}

// Splitter arranges multiple child elements along an axis with draggable handles
// between them. Idle handles use theme.Border; hovered or active handles use
// theme.Accent.
type Splitter struct {
	El       *layout.Element
	theme    *theme.Theme
	axis     Axis
	sections []SplitSection
	handles  []splitHandle

	dragging      bool
	dragHandle    int
	dragStart     float32
	dragStartSize float32
	dragSection   int
}

// NewSplitter builds a splitter along axis with the given sections. At least two
// sections are required.
func NewSplitter(th *theme.Theme, axis Axis, sections ...SplitSection) *Splitter {
	if len(sections) < 2 {
		panic("splitter: need at least two sections")
	}
	s := &Splitter{
		theme:    th,
		axis:     axis,
		sections: append([]SplitSection(nil), sections...),
	}
	dir := layout.Column
	if axis == Horizontal {
		dir = layout.Row
	}
	var children []*layout.Element
	for i := range s.sections {
		s.applySectionStyle(i)
		children = append(children, s.sections[i].El)
		if i < len(s.sections)-1 {
			h := splitHandle{index: i}
			style := layout.Box().FlexShrink(0)
			if axis == Horizontal {
				style = style.W(splitHandleHit)
			} else {
				style = style.H(splitHandleHit)
			}
			h.el = layout.New(style)
			idx := i
			h.el.Paint = s.paintHandle(idx)
			h.el.OnMouse = s.mouseHandle(idx)
			s.handles = append(s.handles, h)
			children = append(children, h.el)
		}
	}
	s.El = layout.New(layout.Box().Direction(dir).FlexGrow(1), children...)
	return s
}

func (s *Splitter) applySectionStyle(i int) {
	sec := &s.sections[i]
	style := layout.Box().FlexShrink(0)
	if sec.Size > 0 {
		if s.axis == Horizontal {
			style = style.W(sec.Size)
		} else {
			style = style.H(sec.Size)
		}
	} else {
		style = style.FlexGrow(1)
	}
	sec.El.Style = style
	sec.El.ReapplyStyle()
}

func (s *Splitter) resizeTarget(handleIdx int) int {
	if s.sections[handleIdx].Size > 0 {
		return handleIdx
	}
	if handleIdx+1 < len(s.sections) && s.sections[handleIdx+1].Size > 0 {
		return handleIdx + 1
	}
	return handleIdx
}

func (s *Splitter) maxSizeForSection(i int) float32 {
	var total float32
	if s.axis == Horizontal {
		total = s.El.Frame.W
	} else {
		total = s.El.Frame.H
	}
	total -= float32(len(s.handles)) * splitHandleHit
	for j, sec := range s.sections {
		if j == i {
			continue
		}
		if sec.Size > 0 {
			total -= sec.Size
		} else {
			total -= minPaneSize
		}
	}
	return f32max(minPaneSize, total)
}

func (s *Splitter) pointerAlong(m *input.Mouse) float32 {
	if s.axis == Horizontal {
		return m.X
	}
	return m.Y
}

func (s *Splitter) startDrag(handleIdx int, m *input.Mouse) {
	s.dragging = true
	s.dragHandle = handleIdx
	s.dragStart = s.pointerAlong(m)
	s.dragSection = s.resizeTarget(handleIdx)
	s.dragStartSize = s.sections[s.dragSection].Size
}

func (s *Splitter) updateDrag(m *input.Mouse) {
	delta := s.pointerAlong(m) - s.dragStart
	sec := &s.sections[s.dragSection]
	newSize := s.dragStartSize
	if s.dragSection <= s.dragHandle {
		newSize += delta
	} else {
		newSize -= delta
	}
	newSize = clampf(newSize, minPaneSize, s.maxSizeForSection(s.dragSection))
	sec.Size = newSize
	s.applySectionStyle(s.dragSection)
}

func (s *Splitter) paintHandle(idx int) layout.PaintFunc {
	return func(dl *render.DrawList, _ *render.FontAtlas) {
		h := &s.handles[idx]
		f := h.el.Frame
		active := h.hovered || (s.dragging && s.dragHandle == idx)
		col := s.theme.Border
		if active {
			col = s.theme.Accent
		}
		var line render.Rect
		if s.axis == Horizontal {
			cx := f.X + (f.W-1)/2
			line = render.Rect{X: cx, Y: f.Y, W: 1, H: f.H}
		} else {
			cy := f.Y + (f.H-1)/2
			line = render.Rect{X: f.X, Y: cy, W: f.W, H: 1}
		}
		dl.AddRect(line, col)
	}
}

func (s *Splitter) setResizeCursor(m *input.Mouse) {
	if s.axis == Horizontal {
		m.SetCursor(input.CursorResizeEW)
	} else {
		m.SetCursor(input.CursorResizeNS)
	}
}

func (s *Splitter) mouseHandle(idx int) layout.MouseFunc {
	return func(e *layout.Element, m *input.Mouse) {
		h := &s.handles[idx]
		inside := e.Frame.Contains(m.X, m.Y)
		h.hovered = inside

		if s.dragging && s.dragHandle == idx {
			s.setResizeCursor(m)
			if m.Down {
				s.updateDrag(m)
				m.Consumed = true
			} else {
				s.dragging = false
			}
			return
		}

		if inside {
			s.setResizeCursor(m)
		}
		if m.Pressed && inside {
			s.startDrag(idx, m)
			m.Consumed = true
		}
	}
}
