package components

import (
	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
	"github.com/mirzakhany/yoga/theme"
)

// AlertVariant selects alert severity styling.
type AlertVariant int

const (
	AlertInfo AlertVariant = iota
	AlertWarning
	AlertError
	AlertSuccess
)

// Alert is an inline banner message.
type Alert struct {
	El       *layout.Element
	theme    *theme.Theme
	text     *shape.Engine
	Message  string
	Variant  AlertVariant
	Dismiss  bool
	OnDismiss func()
}

// NewAlert builds an alert banner sized to its message.
func NewAlert(eng *shape.Engine, th *theme.Theme, message string, variant AlertVariant) *Alert {
	a := &Alert{theme: th, text: eng, Message: message, Variant: variant}
	style := th.Typography.Body
	_, lh := eng.MeasureAt(message, style.Size)
	padY := th.Spacing.S
	a.El = layout.New(layout.Box().H(lh+2*padY).PaddingXY(th.Spacing.M, padY))
	a.El.Paint = a.paint
	a.El.OnMouse = a.onMouse
	return a
}

func (a *Alert) accent() render.Color {
	switch a.Variant {
	case AlertWarning:
		return a.theme.Warning
	case AlertError:
		return a.theme.Error
	case AlertSuccess:
		return a.theme.Success
	default:
		return a.theme.Accent
	}
}

func (a *Alert) paint(dl *render.DrawList, text *shape.Engine) {
	f := a.El.Frame
	accent := a.accent()
	bg := accent
	bg.A = 0.15
	dl.AddRoundedRect(f, a.theme.Radius.Small, bg)
	dl.AddRect(render.Rect{X: f.X, Y: f.Y, W: 3, H: f.H}, accent)
	style := a.theme.Typography.Body
	_, lh := text.MeasureAt(a.Message, style.Size)
	text.DrawStringTopAt(dl, a.Message, f.X+a.theme.Spacing.M, f.Y+(f.H-lh)/2, a.theme.Foreground, style.Size)
}

func (a *Alert) onMouse(e *layout.Element, m *input.Mouse) {
	if a.Dismiss && m.Released && e.Frame.Contains(m.X, m.Y) {
		if a.OnDismiss != nil {
			a.OnDismiss()
		}
	}
}
