package ui

import (
	"math"
	"time"

	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
	"github.com/mirzakhany/yoga/theme"
)

// Spinner is an indeterminate loading indicator.
type Spinner struct {
	El    *layout.Element
	size  float32
	angle float32
	start time.Time
}

// NewSpinner builds a spinner of the given diameter.
func NewSpinner(size float32) *Spinner {
	s := &Spinner{size: size, start: time.Now()}
	s.El = layout.New(layout.Box().Size(size, size).FlexShrink(0))
	s.El.Paint = s.paint
	return s
}

func (s *Spinner) paint(dl *render.DrawList, _ *shape.Engine) {
	th := theme.Current()
	f := s.El.Frame
	cx := f.X + f.W/2
	cy := f.Y + f.H/2
	r := f.W/2 - 2
	segments := 12
	for i := 0; i < segments; i++ {
		t := float32(i) / float32(segments)
		a := s.angle + t*2*math.Pi
		alpha := 0.2 + 0.8*t
		col := th.Accent
		col.A *= alpha
		x0 := cx + r*float32(math.Cos(float64(a)))
		y0 := cy + r*float32(math.Sin(float64(a)))
		x1 := cx + (r-4)*float32(math.Cos(float64(a)))
		y1 := cy + (r-4)*float32(math.Sin(float64(a)))
		dl.AddRect(render.Rect{X: x0 - 1, Y: y0 - 1, W: 2, H: 2}, col)
		_ = x1
		_ = y1
	}
	// arc tick marks around the ring
	for i := 0; i < 8; i++ {
		t := float32(i) / 8
		a := s.angle + t*2*math.Pi
		alpha := 0.15 + 0.85*(1-t)
		col := th.Accent
		col.A *= alpha
		outer := r
		inner := r - 3
		x0 := cx + inner*float32(math.Cos(float64(a)))
		y0 := cy + inner*float32(math.Sin(float64(a)))
		x1 := cx + outer*float32(math.Cos(float64(a)))
		y1 := cy + outer*float32(math.Sin(float64(a)))
		mx := (x0 + x1) / 2
		my := (y0 + y1) / 2
		dl.AddRect(render.Rect{X: mx - 1.5, Y: my - 1.5, W: 3, H: 3}, col)
	}
}

// Update advances rotation. Call each frame.
func (s *Spinner) Update() {
	s.angle += 0.12
}

// Layout is the new ui.View entry point: it advances the animation and
// self-schedules the next repaint via the frame context, so callers no longer
// implement AnimationWait at the app level.
func (s *Spinner) Layout(c *Ctx) *layout.Element {
	s.Update()
	c.Animate(16 * time.Millisecond)
	return s.El
}
