package components

import (
	"github.com/mirzakhany/yoga"
	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
	"github.com/mirzakhany/yoga/theme"
)

// ----------------------------------------------------------------------------
// Button: a bounding-box hit-tested control whose fill color reacts to hover
// and press state. It demonstrates the basic component pattern: a sized
// Element, a Paint hook that emits a background quad + centered label, and an
// OnMouse hook that maintains interaction state and fires OnClick.
// ----------------------------------------------------------------------------

type Button struct {
	El       *layout.Element
	label    string
	icon     string // optional leading icon name, empty for none
	hint     string // optional trailing keyboard-hint chip, e.g. "⌘↵"
	Variant  Variant
	hovered  bool
	pressed  bool
	focused  bool
	disabled bool
	OnClick  func()
}

// NewButton builds a button sized to its label.
func NewButton(label string) *Button {
	return newButtonVariant(label, VariantSecondary, nil)
}

// newButtonVariant builds a button with the given Fluent variant.
func newButtonVariant(label string, variant Variant, onClick func()) *Button {
	th := theme.Current()
	text := yoga.Text()
	style := th.Typography.Body
	tw, lineH := text.MeasureAt(label, style.Size)
	padX := th.Spacing.M
	padY := th.Spacing.SNudge
	h := lineH + 2*padY
	b := &Button{
		label:   label,
		Variant: variant,
		OnClick: onClick,
	}
	// Use H + Min instead of Size so the button has its natural minimum width
	// but can grow in a flex container (e.g. a full-width form button).
	// FlexShrink(0) keeps it from being squashed in tight rows.
	b.El = layout.New(layout.Box().H(h).Min(tw+2*padX, h).FlexShrink(0))
	b.El.Paint = b.paint
	b.El.OnMouse = b.onMouse
	return b
}

// NewButtonVariant builds a button with the given Fluent variant.
// Deprecated: use NewButton(label).Primary() etc. instead.
func NewButtonVariant(label string, variant Variant, onClick func()) *Button {
	return newButtonVariant(label, variant, onClick)
}

// ── Builder/modifier methods ─────────────────────────────────────────────────
// These return the receiver so calls can be chained directly off the
// constructor, SwiftUI-style:
//
//	btn := ui.Button("Save").Primary().Action(save)

// Primary sets the button to the primary (accent-filled) variant.
func (b *Button) Primary() *Button { b.Variant = VariantPrimary; return b }

// Secondary sets the button to the secondary (surface-filled, bordered) variant.
func (b *Button) Secondary() *Button { b.Variant = VariantSecondary; return b }

// Subtle sets the button to the subtle (transparent bg, hover fill) variant.
func (b *Button) Subtle() *Button { b.Variant = VariantSubtle; return b }

// Disabled marks or unmarks the button as non-interactive.
func (b *Button) Disabled(v bool) *Button { b.disabled = v; return b }

// Action sets the click handler, replacing any handler given at construction.
func (b *Button) Action(fn func()) *Button { b.OnClick = fn; return b }

// SetLabel updates the button's text in place. The element keeps its existing
// frame; callers wanting the button to resize to its new natural width should
// trigger a relayout afterwards.
func (b *Button) SetLabel(label string) { b.label = label; b.resize() }

// IconStart sets a leading icon (by sprite name) drawn before the label.
func (b *Button) IconStart(name string) *Button { b.icon = name; b.resize(); return b }

// Hint sets a trailing keyboard-hint chip drawn after the label (e.g. "⌘↵").
func (b *Button) Hint(s string) *Button { b.hint = s; b.resize(); return b }

// iconGap / hintGap separate the icon and hint from the label.
const (
	buttonIconGap = 8
	buttonHintGap = 8
)

// contentWidth measures the full intrinsic content row (icon + label + hint).
func (b *Button) contentWidth(text *shape.Engine) float32 {
	th := theme.Current()
	w, _ := text.MeasureAt(b.label, th.Typography.Body.Size)
	if b.icon != "" {
		w += th.Metrics.IconSizeSM + buttonIconGap
	}
	if b.hint != "" {
		w += buttonHintGap + b.hintChipWidth(text)
	}
	return w
}

// hintChipWidth returns the painted width of the hint chip.
func (b *Button) hintChipWidth(text *shape.Engine) float32 {
	if b.hint == "" {
		return 0
	}
	hw, _ := text.MeasureAt(b.hint, theme.Current().Typography.Caption.Size)
	return hw + 10 // 5px horizontal padding each side
}

// resize recomputes the button's minimum width from its current content.
func (b *Button) resize() {
	th := theme.Current()
	cw := b.contentWidth(yoga.Text())
	padX := th.Spacing.M
	min := cw + 2*padX
	b.El.Style = b.El.Style.Min(min, b.El.Style.MinHeight)
	b.El.ReapplyStyle()
}

// FillWidth removes the minimum-width constraint so the button stretches to
// fill its flex parent (equivalent to FlexGrow(1) on the element).
func (b *Button) FillWidth() *Button {
	b.El.Style = b.El.Style.FlexGrow(1)
	b.El.ReapplyStyle()
	return b
}

func (b *Button) buttonState() State {
	if b.disabled {
		return StateDisabled
	}
	switch {
	case b.pressed:
		return StatePressed
	case b.hovered:
		return StateHover
	default:
		return StateRest
	}
}

func (b *Button) paint(dl *render.DrawList, text *shape.Engine) {
	th := theme.Current()
	state := b.buttonState()
	bg := resolveBg(th, b.Variant, state)
	fg := resolveFg(th, b.Variant, state)
	r := th.Radius.Medium
	border := resolveButtonBorder(th, b.Variant, state)

	switch {
	case border.A > 0:
		// Secondary: draw fill + border in one call so the border sits on top
		// of the fill without alpha-blending issues.
		dl.AddRoundedRectBorder(b.El.Frame, r, th.Stroke.Thin, bg, border)
	case bg.A > 0:
		dl.AddRoundedRect(b.El.Frame, r, bg)
	}
	if b.focused {
		drawFocusRing(dl, b.El.Frame, bg, th)
	}

	f := b.El.Frame
	style := th.Typography.Body
	tw, lh := text.MeasureAt(b.label, style.Size)
	cw := b.contentWidth(text)
	x := f.X + (f.W-cw)/2
	cy := f.Y + f.H/2

	if b.icon != "" {
		isz := th.Metrics.IconSizeSM
		yoga.Icons().Draw(dl, b.icon, render.Rect{X: x, Y: cy - isz/2, W: isz, H: isz}, fg)
		x += isz + buttonIconGap
	}
	text.DrawStringTopAt(dl, b.label, x, f.Y+(f.H-lh)/2, fg, style.Size)
	x += tw
	if b.hint != "" {
		x += buttonHintGap
		chipW := b.hintChipWidth(text)
		hsz := th.Typography.Caption.Size
		hw, hh := text.MeasureAt(b.hint, hsz)
		chipH := hh + 2
		chip := render.Rect{X: x, Y: cy - chipH/2, W: chipW, H: chipH}
		chipBg := render.Color{R: fg.R, G: fg.G, B: fg.B, A: 0.16}
		chipFg := render.Color{R: fg.R, G: fg.G, B: fg.B, A: 0.62}
		dl.AddRoundedRect(chip, th.Radius.Small, chipBg)
		text.DrawStringTopAt(dl, b.hint, x+(chipW-hw)/2, cy-hh/2, chipFg, hsz)
	}
}

// Focus grants keyboard focus to the button.
func (b *Button) Focus() { b.focused = true }

// Blur removes keyboard focus from the button.
func (b *Button) Blur() { b.focused = false }

// Focused reports whether the button has keyboard focus.
func (b *Button) Focused() bool { return b.focused }

func (b *Button) HandleText(_ []rune)           {}
func (b *Button) HandleKeys(_ []input.KeyEvent) {}
func (b *Button) CapturesTab() bool             { return false }
func (b *Button) FocusOnClick() bool            { return true }
func (b *Button) FocusEl() *layout.Element      { return b.El }

func (b *Button) onMouse(e *layout.Element, m *input.Mouse) {
	if b.disabled {
		return
	}
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
