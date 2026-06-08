package components

import (
	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
	"github.com/mirzakhany/yoga/theme"
)

// ScrollView is a clipped vertical scroll container for tall content.
type ScrollView struct {
	El      *layout.Element
	theme   *theme.Theme
	Content *layout.Element
	vbar    *Scrollbar
	scrollY    float32
	contentH   float32
}

// NewScrollView wraps content in a vertically scrollable viewport.
func NewScrollView(th *theme.Theme, content *layout.Element) *ScrollView {
	sv := &ScrollView{theme: th, Content: content}
	if content != nil {
		// Keep scroll content at its intrinsic height; default flex shrink would
		// compress tall pages into the viewport and stack children on top of each other.
		content.Style.Shrink = 0
	}
	bar := th.Metrics.ScrollbarSize
	sv.vbar = NewScrollbarAxis(th, Vertical, &sv.scrollY, &sv.contentH, bar)
	sv.El = layout.New(layout.Box().FlexGrow(1), content, sv.vbar.El)
	sv.El.Clip = true
	sv.El.Paint = sv.paint
	sv.El.OnMouse = sv.onMouse
	sv.El.ScrollOffset = 0
	return sv
}

func (sv *ScrollView) contentHeight() float32 {
	if sv.Content == nil {
		return 0
	}
	sv.contentH = sv.Content.Frame.H
	return sv.contentH
}

func (sv *ScrollView) viewport() render.Rect {
	f := sv.El.Frame
	bar := sv.theme.Metrics.ScrollbarSize
	w := f.W
	if sv.contentHeight() > f.H {
		w = f.W - bar
	}
	return render.Rect{X: f.X, Y: f.Y, W: w, H: f.H}
}

func (sv *ScrollView) syncScroll() {
	ch := sv.contentHeight()
	sv.El.ScrollOffset = sv.scrollY
	if sv.vbar.ContentHeight != nil {
		*sv.vbar.ContentHeight = ch
	}
	f := sv.El.Frame
	vShow := ch > f.H
	if vShow {
		sv.vbar.El.Style = layout.Box().W(sv.theme.Metrics.ScrollbarSize).AbsTop(0).AbsRight(0).AbsBottom(0)
		sv.vbar.El.ReapplyStyle()
	}
	maxOff := f32max(0, ch-f.H)
	sv.scrollY = clampf(sv.scrollY, 0, maxOff)
	sv.El.ScrollOffset = sv.scrollY
}

func (sv *ScrollView) paint(dl *render.DrawList, _ *shape.Engine) {
	dl.AddRect(sv.El.Frame, sv.theme.Surface)
}

func (sv *ScrollView) onMouse(e *layout.Element, m *input.Mouse) {
	if sv.vbar.El.Frame.Contains(m.X, m.Y) {
		return
	}
}

// Update drives scroll wheel and thumb drag. Call after layout each frame.
func (sv *ScrollView) Update(m *input.Mouse) {
	sv.syncScroll()
	sv.vbar.Update(m, sv.viewport())
	sv.syncScroll()
}
