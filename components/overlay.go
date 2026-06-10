package components

import (
	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
)

// Scrim is a full-window dimmed backdrop for modals.
type Scrim struct {
	El   *layout.Element
	Open bool
}

// NewScrim builds a closed overlay scrim. Mount El on the app root.
func NewScrim() *Scrim {
	s := &Scrim{}
	s.El = layout.New(layout.Box())
	s.El.Overlay = true
	s.El.Paint = s.paint
	s.El.OnMouse = s.onMouse
	return s
}

// Show displays the scrim over the given viewport.
func (s *Scrim) Show(x, y, w, h float32) {
	s.Open = true
	s.El.Style = layout.Box().Absolute(x, y).Size(w, h)
	s.El.ReapplyStyle()
	// Seed the frame so the scrim paints and hit-tests correctly on the same
	// frame it is shown (Show may run after this frame's layout pass).
	s.El.Frame = render.Rect{X: x, Y: y, W: w, H: h}
}

// Hide closes the scrim.
func (s *Scrim) Hide() { s.Open = false }

func (s *Scrim) paint(dl *render.DrawList, _ *shape.Engine) {
	if !s.Open {
		return
	}
	c := render.RGBA8(0, 0, 0, 255)
	c.A = 0.45
	dl.AddRect(s.El.Frame, c)
}

func (s *Scrim) onMouse(e *layout.Element, m *input.Mouse) {
	if s.Open && e.Frame.Contains(m.X, m.Y) {
		m.Consumed = true
	}
}

