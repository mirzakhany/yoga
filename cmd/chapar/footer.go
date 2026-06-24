package main

import (
	"github.com/mirzakhany/yoga/components"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/theme"
)

type Footer struct {
	root *layout.Element
}

func NewFooter() *Footer {
	return &Footer{}
}

func (f *Footer) Layout() *layout.Element {
	return layout.New(layout.Box().Direction(layout.Row).Gap(0),
		components.NewLabel("Footer", components.LabelBody).El,
	).Bg(theme.Current().Background)
}
