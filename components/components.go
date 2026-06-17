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
	El      *layout.Element
	label   string
	icon    string // optional leading icon name, empty for none
	hint    string // optional trailing keyboard-hint chip, e.g. "⌘↵"
	Variant Variant
	hovered bool
	pressed bool
	focused bool
	disabled bool
	OnClick func()
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

func (b *Button) HandleText(_ []rune)              {}
func (b *Button) HandleKeys(_ []input.KeyEvent)  {}
func (b *Button) CapturesTab() bool              { return false }
func (b *Button) FocusOnClick() bool             { return true }
func (b *Button) FocusEl() *layout.Element       { return b.El }

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

// ----------------------------------------------------------------------------
// List: a thin helper that arranges children as a row or column stack using
// Yoga's flex flow. It is intentionally just a styled container — the power is
// in the layout engine.
// ----------------------------------------------------------------------------

// NewList returns a container that stacks its children along dir.
func NewList(dir layout.FlexDirection, items ...*layout.Element) *layout.Element {
	return layout.New(layout.Box().Direction(dir), items...)
}

// NewLabelRow is a convenience fixed-height row that paints a single line of
// text, used for file-list entries.
func NewLabelRow(label string, height float32, onClick func()) *layout.Element {
	th := theme.Current()
	hovered := false
	padX := th.Spacing.MNudge
	el := layout.New(layout.Box().H(height).PaddingXY(padX, 0).JustifyContent(layout.JustifyCenter))
	el.Paint = func(dl *render.DrawList, eng *shape.Engine) {
		curTh := theme.Current()
		if hovered {
			dl.AddRect(el.Frame, curTh.ListHover)
		}
		style := curTh.Typography.Body
		_, lh := eng.MeasureAt(label, style.Size)
		ty := el.Frame.Y + (el.Frame.H-lh)/2
		eng.DrawStringTopAt(dl, label, el.Frame.X+padX, ty, curTh.Foreground, style.Size)
	}
	el.OnMouse = func(e *layout.Element, m *input.Mouse) {
		hovered = e.Frame.Contains(m.X, m.Y)
		if hovered && m.Released && onClick != nil {
			onClick()
		}
	}
	return el
}

// ----------------------------------------------------------------------------
// Scrollbar: tracks content vs. viewport height and converts thumb drags / wheel
// movement into a pixel scroll offset. It does not move geometry itself; it
// drives a *float32 offset that the owner (e.g. the editor, or a content
// container's ScrollOffset) reads.
// ----------------------------------------------------------------------------

// Axis selects a scrollbar's orientation.
type Axis int

const (
	Vertical Axis = iota
	Horizontal
)

type Scrollbar struct {
	El   *layout.Element // visual track: an absolute strip on one edge
	axis Axis
	Offset        *float32 // owner-owned scroll position in pixels (along the axis)
	ContentHeight *float32 // owner-owned total content length along the axis, in pixels

	dragging bool
	hovered  bool
	grab     float32 // cursor-to-thumb offset captured at drag start (along axis)
}

// NewScrollbar creates a vertical scrollbar bound to the given offset/content
// pointers. width is the track thickness in pixels.
func NewScrollbar(offset, contentHeight *float32, width float32) *Scrollbar {
	return NewScrollbarAxis(Vertical, offset, contentHeight, width)
}

// NewScrollbarAxis creates a scrollbar for the given axis. The track pins itself
// to the right edge (vertical) or bottom edge (horizontal) of its parent;
// thickness is the cross-axis size in pixels. Offset/content are measured along
// the axis (height for vertical, width for horizontal).
func NewScrollbarAxis(axis Axis, offset, content *float32, thickness float32) *Scrollbar {
	s := &Scrollbar{axis: axis, Offset: offset, ContentHeight: content}
	if axis == Horizontal {
		// Strip across the bottom edge (leaving room for a vertical bar's corner
		// is the owner's concern).
		s.El = layout.New(layout.Box().H(thickness).AbsLeft(0).AbsRight(0).AbsBottom(0))
	} else {
		s.El = layout.New(layout.Box().W(thickness).AbsTop(0).AbsRight(0).AbsBottom(0))
	}
	s.El.Paint = s.paint
	s.El.OnMouse = s.onMouse
	return s
}

// scrollable reports whether content exceeds the track along the scroll axis.
func (s *Scrollbar) scrollable() bool {
	return *s.ContentHeight > s.trackLen()
}

// maxOffset is the maximum scroll offset in pixels.
func (s *Scrollbar) maxOffset() float32 {
	return f32max(0, *s.ContentHeight-s.trackLen())
}

// trackLen is the scrollbar's length along its scroll axis.
func (s *Scrollbar) trackLen() float32 {
	if s.axis == Horizontal {
		return s.El.Frame.W
	}
	return s.El.Frame.H
}

// thumb computes the thumb rectangle from the current track frame and offset.
func (s *Scrollbar) thumb() render.Rect {
	th := theme.Current()
	track := s.El.Frame
	ch := *s.ContentHeight
	along := s.trackLen()
	maxOff := f32max(0, ch-along)

	thumbLen := along
	if ch > along && ch > 0 {
		thumbLen = along * along / ch
	}
	thumbLen = clampf(thumbLen, th.Metrics.ScrollbarMinThumb, along)

	if s.axis == Horizontal {
		tx := track.X
		if maxOff > 0 {
			tx = track.X + (along-thumbLen)*(*s.Offset/maxOff)
		}
		return render.Rect{X: tx, Y: track.Y, W: thumbLen, H: track.H}
	}
	ty := track.Y
	if maxOff > 0 {
		ty = track.Y + (along-thumbLen)*(*s.Offset/maxOff)
	}
	return render.Rect{X: track.X, Y: ty, W: track.W, H: thumbLen}
}

// thumbVisual returns the thumb rectangle inset inside the track for drawing and
// hit-testing.
func (s *Scrollbar) thumbVisual() render.Rect {
	th := theme.Current()
	t := s.thumb()
	inset := th.Metrics.ScrollbarThumbInset
	if s.axis == Horizontal {
		return render.Rect{X: t.X, Y: t.Y + inset, W: t.W, H: t.H - 2*inset}
	}
	return render.Rect{X: t.X + inset, Y: t.Y, W: t.W - 2*inset, H: t.H}
}

// setOffsetFromPointer maps a pointer position along the track to a scroll offset.
func (s *Scrollbar) setOffsetFromPointer(px, py float32) {
	track := s.El.Frame
	t := s.thumb()
	along := s.trackLen()
	maxOff := s.maxOffset()
	if maxOff <= 0 {
		return
	}
	if s.axis == Horizontal {
		travel := along - t.W
		if travel <= 0 {
			return
		}
		*s.Offset = (px - s.grab - track.X) / travel * maxOff
	} else {
		travel := along - t.H
		if travel <= 0 {
			return
		}
		*s.Offset = (py - s.grab - track.Y) / travel * maxOff
	}
	*s.Offset = clampf(*s.Offset, 0, maxOff)
}

func (s *Scrollbar) onMouse(el *layout.Element, m *input.Mouse) {
	s.hovered = false
	if !s.scrollable() || !el.Frame.Contains(m.X, m.Y) {
		return
	}
	tv := s.thumbVisual()
	s.hovered = tv.Contains(m.X, m.Y)

	if m.Pressed {
		m.Consumed = true
		if tv.Contains(m.X, m.Y) {
			s.dragging = true
			if s.axis == Horizontal {
				s.grab = m.X - tv.X
			} else {
				s.grab = m.Y - tv.Y
			}
		} else {
			// Click on track: jump so the thumb center moves toward the click.
			raw := s.thumb()
			if s.axis == Horizontal {
				s.grab = raw.W / 2
				s.setOffsetFromPointer(m.X, m.Y)
			} else {
				s.grab = raw.H / 2
				s.setOffsetFromPointer(m.X, m.Y)
			}
		}
	}
	if !m.Down {
		s.dragging = false
	}
	if s.dragging && m.Down {
		s.setOffsetFromPointer(m.X, m.Y)
		m.Consumed = true
	}
}

// Update processes wheel and drag input. area is the region (usually the
// viewport) over which the wheel should scroll.
func (s *Scrollbar) Update(m *input.Mouse, area render.Rect) {
	if s.scrollable() && area.Contains(m.X, m.Y) {
		if s.axis == Horizontal {
			if m.ScrollX != 0 {
				*s.Offset -= m.ScrollX * 3 * 14
				m.ScrollX = 0
			}
		} else if m.ScrollY != 0 {
			*s.Offset -= m.ScrollY * 3 * 14 // ~3 lines per wheel notch
			m.ScrollY = 0
		}
	}

	if s.dragging && m.Down {
		s.setOffsetFromPointer(m.X, m.Y)
	}
	if !m.Down {
		s.dragging = false
	}
	*s.Offset = clampf(*s.Offset, 0, s.maxOffset())
}

func (s *Scrollbar) paint(dl *render.DrawList, _ *shape.Engine) {
	if !s.scrollable() {
		return
	}
	th := theme.Current()
	dl.AddRect(s.El.Frame, th.ScrollTrack)
	col := th.ScrollThumb
	if s.dragging || s.hovered {
		col = th.ScrollThumbHover
	}
	dl.AddRect(s.thumbVisual(), col)
}

// ----------------------------------------------------------------------------
// Icon: renders a named sprite region from the atlas-backed sprite sheet,
// tinted by a color. The Element is fixed-size and the Paint hook stretches the
// sprite over its frame.
// ----------------------------------------------------------------------------

// NewIcon builds a size x size icon element drawing the named sprite.
func NewIcon(name string, size float32, color render.Color) *layout.Element {
	el := layout.New(layout.Box().Size(size, size).FlexShrink(0))
	el.Paint = func(dl *render.DrawList, _ *shape.Engine) {
		yoga.Icons().Draw(dl, name, el.Frame, color)
	}
	return el
}

// ----------------------------------------------------------------------------
// Menu + Dropdown: an absolutely-positioned overlay that is painted and
// hit-tested on top of the normal tree (Z-axis ordering). The menu paints its
// own item rows rather than nesting child elements, which keeps overlay
// geometry self-contained.
// ----------------------------------------------------------------------------

// MenuItem is a single selectable menu entry.
type MenuItem struct {
	Label    string
	OnSelect func()
}

type Menu struct {
	El    *layout.Element
	items []MenuItem
	width float32

	Open  bool
	hover int
}

// NewMenu builds a closed overlay menu. Add its El to the root of the tree so
// that its absolute Left/Top are interpreted as screen coordinates.
func NewMenu(width float32, items []MenuItem) *Menu {
	mu := &Menu{items: items, width: width, hover: -1}
	mu.El = layout.New(layout.Box())
	mu.El.Overlay = true // render above and hit-test before the base tree
	mu.El.Paint = mu.paint
	mu.El.OnMouse = mu.onMouse
	return mu
}

func (mu *Menu) itemHeight() float32 {
	th := theme.Current()
	if th.Metrics.MenuItemHeight > 0 {
		return th.Metrics.MenuItemHeight
	}
	return th.Metrics.ControlHeight
}

// OpenAt positions and shows the menu at the given screen coordinates, shifted
// to stay inside the viewport recorded via SetViewport (if any).
func (mu *Menu) OpenAt(x, y float32) {
	mu.Open = true
	h := float32(len(mu.items)) * mu.itemHeight()
	x, y = clampToViewport(x, y, mu.width, h)
	mu.El.Style = layout.Box().Absolute(x, y).Size(mu.width, h)
	mu.El.ReapplyStyle()
	// The menu is a screen-space overlay mounted on the root, so its absolute
	// frame is exactly {x, y, width, height}. Seed it now so the menu paints in
	// the right place on the same frame it opens: OpenAt runs during input
	// dispatch, which is after the layout pass, so without this the next paint
	// would use the stale zeroed frame and flash at the window origin until the
	// following Calculate. The next layout pass recomputes the identical frame.
	mu.El.Frame = render.Rect{X: x, Y: y, W: mu.width, H: h}
}

// Close hides the menu.
func (mu *Menu) Close() { mu.Open = false; mu.hover = -1 }

// SetItems replaces the menu's entries. Call before OpenAt when the items depend
// on context (e.g. which tree row was right-clicked). If the menu is already
// open, its height is refreshed in place to match the new item count.
func (mu *Menu) SetItems(items []MenuItem) {
	mu.items = items
	if mu.Open {
		h := float32(len(items)) * mu.itemHeight()
		mu.El.Style.Height = h
		mu.El.Frame.H = h
	}
}

func (mu *Menu) paint(dl *render.DrawList, text *shape.Engine) {
	if !mu.Open {
		return
	}
	th := theme.Current()
	f := mu.El.Frame
	itemH := mu.itemHeight()
	padX := th.Spacing.MNudge
	r := th.Radius.Medium
	drawElevationShadow(dl, f, r, th.Elevation.ShadowMd)
	dl.AddRoundedRectBorder(f, r, th.Stroke.Thin, th.Chrome, th.Border)
	// Labels wider than the configured menu width are clipped to the frame.
	dl.PushClip(f)
	for i, it := range mu.items {
		row := render.Rect{X: f.X, Y: f.Y + float32(i)*itemH, W: f.W, H: itemH}
		if i == mu.hover {
			dl.AddRect(row, th.ListHover)
		}
		style := th.Typography.Body
		_, lh := text.MeasureAt(it.Label, style.Size)
		text.DrawStringTopAt(dl, it.Label, row.X+padX, row.Y+(itemH-lh)/2, th.Foreground, style.Size)
	}
	dl.PopClip()
}

func (mu *Menu) onMouse(e *layout.Element, m *input.Mouse) {
	if !mu.Open {
		return
	}
	if e.Frame.Contains(m.X, m.Y) {
		idx := int((m.Y - e.Frame.Y) / mu.itemHeight())
		mu.hover = idx
		m.Consumed = true // block all events (including hover) from reaching layers below
		if m.Released && idx >= 0 && idx < len(mu.items) {
			if fn := mu.items[idx].OnSelect; fn != nil {
				fn()
			}
			mu.Close()
		}
	} else {
		mu.hover = -1
		if m.Pressed { // click outside closes the menu
			mu.Close()
			m.Consumed = true
		}
	}
}

// Dropdown combines a trigger Button with a Menu that opens beneath it.
type Dropdown struct {
	Button *Button
	Menu   *Menu
}

// NewDropdown builds a labelled trigger button plus its overlay menu. Add
// Dropdown.Button.El into the layout where the trigger should appear, and add
// Dropdown.Menu.El to the tree root.
func NewDropdown(label string, width float32, items []MenuItem) *Dropdown {
	d := &Dropdown{}
	d.Menu = NewMenu(width, items)
	d.Button = NewButton(label).Action(func() {
		if d.Menu.Open {
			d.Menu.Close()
			return
		}
		f := d.Button.El.Frame
		d.Menu.OpenAt(f.X, f.Y+f.H)
	})
	return d
}
