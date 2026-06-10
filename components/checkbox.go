package components

import (
	"github.com/mirzakhany/yoga"
	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
	"github.com/mirzakhany/yoga/theme"
)

// Checkbox is a toggle control with a label.
type Checkbox struct {
	El       *layout.Element
	Label    string
	Checked  bool
	OnChange func(checked bool)
	hovered  bool
	focused  bool
}

// NewCheckbox builds a labeled checkbox.
func NewCheckbox(label string) *Checkbox {
	th := theme.Current()
	eng := yoga.Text()
	c := &Checkbox{Label: label}
	box := th.Metrics.IconSizeSM
	style := th.Typography.Body
	tw, lh := eng.MeasureAt(label, style.Size)
	h := f32max(box, lh)
	w := box + th.Spacing.S + tw
	c.El = layout.New(layout.Box().Size(w, h).FlexShrink(0))
	c.El.Paint = c.paint
	c.El.OnMouse = c.onMouse
	return c
}

func (c *Checkbox) paint(dl *render.DrawList, text *shape.Engine) {
	th := theme.Current()
	f := c.El.Frame
	box := th.Metrics.IconSizeSM
	bx := f.X
	by := f.Y + (f.H-box)/2
	br := render.Rect{X: bx, Y: by, W: box, H: box}
	border := th.Border
	if c.focused {
		border = th.FocusRing
	}
	fill := th.Chrome
	if c.Checked {
		fill = th.Accent
		border = th.Accent
	}
	if c.hovered {
		fill = th.ListHover
		if c.Checked {
			fill = th.AccentHover
		}
	}
	dl.AddRoundedRectBorder(br, th.Radius.Small, th.Stroke.Thin, fill, border)
	if c.Checked {
		inner := render.Rect{X: bx + 2, Y: by + 2, W: box - 4, H: box - 4}
		yoga.Icons().Draw(dl, "check", inner, th.AccentForeground)
	}
	style := th.Typography.Body
	tx := bx + box + th.Spacing.S
	_, lh := text.MeasureAt(c.Label, style.Size)
	text.DrawStringTopAt(dl, c.Label, tx, f.Y+(f.H-lh)/2, th.Foreground, style.Size)
}

func (c *Checkbox) onMouse(e *layout.Element, m *input.Mouse) {
	c.hovered = e.Frame.Contains(m.X, m.Y)
	if c.hovered && m.Released {
		c.Checked = !c.Checked
		if c.OnChange != nil {
			c.OnChange(c.Checked)
		}
		m.Consumed = true
	}
}

func (c *Checkbox) Focus()   { c.focused = true }
func (c *Checkbox) Blur()    { c.focused = false }
func (c *Checkbox) Focused() bool { return c.focused }
func (c *Checkbox) FocusEl() *layout.Element { return c.El }
func (c *Checkbox) FocusOnClick() bool { return true }
func (c *Checkbox) CapturesTab() bool { return false }
func (c *Checkbox) HandleText(runes []rune) {
	if !c.focused {
		return
	}
	for _, r := range runes {
		if r == ' ' {
			c.Checked = !c.Checked
			if c.OnChange != nil {
				c.OnChange(c.Checked)
			}
		}
	}
}

func (c *Checkbox) HandleKeys(_ []input.KeyEvent) {}

// Changed sets the OnChange callback.
func (c *Checkbox) Changed(fn func(bool)) *Checkbox { c.OnChange = fn; return c }

// Check sets the initial checked state.
func (c *Checkbox) Check(v bool) *Checkbox { c.Checked = v; return c }
