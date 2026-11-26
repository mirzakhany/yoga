package ui

import (
	"testing"

	"github.com/yogaui/yoga/input"
	"github.com/yogaui/yoga/render"
	"github.com/yogaui/yoga/shape"
)

func newTestCtx(t *testing.T) *Ctx {
	t.Helper()
	text, err := shape.NewEngine(1, false)
	if err != nil {
		t.Skip(err)
	}
	SetFrameResources(text, render.NewSpriteSheet(text.Atlas), &input.MemClipboard{})
	c := New(text, NewFocusScope(), nil)
	c.SetIcons(render.NewSpriteSheet(text.Atlas))
	c.SetClipboard(&input.MemClipboard{})
	return c
}
