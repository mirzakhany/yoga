package ui

import (
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
)

// NewIcon builds a size x size icon element drawing the named sprite.
func NewIcon(name string, size float32, color render.Color) *layout.Element {
	el := layout.New(layout.Box().Size(size, size).FlexShrink(0))
	el.Paint = func(dl *render.DrawList, _ *shape.Engine) {
		if sheet := frameIcons(); sheet != nil {
			sheet.Draw(dl, name, el.Frame, color)
		}
	}
	return el
}
