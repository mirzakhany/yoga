package main

import (
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/theme"
)

func HLine() *layout.Element {
	th := theme.Current()
	return layout.New(layout.Box().H(th.Stroke.Thin)).BgPtr(&th.Border)
}
