package ui

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
	Segments []BreadcrumbSegment
	hover    int
}

// NewBreadcrumb builds a breadcrumb bar.
func NewBreadcrumb(segments []BreadcrumbSegment) *Breadcrumb {
	th := theme.Current()
	b := &Breadcrumb{Segments: segments, hover: -1}
	b.El = layout.New(layout.Box().H(th.Metrics.ControlHeight))
	b.El.Paint = b.paint
	b.El.OnMouse = b.onMouse
	return b
}

func (b *Breadcrumb) layout() []struct{ x, w float32 } {
	th := theme.Current()
	f := b.El.Frame
	style := th.Typography.Body
	chevW := th.Metrics.TreeChevronSize
	gap := th.Spacing.S
	out := make([]struct{ x, w float32 }, len(b.Segments))
	x := f.X + th.Spacing.S
	for i, seg := range b.Segments {
		tw, _ := frameText().MeasureAt(seg.Label, style.Size)
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
	th := theme.Current()
	f := b.El.Frame
	locs := b.layout()
	style := th.Typography.Body
	chevSz := th.Metrics.TreeChevronSize
	dl.PushClip(f)
	defer dl.PopClip()
	for i, seg := range b.Segments {
		loc := locs[i]
		col := th.ForegroundMuted
		if i == len(b.Segments)-1 {
			col = th.Foreground
		} else if i == b.hover {
			col = th.Accent
		}
		_, lh := text.MeasureAt(seg.Label, style.Size)
		text.DrawStringTopAt(dl, seg.Label, loc.x, f.Y+(f.H-lh)/2, col, style.Size)
		if i < len(b.Segments)-1 {
			tw, _ := text.MeasureAt(seg.Label, style.Size)
			cx := loc.x + tw + th.Spacing.S
			cr := render.Rect{X: cx, Y: f.Y + (f.H-chevSz)/2, W: chevSz, H: chevSz}
			frameIcons().Draw(dl, "chevron_right", cr, th.ForegroundSubtle)
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
