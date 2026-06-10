package components

import (
	"github.com/mirzakhany/yoga"
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
func NewSelect(width float32, options []SelectOption) *Select {
	th := theme.Current()
	s := &Select{Options: options, Width: width}
	if len(options) > 0 {
		s.Selected = 0
	}
	h := th.Metrics.ControlHeight
	s.El = layout.New(layout.Box().W(width).H(h).FlexShrink(0))
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
		s.menu = NewMenu(s.Width, items)
	} else {
		s.menu.SetItems(items)
	}
}

// MenuEl returns the overlay menu element (mount on app root).
func (s *Select) MenuEl() *layout.Element { return s.menu.El }

func (s *Select) paint(dl *render.DrawList, text *shape.Engine) {
	th := theme.Current()
	f := s.El.Frame
	bg := th.ChromeMuted
	if s.hovered || s.pressed {
		bg = th.ListHover
	}
	border := th.Border
	if s.focused {
		border = th.FocusRing
	}
	dl.AddRoundedRectBorder(f, th.Radius.Medium, th.Stroke.Thin, bg, border)
	label := s.selectedLabel()
	style := th.Typography.Body
	_, lh := text.MeasureAt(label, style.Size)
	pad := th.Spacing.MNudge
	text.DrawStringTopAt(dl, label, f.X+pad, f.Y+(f.H-lh)/2, th.Foreground, style.Size)
	iconSz := th.Metrics.IconSizeSM
	ix := f.X + f.W - pad - iconSz
	iy := f.Y + (f.H-iconSz)/2
	yoga.Icons().Draw(dl, "expand_more", render.Rect{X: ix, Y: iy, W: iconSz, H: iconSz}, th.ForegroundMuted)
}

func (s *Select) onMouse(e *layout.Element, m *input.Mouse) {
	s.hovered = e.Frame.Contains(m.X, m.Y)
	if s.hovered && m.Pressed {
		s.pressed = true
		m.Consumed = true
	}
	if m.Released && s.pressed && s.hovered {
		if s.menu.Open {
			s.menu.Close()
		} else {
			s.rebuildMenu()
			fr := e.Frame
			s.menu.OpenAt(fr.X, fr.Y+fr.H)
		}
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

// Changed sets the OnChange callback.
func (s *Select) Changed(fn func(string)) *Select { s.OnChange = fn; return s }
