package components

import (
	"github.com/mirzakhany/yoga"
	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
	"github.com/mirzakhany/yoga/theme"
)

// RadioGroup owns a set of mutually exclusive radio buttons.
type RadioGroup struct {
	Value   int
	OnChange func(value int)
	items   []*Radio
}

// NewRadioGroup creates an empty radio group.
func NewRadioGroup() *RadioGroup {
	return &RadioGroup{Value: -1}
}

// Add registers a radio option and returns it.
func (g *RadioGroup) Add(label string) *Radio {
	idx := len(g.items)
	r := newRadio(label, g, idx)
	g.items = append(g.items, r)
	return r
}

// Select sets the active index.
func (g *RadioGroup) Select(i int) {
	if i < 0 || i >= len(g.items) {
		return
	}
	g.Value = i
	if g.OnChange != nil {
		g.OnChange(i)
	}
}

// Radio is one option in a RadioGroup.
type Radio struct {
	El      *layout.Element
	Label   string
	group   *RadioGroup
	index   int
	hovered bool
	focused bool
}

func newRadio(label string, group *RadioGroup, index int) *Radio {
	th := theme.Current()
	eng := yoga.Text()
	r := &Radio{Label: label, group: group, index: index}
	box := th.Metrics.IconSizeSM
	style := th.Typography.Body
	tw, lh := eng.MeasureAt(label, style.Size)
	h := f32max(box, lh)
	w := box + th.Spacing.S + tw
	r.El = layout.New(layout.Box().Size(w, h).FlexShrink(0))
	r.El.Paint = r.paint
	r.El.OnMouse = r.onMouse
	return r
}

func (r *Radio) selected() bool { return r.group.Value == r.index }

func (r *Radio) paint(dl *render.DrawList, text *shape.Engine) {
	th := theme.Current()
	f := r.El.Frame
	box := th.Metrics.IconSizeSM
	bx := f.X
	by := f.Y + (f.H-box)/2
	br := render.Rect{X: bx, Y: by, W: box, H: box}
	border := th.Border
	if r.focused {
		border = th.FocusRing
	}
	fill := th.Chrome
	if r.selected() {
		border = th.Accent
	}
	if r.hovered {
		fill = th.ListHover
	}
	dl.AddRoundedRectBorder(br, box/2, th.Stroke.Thin, fill, border)
	if r.selected() {
		dot := box * 0.35
		dl.AddRoundedRect(render.Rect{
			X: bx + (box-dot)/2, Y: by + (box-dot)/2, W: dot, H: dot,
		}, dot/2, th.Accent)
	}
	style := th.Typography.Body
	tx := bx + box + th.Spacing.S
	_, lh := text.MeasureAt(r.Label, style.Size)
	text.DrawStringTopAt(dl, r.Label, tx, f.Y+(f.H-lh)/2, th.Foreground, style.Size)
}

func (r *Radio) onMouse(e *layout.Element, m *input.Mouse) {
	r.hovered = e.Frame.Contains(m.X, m.Y)
	if r.hovered && m.Released {
		r.group.Select(r.index)
		m.Consumed = true
	}
}

func (r *Radio) Focus()   { r.focused = true }
func (r *Radio) Blur()    { r.focused = false }
func (r *Radio) Focused() bool { return r.focused }
func (r *Radio) FocusEl() *layout.Element { return r.El }
func (r *Radio) FocusOnClick() bool { return true }
func (r *Radio) CapturesTab() bool { return false }
func (r *Radio) HandleText(runes []rune) {
	if !r.focused {
		return
	}
	for _, ch := range runes {
		if ch == ' ' {
			r.group.Select(r.index)
		}
	}
}

func (r *Radio) HandleKeys(_ []input.KeyEvent) {}

// Changed sets the RadioGroup's OnChange callback (convenience so it can be
// wired while chaining from RadioGroup.Add).
func (g *RadioGroup) Changed(fn func(int)) *RadioGroup { g.OnChange = fn; return g }
