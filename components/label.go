package components

import (
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
)

// Label is static text using Yoga typography tokens.
type Label struct {
	El      *layout.Element
	theme   *theme.Theme
	text    *shape.Engine
	content string
	variant LabelVariant
}

// NewLabel builds a label sized to its text.
func NewLabel(eng *shape.Engine, th *theme.Theme, content string, variant LabelVariant) *Label {
	l := &Label{theme: th, text: eng, content: content, variant: variant}
	style, col := l.style()
	tw, lh := eng.MeasureAt(content, style.Size)
	l.El = layout.New(layout.Box().Size(tw, lh))
	l.El.Paint = func(dl *render.DrawList, text *shape.Engine) {
		text.DrawStringTopAt(dl, l.content, l.El.Frame.X, l.El.Frame.Y, col, style.Size)
	}
	return l
}

func (l *Label) style() (theme.TypographyStyle, render.Color) {
	switch l.variant {
	case LabelCaption:
		return l.theme.Typography.Caption, l.theme.ForegroundMuted
	case LabelMuted:
		return l.theme.Typography.Body, l.theme.ForegroundMuted
	case LabelStrong:
		return l.theme.Typography.BodyStrong, l.theme.Foreground
	case LabelSubtitle:
		return l.theme.Typography.Subtitle, l.theme.Foreground
	default:
		return l.theme.Typography.Body, l.theme.Foreground
	}
}

// SetText updates the label content (caller should relayout).
func (l *Label) SetText(s string) { l.content = s }
