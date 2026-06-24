package components

import (
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
)

func HLine(w layout.Px, color render.Color) *layout.Element {
	return layout.New(layout.Box().H(w)).Bg(color)
}

func VLine(h layout.Px, color render.Color) *layout.Element {
	return layout.New(layout.Box().W(h)).Bg(color)
}
