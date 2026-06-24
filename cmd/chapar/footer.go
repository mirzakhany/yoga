package main

import (
	"github.com/mirzakhany/yoga/components"
	"github.com/mirzakhany/yoga/theme"
	"github.com/mirzakhany/yoga/ui"
)

type Footer struct {
	label *components.Label
}

func NewFooter() *Footer {
	return &Footer{label: components.NewLabel("Footer", components.LabelBody)}
}

func (f *Footer) Layout(c *ui.Ctx) *ui.Element {
	return ui.HStack(f.label.Layout(c)).Bg(theme.Current().Background)
}
