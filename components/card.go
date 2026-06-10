package components

import (
	"github.com/mirzakhany/yoga"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
	"github.com/mirzakhany/yoga/theme"
)

// CardVariant selects the card's visual elevation level.
type CardVariant int

const (
	// CardFlat renders a surface-colored card with a subtle border and no
	// shadow — good for inline content grouped on a contrasting background.
	CardFlat CardVariant = iota

	// CardRaised adds a small shadow (ShadowSm) — the default for most cards.
	CardRaised

	// CardElevated adds a medium shadow (ShadowMd) — use for floating panels
	// or cards that need to stand out more prominently.
	CardElevated
)

// Card is a surfaced container with optional title, subtitle, and body content.
type Card struct {
	El       *layout.Element
	Title    string
	Subtitle string
	Body     *layout.Element
	variant  CardVariant
}

// NewCard builds a raised card (CardRaised) wrapping optional body content.
func NewCard(title, subtitle string, body *layout.Element) *Card {
	return newCard(title, subtitle, body, CardRaised)
}

func newCard(title, subtitle string, body *layout.Element, variant CardVariant) *Card {
	th := theme.Current()
	c := &Card{Title: title, Subtitle: subtitle, Body: body, variant: variant}
	pad := th.Spacing.L
	var children []*layout.Element
	if title != "" || subtitle != "" {
		header := layout.New(layout.Box().H(c.headerHeight(pad)).PaddingXY(pad, pad))
		header.Paint = func(dl *render.DrawList, text *shape.Engine) {
			c.paintHeader(dl, text, header.Frame, pad)
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

func (c *Card) headerHeight(pad float32) float32 {
	th := theme.Current()
	inner := float32(0)
	if c.Title != "" {
		_, lh := yoga.Text().MeasureAt(c.Title, th.Typography.Subtitle.Size)
		inner += lh + th.Spacing.XS
	}
	if c.Subtitle != "" {
		_, lh := yoga.Text().MeasureAt(c.Subtitle, th.Typography.Caption.Size)
		inner += lh
	}
	return inner + 2*pad
}

func (c *Card) paintHeader(dl *render.DrawList, text *shape.Engine, f render.Rect, pad float32) {
	th := theme.Current()
	x := f.X + pad
	y := f.Y + pad
	if c.Title != "" {
		style := th.Typography.Subtitle
		text.DrawStringTopAt(dl, c.Title, x, y, th.Foreground, style.Size)
		_, lh := text.MeasureAt(c.Title, style.Size)
		y += lh + th.Spacing.XS
	}
	if c.Subtitle != "" {
		style := th.Typography.Caption
		text.DrawStringTopAt(dl, c.Subtitle, x, y, th.ForegroundMuted, style.Size)
	}
}

func (c *Card) paint(dl *render.DrawList, _ *shape.Engine) {
	th := theme.Current()
	r := th.Radius.Large
	switch c.variant {
	case CardFlat:
		// No shadow; just surface + border.
	case CardElevated:
		drawElevationShadow(dl, c.El.Frame, r, th.Elevation.ShadowMd)
	default: // CardRaised
		drawElevationShadow(dl, c.El.Frame, r, th.Elevation.ShadowSm)
	}
	dl.AddRoundedRectBorder(c.El.Frame, r, th.Stroke.Thin, th.Chrome, th.Border)
}

// ── Builder/modifier methods ─────────────────────────────────────────────────

// Flat removes the shadow so the card sits flush with its background.
func (c *Card) Flat() *Card { c.variant = CardFlat; return c }

// Raised adds a small drop shadow (ShadowSm). This is the default.
func (c *Card) Raised() *Card { c.variant = CardRaised; return c }

// Elevated adds a medium drop shadow (ShadowMd) for a more prominent float.
func (c *Card) Elevated() *Card { c.variant = CardElevated; return c }
