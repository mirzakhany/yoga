package components

import (
	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
	"github.com/mirzakhany/yoga/theme"
)

// BreadcrumbSegment is one clickable crumb.
type BreadcrumbSegment struct {
	Label    string
	OnSelect func()
}

// Breadcrumb is a horizontal navigation trail.
type Breadcrumb struct {
	El       *layout.Element
	theme    *theme.Theme
	text     *shape.Engine
	sheet    *render.SpriteSheet
	Segments []BreadcrumbSegment
	hover    int
}

// NewBreadcrumb builds a breadcrumb bar.
func NewBreadcrumb(eng *shape.Engine, th *theme.Theme, sheet *render.SpriteSheet, segments []BreadcrumbSegment) *Breadcrumb {
	b := &Breadcrumb{theme: th, text: eng, sheet: sheet, Segments: segments, hover: -1}
	b.El = layout.New(layout.Box().H(th.Metrics.ControlHeight))
	b.El.Paint = b.paint
	b.El.OnMouse = b.onMouse
	return b
}

func (b *Breadcrumb) layout() []struct{ x, w float32 } {
	f := b.El.Frame
	style := b.theme.Typography.Body
	chevW := b.theme.Metrics.TreeChevronSize
	gap := b.theme.Spacing.S
	out := make([]struct{ x, w float32 }, len(b.Segments))
	x := f.X + b.theme.Spacing.S
	for i, seg := range b.Segments {
		tw, _ := b.text.MeasureAt(seg.Label, style.Size)
		w := tw
		if i < len(b.Segments)-1 {
			w += gap + chevW
		}
		out[i] = struct{ x, w float32 }{x: x, w: w}
		x += w + gap
	}
	return out
}

func (b *Breadcrumb) paint(dl *render.DrawList, text *shape.Engine) {
	f := b.El.Frame
	locs := b.layout()
	style := b.theme.Typography.Body
	chevSz := b.theme.Metrics.TreeChevronSize
	for i, seg := range b.Segments {
		loc := locs[i]
		col := b.theme.ForegroundMuted
		if i == len(b.Segments)-1 {
			col = b.theme.Foreground
		} else if i == b.hover {
			col = b.theme.Accent
		}
		_, lh := text.MeasureAt(seg.Label, style.Size)
		text.DrawStringTopAt(dl, seg.Label, loc.x, f.Y+(f.H-lh)/2, col, style.Size)
		if i < len(b.Segments)-1 {
			tw, _ := text.MeasureAt(seg.Label, style.Size)
			cx := loc.x + tw + b.theme.Spacing.XS
			cr := render.Rect{X: cx, Y: f.Y + (f.H-chevSz)/2, W: chevSz, H: chevSz}
			b.sheet.Draw(dl, "chevron_right", cr, b.theme.ForegroundSubtle)
		}
	}
}

func (b *Breadcrumb) onMouse(e *layout.Element, m *input.Mouse) {
	b.hover = -1
	if !e.Frame.Contains(m.X, m.Y) {
		return
	}
	locs := b.layout()
	for i, loc := range locs {
		if m.X >= loc.x && m.X <= loc.x+loc.w {
			b.hover = i
			if m.Released && i < len(b.Segments)-1 && b.Segments[i].OnSelect != nil {
				b.Segments[i].OnSelect()
			}
			return
		}
	}
}
