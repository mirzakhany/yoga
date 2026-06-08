package components

import (
	"strings"

	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
	"github.com/mirzakhany/yoga/theme"
)

// TagEdit is a chip input: removable tags plus an inline text field.
type TagEdit struct {
	El       *layout.Element
	theme    *theme.Theme
	text     *shape.Engine
	sheet    *render.SpriteSheet
	clip     input.Clipboard
	Tags     []string
	OnChange func(tags []string)
	field    *TextField
	content  *layout.Element
	inputW   float32
}

// NewTagEdit builds a tag editor with the given width.
func NewTagEdit(eng *shape.Engine, th *theme.Theme, sheet *render.SpriteSheet, clip input.Clipboard, width float32) *TagEdit {
	chipH := th.Metrics.ControlHeight - th.Spacing.S
	t := &TagEdit{theme: th, text: eng, sheet: sheet, clip: clip, inputW: 80}
	t.field = NewTextField(eng, th, sheet, clip, TextFieldConfig{
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

func (t *TagEdit) addTag(s string) {
	s = strings.TrimSpace(s)
	if s == "" {
		return
	}
	for _, existing := range t.Tags {
		if existing == s {
			return
		}
	}
	t.Tags = append(t.Tags, s)
	t.field.setValue("")
	t.syncChildren()
	if t.OnChange != nil {
		t.OnChange(t.Tags)
	}
}

func (t *TagEdit) removeTag(i int) {
	if i < 0 || i >= len(t.Tags) {
		return
	}
	t.Tags = append(t.Tags[:i], t.Tags[i+1:]...)
	t.syncChildren()
	if t.OnChange != nil {
		t.OnChange(t.Tags)
	}
}

func (t *TagEdit) chipSize(tag string) (w, h float32) {
	style := t.theme.Typography.Caption
	tw, _ := t.text.MeasureAt(tag, style.Size)
	h = t.theme.Metrics.ControlHeight - t.theme.Spacing.S
	w = tw + t.theme.Spacing.M + t.theme.Metrics.IconSizeSM
	return w, h
}

func (t *TagEdit) syncChildren() {
	chipH := t.theme.Metrics.ControlHeight - t.theme.Spacing.S
	children := make([]*layout.Element, 0, len(t.Tags)+1)
	for i, tag := range t.Tags {
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
	f := el.Frame
	style := t.theme.Typography.Caption
	chipH := t.theme.Metrics.ControlHeight - t.theme.Spacing.S
	dl.AddRoundedRect(f, t.theme.Radius.Small, t.theme.ChromeMuted)
	_, lh := eng.MeasureAt(tag, style.Size)
	eng.DrawStringTopAt(dl, tag, f.X+t.theme.Spacing.S, f.Y+(chipH-lh)/2, t.theme.Foreground, style.Size)
	iconSz := t.theme.Metrics.IconSizeSM - 4
	ix := f.X + f.W - iconSz - t.theme.Spacing.XS
	iy := f.Y + (chipH-iconSz)/2
	t.sheet.Draw(dl, "close", render.Rect{X: ix, Y: iy, W: iconSz, H: iconSz}, t.theme.ForegroundMuted)
}

func (t *TagEdit) onChipMouse(e *layout.Element, m *input.Mouse, idx int) {
	f := e.Frame
	iconSz := t.theme.Metrics.IconSizeSM - 4
	chipH := t.theme.Metrics.ControlHeight - t.theme.Spacing.S
	closeR := render.Rect{
		X: f.X + f.W - iconSz - t.theme.Spacing.XS,
		Y: f.Y + (chipH-iconSz)/2,
		W: iconSz, H: iconSz,
	}
	if closeR.Contains(m.X, m.Y) && m.Released {
		t.removeTag(idx)
		m.Consumed = true
	}
}

func (t *TagEdit) paint(dl *render.DrawList, _ *shape.Engine) {
	f := t.El.Frame
	dl.AddRoundedRectBorder(f, t.theme.Radius.Medium, t.theme.Stroke.Thin, t.theme.Chrome, t.theme.Border)
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
		parts := strings.Split(s, ",")
		for _, p := range parts {
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
		if ev.Key == input.KeyBackspace && t.field.Value == "" && len(t.Tags) > 0 {
			t.removeTag(len(t.Tags) - 1)
			return
		}
	}
	t.field.HandleKeys(keys)
}
