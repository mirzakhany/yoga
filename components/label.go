package components

import (
	"github.com/mirzakhany/yoga"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
	"github.com/mirzakhany/yoga/theme"
)

// LabelVariant selects typography and color for a Label.
type LabelVariant int

const (
	LabelBody LabelVariant = iota
	LabelCaption
	LabelMuted
	LabelStrong
	LabelSubtitle
	LabelTitle
)

// Label is static text using Yoga typography tokens.
type Label struct {
	El      *layout.Element
	content string
	variant LabelVariant
}

// NewLabel builds a label sized to its text.
func NewLabel(content string, variant LabelVariant) *Label {
	th := theme.Current()
	text := yoga.Text()
	l := &Label{content: content, variant: variant}
	style, _ := l.style()
	tw, lh := text.MeasureAt(content, style.Size)
	l.El = layout.New(layout.Box().Size(tw, lh))
	// Paint resolves style/color at draw time so variant changes via builder
	// methods take effect immediately without rebuilding the element.
	l.El.Paint = func(dl *render.DrawList, text *shape.Engine) {
		s, col := l.style()
		text.DrawStringTopAt(dl, l.content, l.El.Frame.X, l.El.Frame.Y, col, s.Size)
	}
	_ = th
	return l
}

func (l *Label) style() (theme.TypographyStyle, render.Color) {
	th := theme.Current()
	switch l.variant {
	case LabelCaption:
		return th.Typography.Caption, th.ForegroundMuted
	case LabelMuted:
		return th.Typography.Body, th.ForegroundMuted
	case LabelStrong:
		return th.Typography.BodyStrong, th.Foreground
	case LabelSubtitle:
		return th.Typography.Subtitle, th.Foreground
	case LabelTitle:
		return th.Typography.Title, th.Foreground
	default:
		return th.Typography.Body, th.Foreground
	}
}

// setVariant changes the variant and re-measures the element so the next
// layout pass sizes the frame correctly.
func (l *Label) setVariant(v LabelVariant) *Label {
	l.variant = v
	style, _ := l.style()
	tw, lh := yoga.Text().MeasureAt(l.content, style.Size)
	l.El.Style.Width = tw
	l.El.Style.Height = lh
	l.El.ReapplyStyle()
	return l
}

// SetText updates the label content and re-measures the element so the next
// layout pass sizes the frame to the new text.
func (l *Label) SetText(s string) {
	if s == l.content {
		return
	}
	l.content = s
	style, _ := l.style()
	tw, lh := yoga.Text().MeasureAt(s, style.Size)
	l.El.Style.Width = tw
	l.El.Style.Height = lh
	l.El.ReapplyStyle()
}

// ── Builder/modifier methods ─────────────────────────────────────────────────

// Body sets the variant to regular body text (14 px, normal weight).
func (l *Label) Body() *Label { return l.setVariant(LabelBody) }

// Caption sets the variant to small muted caption text (12 px).
func (l *Label) Caption() *Label { return l.setVariant(LabelCaption) }

// Muted sets the variant to muted body text (uses ForegroundMuted).
func (l *Label) Muted() *Label { return l.setVariant(LabelMuted) }

// Strong sets the variant to semibold body text.
func (l *Label) Strong() *Label { return l.setVariant(LabelStrong) }

// Subtitle sets the variant to subtitle text (16 px, semibold).
func (l *Label) Subtitle() *Label { return l.setVariant(LabelSubtitle) }

// Title sets the variant to title text (20 px, semibold).
func (l *Label) Title() *Label { return l.setVariant(LabelTitle) }
