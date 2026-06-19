package components

import (
	"github.com/mirzakhany/yoga"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
)

// ----------------------------------------------------------------------------
// Icon: renders a named sprite region from the atlas-backed sprite sheet,
// tinted by a color. The Element is fixed-size and the Paint hook stretches the
// sprite over its frame.
// ----------------------------------------------------------------------------

// NewIcon builds a size x size icon element drawing the named sprite.
func NewIcon(name string, size float32, color render.Color) *layout.Element {
	el := layout.New(layout.Box().Size(size, size).FlexShrink(0))
	el.Paint = func(dl *render.DrawList, _ *shape.Engine) {
		yoga.Icons().Draw(dl, name, el.Frame, color)
	}
	return el
}
