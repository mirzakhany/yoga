package components

import (
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
	theme   *theme.Theme
	text  *shape.Engine
	label   string
	hovered bool
	pressed bool
	OnClick func()
}

const (
	btnPadX = 12
	btnPadY = 6
)

// NewButton builds a button sized to its label.
func NewButton(text *shape.Engine, theme *theme.Theme, label string, onClick func()) *Button {
	tw, th := text.Measure(label)
	b := &Button{
		theme:   theme,
		text:    text,
		label:   label,
		OnClick: onClick,
	}
	b.El = layout.New(layout.Box().Size(tw+2*btnPadX, th+2*btnPadY))
	b.El.Paint = b.paint
	b.El.OnMouse = b.onMouse
	return b
}

func (b *Button) paint(dl *render.DrawList, text *shape.Engine) {
	bg := b.theme.Panel
	switch {
	case b.pressed:
		bg = b.theme.Active
	case b.hovered:
		bg = b.theme.Hover
	}
	dl.AddRect(b.El.Frame, bg)

	tw, th := text.Measure(b.label)
	tx := b.El.Frame.X + (b.El.Frame.W-tw)/2
	ty := b.El.Frame.Y + (b.El.Frame.H-th)/2
	text.DrawStringTop(dl, b.label, tx, ty, b.theme.Text)
}

func (b *Button) onMouse(e *layout.Element, m *input.Mouse) {
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
func NewLabelRow(text *shape.Engine, theme *theme.Theme, label string, height float32, onClick func()) *layout.Element {
	hovered := false
	el := layout.New(layout.Box().H(height).PaddingXY(10, 0).JustifyContent(layout.JustifyCenter))
	el.Paint = func(dl *render.DrawList, eng *shape.Engine) {
		if hovered {
			dl.AddRect(el.Frame, theme.Hover)
		}
		_, th := eng.Measure(label)
		ty := el.Frame.Y + (el.Frame.H-th)/2
		eng.DrawStringTop(dl, label, el.Frame.X+10, ty, theme.Text)
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
	theme *theme.Theme
	Offset        *float32 // owner-owned scroll position in pixels (along the axis)
	ContentHeight *float32 // owner-owned total content length along the axis, in pixels

	dragging bool
	hovered  bool
	grab     float32 // cursor-to-thumb offset captured at drag start (along axis)
}

const (
	minThumb    = 24
	thumbInset  = 2 // margin between thumb and track edge (each side)
)

// NewScrollbar creates a vertical scrollbar bound to the given offset/content
// pointers. width is the track thickness in pixels.
func NewScrollbar(theme *theme.Theme, offset, contentHeight *float32, width float32) *Scrollbar {
	return NewScrollbarAxis(theme, Vertical, offset, contentHeight, width)
}

// NewScrollbarAxis creates a scrollbar for the given axis. The track pins itself
// to the right edge (vertical) or bottom edge (horizontal) of its parent;
// thickness is the cross-axis size in pixels. Offset/content are measured along
// the axis (height for vertical, width for horizontal).
func NewScrollbarAxis(theme *theme.Theme, axis Axis, offset, content *float32, thickness float32) *Scrollbar {
	s := &Scrollbar{theme: theme, axis: axis, Offset: offset, ContentHeight: content}
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
	track := s.El.Frame
	ch := *s.ContentHeight
	along := s.trackLen()
	maxOff := f32max(0, ch-along)

	thumbLen := along
	if ch > along && ch > 0 {
		thumbLen = along * along / ch
	}
	thumbLen = clampf(thumbLen, minThumb, along)

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
	th := s.thumb()
	if s.axis == Horizontal {
		return render.Rect{X: th.X, Y: th.Y + thumbInset, W: th.W, H: th.H - 2*thumbInset}
	}
	return render.Rect{X: th.X + thumbInset, Y: th.Y, W: th.W - 2*thumbInset, H: th.H}
}

// setOffsetFromPointer maps a pointer position along the track to a scroll offset.
func (s *Scrollbar) setOffsetFromPointer(px, py float32) {
	track := s.El.Frame
	th := s.thumb()
	along := s.trackLen()
	maxOff := s.maxOffset()
	if maxOff <= 0 {
		return
	}
	if s.axis == Horizontal {
		travel := along - th.W
		if travel <= 0 {
			return
		}
		*s.Offset = (px - s.grab - track.X) / travel * maxOff
	} else {
		travel := along - th.H
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
	th := s.thumbVisual()
	s.hovered = th.Contains(m.X, m.Y)

	if m.Pressed {
		m.Consumed = true
		if th.Contains(m.X, m.Y) {
			s.dragging = true
			if s.axis == Horizontal {
				s.grab = m.X - th.X
			} else {
				s.grab = m.Y - th.Y
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
	if area.Contains(m.X, m.Y) {
		if s.axis == Horizontal {
			if m.ScrollX != 0 {
				*s.Offset -= m.ScrollX * 3 * 14
			}
		} else if m.ScrollY != 0 {
			*s.Offset -= m.ScrollY * 3 * 14 // ~3 lines per wheel notch
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
	dl.AddRect(s.El.Frame, s.theme.ScrollTrack)
	col := s.theme.ScrollThumb
	if s.dragging || s.hovered {
		col = s.theme.ScrollThumbHover
	}
	dl.AddRect(s.thumbVisual(), col)
}

// ----------------------------------------------------------------------------
// Icon: renders a named sprite region from the atlas-backed sprite sheet,
// tinted by a color. The Element is fixed-size and the Paint hook stretches the
// sprite over its frame.
// ----------------------------------------------------------------------------

// NewIcon builds a size x size icon element drawing the named sprite.
func NewIcon(sheet *render.SpriteSheet, name string, size float32, color render.Color) *layout.Element {
	el := layout.New(layout.Box().Size(size, size))
	el.Paint = func(dl *render.DrawList, _ *shape.Engine) {
		sheet.Draw(dl, name, el.Frame, color)
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

const menuItemH = 26

type Menu struct {
	El    *layout.Element
	theme *theme.Theme
	text *shape.Engine
	items []MenuItem
	width float32

	Open  bool
	hover int
}

// NewMenu builds a closed overlay menu. Add its El to the root of the tree so
// that its absolute Left/Top are interpreted as screen coordinates.
func NewMenu(text *shape.Engine, theme *theme.Theme, width float32, items []MenuItem) *Menu {
	mu := &Menu{theme: theme, text: text, items: items, width: width, hover: -1}
	mu.El = layout.New(layout.Box())
	mu.El.Overlay = true // render above and hit-test before the base tree
	mu.El.Paint = mu.paint
	mu.El.OnMouse = mu.onMouse
	return mu
}

// OpenAt positions and shows the menu at the given screen coordinates.
func (mu *Menu) OpenAt(x, y float32) {
	mu.Open = true
	h := float32(len(mu.items)) * menuItemH
	mu.El.Style = layout.Box().Absolute(x, y).Size(mu.width, h)
	mu.El.ReapplyStyle()
}

// Close hides the menu.
func (mu *Menu) Close() { mu.Open = false; mu.hover = -1 }

// SetItems replaces the menu's entries. Call before OpenAt when the items depend
// on context (e.g. which tree row was right-clicked).
func (mu *Menu) SetItems(items []MenuItem) { mu.items = items }

func (mu *Menu) paint(dl *render.DrawList, text *shape.Engine) {
	if !mu.Open {
		return
	}
	f := mu.El.Frame
	dl.AddRect(f, mu.theme.Panel)
	for i, it := range mu.items {
		row := render.Rect{X: f.X, Y: f.Y + float32(i)*menuItemH, W: f.W, H: menuItemH}
		if i == mu.hover {
			dl.AddRect(row, mu.theme.Hover)
		}
		_, th := text.Measure(it.Label)
		text.DrawStringTop(dl, it.Label, row.X+10, row.Y+(menuItemH-th)/2, mu.theme.Text)
	}
}

func (mu *Menu) onMouse(e *layout.Element, m *input.Mouse) {
	if !mu.Open {
		return
	}
	if e.Frame.Contains(m.X, m.Y) {
		idx := int((m.Y - e.Frame.Y) / menuItemH)
		mu.hover = idx
		if m.Pressed {
			m.Consumed = true
		}
		if m.Released && idx >= 0 && idx < len(mu.items) {
			if fn := mu.items[idx].OnSelect; fn != nil {
				fn()
			}
			mu.Close()
			m.Consumed = true
		}
	} else {
		mu.hover = -1
		if m.Pressed { // click outside closes the menu
			mu.Close()
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
func NewDropdown(text *shape.Engine, theme *theme.Theme, label string, width float32, items []MenuItem) *Dropdown {
	d := &Dropdown{}
	d.Menu = NewMenu(text, theme, width, items)
	d.Button = NewButton(text, theme, label, func() {
		if d.Menu.Open {
			d.Menu.Close()
			return
		}
		f := d.Button.El.Frame
		d.Menu.OpenAt(f.X, f.Y+f.H)
	})
	return d
}
