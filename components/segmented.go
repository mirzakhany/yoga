package components

import (
	"github.com/mirzakhany/yoga"
	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
	"github.com/mirzakhany/yoga/theme"
)

// SegmentItem is one choice in a Segmented control. Icon and Label are both
// optional: give an Icon for an icon-only switch, a Label for a text switch, or
// both. Value is an opaque tag handed to the OnChange(value) callback.
type SegmentItem struct {
	Icon  string // optional sprite name drawn before the label
	Label string // optional text
	Value string // opaque value reported by OnChangeValue
}

// Segmented is a single-select switch: a row of equal-width cells where exactly
// one is active (filled). Each cell may carry an icon, a label, or both. It is
// the general form of controls like a split-axis toggle or a view switcher.
type Segmented struct {
	El       *layout.Element
	Items    []SegmentItem
	Selected int

	// OnChange fires with the newly selected index; OnChangeValue with its Value.
	OnChange      func(index int)
	OnChangeValue func(value string)

	hover int
	cellW float32
}

const (
	segPad          = 3 // padding between the track edge and the cells
	segCellPadX     = 8 // horizontal padding inside each cell
	segCellGap      = 2 // gap between adjacent cells
	segIconLabelGap = 6 // gap between an icon and its label
)

// NewSegmented builds a segmented switch over the given items, defaulting the
// selection to the first item.
func NewSegmented(items []SegmentItem) *Segmented {
	th := theme.Current()
	s := &Segmented{Items: items, hover: -1}
	s.cellW = s.computeCellW()
	n := float32(len(items))
	w := n*s.cellW + f32max(0, n-1)*segCellGap + 2*segPad
	s.El = layout.New(layout.Box().W(w).H(th.Metrics.ControlHeight).FlexShrink(0))
	s.El.Paint = s.paint
	s.El.OnMouse = s.onMouse
	return s
}

// Changed sets the index callback. Chainable.
func (s *Segmented) Changed(fn func(int)) *Segmented { s.OnChange = fn; return s }

// ChangedValue sets the value callback. Chainable.
func (s *Segmented) ChangedValue(fn func(string)) *Segmented { s.OnChangeValue = fn; return s }

// Select sets the active index without firing callbacks.
func (s *Segmented) Select(i int) {
	if i >= 0 && i < len(s.Items) {
		s.Selected = i
	}
}

// computeCellW returns the shared cell width: the widest item's content plus
// horizontal padding, kept at least square so icon-only cells look balanced.
func (s *Segmented) computeCellW() float32 {
	th := theme.Current()
	text := yoga.Text()
	iconSz := th.Metrics.IconSizeSM
	var maxContent float32
	for _, it := range s.Items {
		var c float32
		if it.Icon != "" {
			c += iconSz
		}
		if it.Label != "" {
			lw, _ := text.MeasureAt(it.Label, th.Typography.Body.Size)
			if it.Icon != "" {
				c += segIconLabelGap
			}
			c += lw
		}
		if c > maxContent {
			maxContent = c
		}
	}
	cellW := maxContent + 2*segCellPadX
	if min := th.Metrics.ControlHeight - 2*segPad; cellW < min {
		cellW = min // square-ish floor for icon-only cells
	}
	return cellW
}

// cellRect returns the rect of cell i within the track frame.
func (s *Segmented) cellRect(i int) render.Rect {
	f := s.El.Frame
	x := f.X + segPad + float32(i)*(s.cellW+segCellGap)
	return render.Rect{X: x, Y: f.Y + segPad, W: s.cellW, H: f.H - 2*segPad}
}

func (s *Segmented) paint(dl *render.DrawList, text *shape.Engine) {
	th := theme.Current()
	f := s.El.Frame
	dl.AddRoundedRectBorder(f, th.Radius.Medium, th.Stroke.Thin, th.ChromeMuted, th.Border)

	iconSz := th.Metrics.IconSizeSM
	style := th.Typography.Body
	for i, it := range s.Items {
		active := i == s.Selected
		cell := s.cellRect(i)
		switch {
		case active:
			dl.AddRoundedRect(cell, th.Radius.Small, th.ListActive)
		case i == s.hover:
			dl.AddRoundedRect(cell, th.Radius.Small, th.ListHover)
		}

		fg := th.ForegroundSubtle
		if active {
			fg = th.Foreground
		}

		// Center icon + label as a group within the cell.
		var content float32
		var lw, lh float32
		if it.Icon != "" {
			content += iconSz
		}
		if it.Label != "" {
			lw, lh = text.MeasureAt(it.Label, style.Size)
			if it.Icon != "" {
				content += segIconLabelGap
			}
			content += lw
		}
		x := cell.X + (cell.W-content)/2
		cy := cell.Y + cell.H/2
		if it.Icon != "" {
			yoga.Icons().Draw(dl, it.Icon, render.Rect{X: x, Y: cy - iconSz/2, W: iconSz, H: iconSz}, fg)
			x += iconSz + segIconLabelGap
		}
		if it.Label != "" {
			text.DrawStringTopAt(dl, it.Label, x, cy-lh/2, fg, style.Size)
		}
	}
}

func (s *Segmented) onMouse(_ *layout.Element, m *input.Mouse) {
	s.hover = -1
	for i := range s.Items {
		if !s.cellRect(i).Contains(m.X, m.Y) {
			continue
		}
		s.hover = i
		if m.Pressed {
			m.Consumed = true
		}
		if m.Released && i != s.Selected {
			s.Selected = i
			if s.OnChange != nil {
				s.OnChange(i)
			}
			if s.OnChangeValue != nil {
				s.OnChangeValue(s.Items[i].Value)
			}
		}
	}
}
