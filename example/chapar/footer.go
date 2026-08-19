package main

import (
	"github.com/mirzakhany/yoga/ui"
)

type Footer struct{}

func NewFooter() *Footer { return &Footer{} }

func (f *Footer) Layout(c *ui.Ctx) ui.View {
	return ui.Row(ui.Text("Footer")).Background(ui.TokenSurface).MarginLeft(10)
}
