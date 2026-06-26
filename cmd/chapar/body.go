package main

import (
	"github.com/mirzakhany/yoga/components"
	"github.com/mirzakhany/yoga/ui"
)

type Body struct {
	label *components.Label
}

func NewBody() *Body {
	return &Body{label: components.NewLabel("Body", components.LabelBody)}
}

func (b *Body) Layout(c *ui.Ctx) *ui.Element {
	return b.label.Layout(c)
}

func (b *Body) SetText(text string) {
	b.label.SetText(text)
}
