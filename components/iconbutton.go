package components

import (
	"github.com/mirzakhany/yoga"
	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
	"github.com/mirzakhany/yoga/theme"
)

// IconButton is a square icon-only button.
type IconButton struct {
	El      *layout.Element
	icon    string
	size    float32
	Variant Variant
	hovered bool
	pressed bool
	focused bool
	OnClick func()
}

// NewIconButton builds an icon button of the given logical size.
func NewIconButton(icon string, size float32, onClick func()) *IconButton {
	return NewIconButtonVariant(icon, size, VariantSubtle, onClick)
}

// NewIconButtonVariant builds an icon button with a visual variant.
func NewIconButtonVariant(icon string, size float32, variant Variant, onClick func()) *IconButton {
	b := &IconButton{
		icon: icon, size: size,
		Variant: variant, OnClick: onClick,
	}
	b.El = layout.New(layout.Box().Size(size, size).FlexShrink(0))
	b.El.Paint = b.paint
	b.El.OnMouse = b.onMouse
	return b
}

func (b *IconButton) state() State {
	switch {
	case b.pressed:
		return StatePressed
	case b.hovered:
		return StateHover
	default:
		return StateRest
	}
}

func (b *IconButton) paint(dl *render.DrawList, _ *shape.Engine) {
	th := theme.Current()
	bg := resolveBg(th, b.Variant, b.state())
	r := th.Radius.Medium
	if bg.A > 0 {
		dl.AddRoundedRect(b.El.Frame, r, bg)
	}
	if b.focused {
		drawFocusRing(dl, b.El.Frame, bg, th)
	}
	pad := th.Spacing.XS
	inner := render.Rect{
		X: b.El.Frame.X + pad, Y: b.El.Frame.Y + pad,
		W: b.El.Frame.W - 2*pad, H: b.El.Frame.H - 2*pad,
	}
	col := resolveFg(th, b.Variant, b.state())
	yoga.Icons().Draw(dl, b.icon, inner, col)
}

func (b *IconButton) onMouse(e *layout.Element, m *input.Mouse) {
	inside := e.Frame.Contains(m.X, m.Y)
	b.hovered = inside
	if inside && m.Pressed {
		b.pressed = true
		m.Consumed = true
	}
	if m.Released {
		if b.pressed && inside && b.OnClick != nil {
			b.OnClick()
		}
		b.pressed = false
	}
}

func (b *IconButton) Focus()   { b.focused = true }
func (b *IconButton) Blur()    { b.focused = false }
func (b *IconButton) Focused() bool { return b.focused }

// Action sets the click handler.
func (b *IconButton) Action(fn func()) *IconButton { b.OnClick = fn; return b }

// Primary sets the icon button to the primary (accent-filled) variant.
func (b *IconButton) Primary() *IconButton { b.Variant = VariantPrimary; return b }

// Secondary sets the icon button to the secondary variant.
func (b *IconButton) Secondary() *IconButton { b.Variant = VariantSecondary; return b }
