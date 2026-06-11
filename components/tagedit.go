package components

import (
	"strings"

	"github.com/mirzakhany/yoga"
	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
	"github.com/mirzakhany/yoga/theme"
)

// TagEdit is a chip input: removable tags plus an inline text field.
type TagEdit struct {
	El       *layout.Element
	tags     []string
	OnChange func(tags []string)
	field    *TextField
	content  *layout.Element
	inputW   float32
}

// NewTagEdit builds a tag editor with the given width and optional initial tags.
func NewTagEdit(width float32, initial ...string) *TagEdit {
	th := theme.Current()
	chipH := th.Metrics.ControlHeight - th.Spacing.S
	t := &TagEdit{inputW: 80, tags: append([]string(nil), initial...)}
	t.field = NewTextField(TextFieldConfig{
		Placeholder: "Add tag...",
		Height:      chipH,
	})
	t.content = layout.New(layout.Box().
		Direction(layout.Row).
		FlexWrap(layout.DoWrap).
		Gap(th.Spacing.XS).
		AlignItems(layout.AlignCenter))
	t.El = layout.New(layout.Box().W(width).PaddingAll(th.Spacing.S), t.content)
	t.El.Clip = true
	t.El.Paint = t.paint
	t.syncChildren()
	return t
}

// Tags returns a copy of the current tag list.
func (t *TagEdit) Tags() []string {
	out := make([]string, len(t.tags))
	copy(out, t.tags)
	return out
}

// SetTags replaces the current tag list, rebuilds the layout, and fires OnChange.
func (t *TagEdit) SetTags(tags []string) {
	t.tags = append([]string(nil), tags...)
	t.syncChildren()
	if t.OnChange != nil {
		t.OnChange(t.Tags())
	}
}

func (t *TagEdit) addTag(s string) {
	s = strings.TrimSpace(s)
	if s == "" {
		return
	}
	for _, existing := range t.tags {
		if strings.EqualFold(existing, s) {
			return
		}
	}
	t.tags = append(t.tags, s)
	t.field.setValue("")
	t.syncChildren()
	if t.OnChange != nil {
		t.OnChange(t.Tags())
	}
}

func (t *TagEdit) removeTag(i int) {
	if i < 0 || i >= len(t.tags) {
		return
	}
	t.tags = append(t.tags[:i], t.tags[i+1:]...)
	t.syncChildren()
	if t.OnChange != nil {
		t.OnChange(t.Tags())
	}
}

func (t *TagEdit) chipSize(tag string) (w, h float32) {
	th := theme.Current()
	style := th.Typography.Caption
	tw, _ := yoga.Text().MeasureAt(tag, style.Size)
	h = th.Metrics.ControlHeight - th.Spacing.S
	w = tw + th.Spacing.M + th.Metrics.IconSizeSM
	return w, h
}

func (t *TagEdit) syncChildren() {
	th := theme.Current()
	chipH := th.Metrics.ControlHeight - th.Spacing.S
	children := make([]*layout.Element, 0, len(t.tags)+1)
	for i, tag := range t.tags {
		w, _ := t.chipSize(tag)
		idx := i
		label := tag
		el := layout.New(layout.Box().Size(w, chipH).FlexShrink(0))
		el.Paint = func(dl *render.DrawList, eng *shape.Engine) {
			t.paintChip(dl, eng, el, label)
		}
		el.OnMouse = func(e *layout.Element, m *input.Mouse) {
			t.onChipMouse(e, m, idx)
		}
		children = append(children, el)
	}

	s := t.field.El.Style
	s.Grow = 1
	s.Shrink = 0
	s.MinWidth = t.inputW
	t.field.El.Style = s
	children = append(children, t.field.El)

	t.content.Children = children
	t.El.MarkDirty()
}

func (t *TagEdit) paintChip(dl *render.DrawList, eng *shape.Engine, el *layout.Element, tag string) {
	th := theme.Current()
	f := el.Frame
	style := th.Typography.Caption
	chipH := th.Metrics.ControlHeight - th.Spacing.S
	dl.AddRoundedRect(f, th.Radius.Small, th.ChromeMuted)
	_, lh := eng.MeasureAt(tag, style.Size)
	eng.DrawStringTopAt(dl, tag, f.X+th.Spacing.S, f.Y+(chipH-lh)/2, th.Foreground, style.Size)
	iconSz := th.Metrics.IconSizeSM - 4
	ix := f.X + f.W - iconSz - th.Spacing.XS
	iy := f.Y + (chipH-iconSz)/2
	yoga.Icons().Draw(dl, "close", render.Rect{X: ix, Y: iy, W: iconSz, H: iconSz}, th.ForegroundMuted)
}

func (t *TagEdit) onChipMouse(e *layout.Element, m *input.Mouse, idx int) {
	th := theme.Current()
	f := e.Frame
	iconSz := th.Metrics.IconSizeSM - 4
	chipH := th.Metrics.ControlHeight - th.Spacing.S
	closeR := render.Rect{
		X: f.X + f.W - iconSz - th.Spacing.XS,
		Y: f.Y + (chipH-iconSz)/2,
		W: iconSz, H: iconSz,
	}
	if closeR.Contains(m.X, m.Y) && m.Released {
		t.removeTag(idx)
		m.Consumed = true
	}
}

func (t *TagEdit) paint(dl *render.DrawList, _ *shape.Engine) {
	th := theme.Current()
	f := t.El.Frame
	dl.AddRoundedRectBorder(f, th.Radius.Medium, th.Stroke.Thin, th.Chrome, th.Border)
}

func (t *TagEdit) Update(m *input.Mouse) { t.field.Update(m) }

func (t *TagEdit) Focus()                   { t.field.Focus() }
func (t *TagEdit) Blur()                    { t.field.Blur() }
func (t *TagEdit) Focused() bool            { return t.field.Focused() }
func (t *TagEdit) FocusEl() *layout.Element { return t.El }
func (t *TagEdit) FocusOnClick() bool       { return true }
func (t *TagEdit) CapturesTab() bool        { return false }

func (t *TagEdit) HandleText(runes []rune) {
	if !t.field.Focused() {
		return
	}
	s := string(runes)
	if strings.ContainsAny(s, "\n\r") {
		t.addTag(t.field.Value)
		return
	}
	if strings.Contains(s, ",") {
		combined := t.field.Value + s
		t.field.setValue("")
		for _, p := range strings.Split(combined, ",") {
			t.addTag(p)
		}
		return
	}
	t.field.HandleText(runes)
}

func (t *TagEdit) HandleKeys(keys []input.KeyEvent) {
	if !t.field.Focused() {
		return
	}
	for _, ev := range keys {
		if ev.Key == input.KeyEnter && ev.Mods == 0 {
			t.addTag(t.field.Value)
			return
		}
		if ev.Key == input.KeyBackspace && t.field.Value == "" && len(t.tags) > 0 {
			t.removeTag(len(t.tags) - 1)
			return
		}
	}
	t.field.HandleKeys(keys)
}
