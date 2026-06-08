package components

import (
	"time"

	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
	"github.com/mirzakhany/yoga/theme"
)

// ToastVariant selects toast styling.
type ToastVariant int

const (
	ToastInfo ToastVariant = iota
	ToastSuccess
	ToastWarning
	ToastError
)

type toastEntry struct {
	message string
	variant ToastVariant
	until   time.Time
}

// ToastHost manages a bottom-right toast stack.
type ToastHost struct {
	El     *layout.Element
	theme  *theme.Theme
	text   *shape.Engine
	toasts []toastEntry
	margin float32
	width  float32
}

// NewToastHost builds a toast overlay host. Mount El on app root.
func NewToastHost(eng *shape.Engine, th *theme.Theme) *ToastHost {
	t := &ToastHost{theme: th, text: eng, margin: 16, width: 280}
	t.El = layout.New(layout.Box())
	t.El.Overlay = true
	t.El.Paint = t.paint
	return t
}

// Show enqueues a toast that auto-dismisses after d.
func (t *ToastHost) Show(message string, variant ToastVariant, d time.Duration) {
	if d <= 0 {
		d = 3 * time.Second
	}
	t.toasts = append(t.toasts, toastEntry{message: message, variant: variant, until: time.Now().Add(d)})
}

// Position anchors the host to the bottom-right of the viewport.
func (t *ToastHost) Position(viewW, viewH float32) {
	t.El.Style = layout.Box().Absolute(0, 0).Size(viewW, viewH)
	t.El.ReapplyStyle()
}

func (t *ToastHost) prune() {
	now := time.Now()
	alive := t.toasts[:0]
	for _, e := range t.toasts {
		if now.Before(e.until) {
			alive = append(alive, e)
		}
	}
	t.toasts = alive
}

func (t *ToastHost) variantColor(v ToastVariant) render.Color {
	switch v {
	case ToastSuccess:
		return t.theme.Success
	case ToastWarning:
		return t.theme.Warning
	case ToastError:
		return t.theme.Error
	default:
		return t.theme.Accent
	}
}

func (t *ToastHost) paint(dl *render.DrawList, text *shape.Engine) {
	t.prune()
	if len(t.toasts) == 0 {
		return
	}
	f := t.El.Frame
	pad := t.theme.Spacing.S
	style := t.theme.Typography.Body
	itemH := style.LineHeight + 2*pad
	y := f.Y + f.H - t.margin
	for i := len(t.toasts) - 1; i >= 0; i-- {
		e := t.toasts[i]
		y -= itemH
		x := f.X + f.W - t.margin - t.width
		rect := render.Rect{X: x, Y: y, W: t.width, H: itemH}
		accent := t.variantColor(e.variant)
		bg := accent
		bg.A = 0.2
		r := t.theme.Radius.Medium
		drawElevationShadow(dl, rect, r, t.theme.Elevation.ShadowMd)
		dl.AddRoundedRectBorder(rect, r, t.theme.Stroke.Thin, t.theme.Chrome, accent)
		_, lh := text.MeasureAt(e.message, style.Size)
		text.DrawStringTopAt(dl, e.message, x+pad, y+(itemH-lh)/2, t.theme.Foreground, style.Size)
		y -= t.theme.Spacing.S
	}
}

// AnimationWait reports when a toast needs repaint for expiry.
func (t *ToastHost) AnimationWait() (time.Duration, bool) {
	t.prune()
	if len(t.toasts) == 0 {
		return 0, false
	}
	soonest := time.Until(t.toasts[0].until)
	for _, e := range t.toasts[1:] {
		d := time.Until(e.until)
		if d < soonest {
			soonest = d
		}
	}
	if soonest < 0 {
		soonest = 0
	}
	return soonest, true
}
