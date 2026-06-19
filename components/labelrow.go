package components

import (
	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
	"github.com/mirzakhany/yoga/theme"
)

// NewLabelRow is a convenience fixed-height row that paints a single line of
// text, used for file-list entries.
func NewLabelRow(label string, height float32, onClick func()) *layout.Element {
	th := theme.Current()
	hovered := false
	padX := th.Spacing.MNudge
	el := layout.New(layout.Box().H(height).PaddingXY(padX, 0).JustifyContent(layout.JustifyCenter))
	el.Paint = func(dl *render.DrawList, eng *shape.Engine) {
		curTh := theme.Current()
		if hovered {
			dl.AddRect(el.Frame, curTh.ListHover)
		}
		style := curTh.Typography.Body
		_, lh := eng.MeasureAt(label, style.Size)
		ty := el.Frame.Y + (el.Frame.H-lh)/2
		eng.DrawStringTopAt(dl, label, el.Frame.X+padX, ty, curTh.Foreground, style.Size)
	}
	el.OnMouse = func(e *layout.Element, m *input.Mouse) {
		hovered = e.Frame.Contains(m.X, m.Y)
		if hovered && m.Released && onClick != nil {
			onClick()
		}
	}
	return el
}
