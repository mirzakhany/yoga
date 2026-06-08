package components

import (
	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
	"github.com/mirzakhany/yoga/theme"
)

// SelectOption is one entry in a Select control.
type SelectOption struct {
	Label string
	Value string
}

// Select is a form dropdown that shows the selected value and opens a menu.
type Select struct {
	El       *layout.Element
	theme    *theme.Theme
	text     *shape.Engine
	sheet    *render.SpriteSheet
	Options  []SelectOption
	Selected int
	Width    float32
	OnChange func(value string)
	menu     *Menu
	hovered  bool
	pressed  bool
	focused  bool
}

// NewSelect builds a select control with the given width.
func NewSelect(eng *shape.Engine, th *theme.Theme, sheet *render.SpriteSheet, width float32, options []SelectOption) *Select {
	s := &Select{
		theme: th, text: eng, sheet: sheet, Options: options, Width: width,
	}
	if len(options) > 0 {
		s.Selected = 0
	}
	h := th.Metrics.ControlHeight
	s.El = layout.New(layout.Box().W(width).H(h))
	s.El.Paint = s.paint
	s.El.OnMouse = s.onMouse
	s.rebuildMenu()
	return s
}

func (s *Select) selectedLabel() string {
	if s.Selected < 0 || s.Selected >= len(s.Options) {
		return ""
	}
	return s.Options[s.Selected].Label
}

func (s *Select) rebuildMenu() {
	items := make([]MenuItem, len(s.Options))
	for i, opt := range s.Options {
		i, opt := i, opt
		items[i] = MenuItem{Label: opt.Label, OnSelect: func() {
			s.Selected = i
			if s.OnChange != nil {
				s.OnChange(opt.Value)
			}
		}}
	}
	if s.menu == nil {
		s.menu = NewMenu(s.text, s.theme, s.Width, items)
	} else {
		s.menu.SetItems(items)
	}
}

// MenuEl returns the overlay menu element (mount on app root).
func (s *Select) MenuEl() *layout.Element { return s.menu.El }

func (s *Select) paint(dl *render.DrawList, text *shape.Engine) {
	f := s.El.Frame
	bg := s.theme.ChromeMuted
	if s.hovered || s.pressed {
		bg = s.theme.ListHover
	}
	border := s.theme.Border
	if s.focused {
		border = s.theme.FocusRing
	}
	dl.AddRoundedRectBorder(f, s.theme.Radius.Medium, s.theme.Stroke.Thin, bg, border)
	label := s.selectedLabel()
	style := s.theme.Typography.Body
	_, lh := text.MeasureAt(label, style.Size)
	pad := s.theme.Spacing.MNudge
	text.DrawStringTopAt(dl, label, f.X+pad, f.Y+(f.H-lh)/2, s.theme.Foreground, style.Size)
	iconSz := s.theme.Metrics.IconSizeSM
	ix := f.X + f.W - pad - iconSz
	iy := f.Y + (f.H-iconSz)/2
	s.sheet.Draw(dl, "expand_more", render.Rect{X: ix, Y: iy, W: iconSz, H: iconSz}, s.theme.ForegroundMuted)
}

func (s *Select) onMouse(e *layout.Element, m *input.Mouse) {
	s.hovered = e.Frame.Contains(m.X, m.Y)
	if s.hovered && m.Pressed {
		s.pressed = true
		m.Consumed = true
	}
	if m.Released && s.pressed && s.hovered {
		s.rebuildMenu()
		fr := e.Frame
		s.menu.OpenAt(fr.X, fr.Y+fr.H)
	}
	if !m.Down {
		s.pressed = false
	}
}

func (s *Select) Focus()   { s.focused = true }
func (s *Select) Blur()    { s.focused = false }
func (s *Select) Focused() bool { return s.focused }
func (s *Select) FocusEl() *layout.Element { return s.El }
func (s *Select) FocusOnClick() bool { return true }
func (s *Select) CapturesTab() bool { return false }
func (s *Select) HandleText(_ []rune) {}
func (s *Select) HandleKeys(_ []input.KeyEvent) {}
