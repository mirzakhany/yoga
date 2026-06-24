package components

import (
	"github.com/mirzakhany/yoga"
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
	El        *layout.Element
	Message   string
	Variant   AlertVariant
	Dismiss   bool
	OnDismiss func()
}

// NewAlert builds an alert banner sized to its message.
func NewAlert(message string, variant AlertVariant) *Alert {
	th := theme.Current()
	eng := yoga.Text()
	a := &Alert{Message: message, Variant: variant}
	style := th.Typography.Body
	tw, lh := eng.MeasureAt(message, style.Size)
	padY := th.Spacing.S
	a.El = layout.New(layout.Box().
		H(lh+2*padY).
		PaddingXY(th.Spacing.M, padY).
		Min(tw+2*th.Spacing.M, 0))
	a.El.Paint = a.paint
	a.El.OnMouse = a.onMouse
	return a
}

func (a *Alert) accent() render.Color {
	th := theme.Current()
	switch a.Variant {
	case AlertWarning:
		return th.Warning
	case AlertError:
		return th.Error
	case AlertSuccess:
		return th.Success
	default:
		return th.Accent
	}
}

func (a *Alert) paint(dl *render.DrawList, text *shape.Engine) {
	th := theme.Current()
	f := a.El.Frame
	accent := a.accent()
	bg := accent
	bg.A = 0.15
	dl.AddRoundedRect(f, th.Radius.Small, bg)
	dl.AddRect(render.Rect{X: f.X, Y: f.Y, W: 3, H: f.H}, accent)
	style := th.Typography.Body
	_, lh := text.MeasureAt(a.Message, style.Size)
	dl.PushClip(f)
	text.DrawStringTopAt(dl, a.Message, f.X+th.Spacing.M, f.Y+(f.H-lh)/2, th.Foreground, style.Size)
	dl.PopClip()
}

func (a *Alert) onMouse(e *layout.Element, m *input.Mouse) {
	if a.Dismiss && m.Released && e.Frame.Contains(m.X, m.Y) {
		if a.OnDismiss != nil {
			a.OnDismiss()
		}
	}
}
