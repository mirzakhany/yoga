package components

import (
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
	"github.com/mirzakhany/yoga/theme"
)

// Card is a surfaced container with optional title and child content.
type Card struct {
	El       *layout.Element
	theme    *theme.Theme
	text     *shape.Engine
	Title    string
	Subtitle string
	Body     *layout.Element
}

// NewCard builds a card wrapping body content.
func NewCard(eng *shape.Engine, th *theme.Theme, title, subtitle string, body *layout.Element) *Card {
	c := &Card{theme: th, text: eng, Title: title, Subtitle: subtitle, Body: body}
	pad := th.Spacing.M
	var children []*layout.Element
	if title != "" || subtitle != "" {
		header := layout.New(layout.Box().H(c.headerHeight()).PaddingAll(pad))
		header.Paint = func(dl *render.DrawList, text *shape.Engine) {
			c.paintHeader(dl, text, header.Frame)
		}
		children = append(children, header)
	}
	if body != nil {
		wrapped := layout.New(layout.Box().PaddingXY(pad, pad), body)
		children = append(children, wrapped)
	}
	c.El = layout.New(layout.Box().Direction(layout.Column).Gap(th.Spacing.XS), children...)
	c.El.Paint = c.paint
	return c
}

func (c *Card) headerHeight() float32 {
	pad := c.theme.Spacing.M
	inner := float32(0)
	if c.Title != "" {
		_, lh := c.text.MeasureAt(c.Title, c.theme.Typography.Subtitle.Size)
		inner += lh + c.theme.Spacing.XS
	}
	if c.Subtitle != "" {
		_, lh := c.text.MeasureAt(c.Subtitle, c.theme.Typography.Caption.Size)
		inner += lh
	}
	return inner + 2*pad
}

func (c *Card) paintHeader(dl *render.DrawList, text *shape.Engine, f render.Rect) {
	pad := c.theme.Spacing.M
	x := f.X + pad
	y := f.Y + pad
	if c.Title != "" {
		style := c.theme.Typography.Subtitle
		text.DrawStringTopAt(dl, c.Title, x, y, c.theme.Foreground, style.Size)
		_, lh := text.MeasureAt(c.Title, style.Size)
		y += lh + c.theme.Spacing.XS
	}
	if c.Subtitle != "" {
		style := c.theme.Typography.Caption
		text.DrawStringTopAt(dl, c.Subtitle, x, y, c.theme.ForegroundMuted, style.Size)
	}
}

func (c *Card) paint(dl *render.DrawList, _ *shape.Engine) {
	r := c.theme.Radius.Medium
	drawElevationShadow(dl, c.El.Frame, r, c.theme.Elevation.ShadowSm)
	dl.AddRoundedRectBorder(c.El.Frame, r, c.theme.Stroke.Thin, c.theme.Chrome, c.theme.Border)
}
