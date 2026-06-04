package components

import (
	"strings"
	"time"
	"unicode/utf8"

	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/theme"
)

const (
	textFieldPadX      = 10
	textFieldIconSize  = 16
	textFieldIconGap   = 6
	textFieldCaretW    = 2
	textFieldBlink     = 500 * time.Millisecond
)

// TextFieldConfig configures a single-line text input.
type TextFieldConfig struct {
	Placeholder string
	Password    bool
	IconStart   string
	IconEnd     string
	Radius      float32
	BorderWidth float32
	Height      float32
}

// TextField is a single-line text input with optional border, rounded corners,
// start/end icons, and password masking.
type TextField struct {
	El    *layout.Element
	theme *theme.Theme
	atlas *render.FontAtlas
	sheet *render.SpriteSheet
	clip  input.Clipboard
	cfg   TextFieldConfig

	Value    string
	OnChange func(value string)

	focused    bool
	caret      int // byte offset
	caretShown bool
	blinkStart time.Time
}

// NewTextField builds a text field with the given configuration.
func NewTextField(atlas *render.FontAtlas, th *theme.Theme, sheet *render.SpriteSheet, clip input.Clipboard, cfg TextFieldConfig) *TextField {
	if cfg.Radius <= 0 {
		cfg.Radius = 4
	}
	if cfg.BorderWidth <= 0 {
		cfg.BorderWidth = 1
	}
	if cfg.Height <= 0 {
		cfg.Height = atlas.CellH + 16
	}
	tf := &TextField{
		theme:      th,
		atlas:      atlas,
		sheet:      sheet,
		clip:       clip,
		cfg:        cfg,
		blinkStart: time.Now(),
		caretShown: true,
	}
	tf.El = layout.New(layout.Box().H(cfg.Height).PaddingXY(textFieldPadX, 0))
	tf.El.Paint = tf.paint
	tf.El.OnMouse = tf.onMouse
	return tf
}

// Focused reports whether the field has keyboard focus.
func (tf *TextField) Focused() bool { return tf.focused }

// Blur removes keyboard focus.
func (tf *TextField) Blur() { tf.focused = false }

func (tf *TextField) displayText() string {
	if tf.cfg.Password {
		return strings.Repeat("•", utf8.RuneCountInString(tf.Value))
	}
	return tf.Value
}

func (tf *TextField) textLeft() float32 {
	x := tf.El.Frame.X + textFieldPadX
	if tf.cfg.IconStart != "" {
		x += textFieldIconSize + textFieldIconGap
	}
	return x
}

func (tf *TextField) textRight() float32 {
	x := tf.El.Frame.X + tf.El.Frame.W - textFieldPadX
	if tf.cfg.IconEnd != "" {
		x -= textFieldIconSize + textFieldIconGap
	}
	return x
}

func (tf *TextField) displayPrefixForCaret() string {
	disp := tf.displayText()
	runes := 0
	for i := 0; i < len(tf.Value) && i < tf.caret; {
		_, sz := utf8.DecodeRuneInString(tf.Value[i:])
		i += sz
		runes++
	}
	ri := 0
	for count := 0; count < runes && ri < len(disp); count++ {
		_, sz := utf8.DecodeRuneInString(disp[ri:])
		ri += sz
	}
	return disp[:ri]
}

func (tf *TextField) caretX() float32 {
	tw, _ := tf.atlas.Measure(tf.displayPrefixForCaret())
	return tf.textLeft() + tw
}

func (tf *TextField) setCaretFromX(px float32) {
	s := tf.displayText()
	x := tf.textLeft()
	if px <= x {
		tf.caret = 0
		tf.blinkStart = time.Now()
		tf.caretShown = true
		return
	}
	runeIdx := 0
	for i := 0; i < len(s); {
		_, sz := utf8.DecodeRuneInString(s[i:])
		tw, _ := tf.atlas.Measure(s[:i+sz])
		if x+tw > px {
			break
		}
		i += sz
		runeIdx++
	}
	// Map rune index to byte offset in Value.
	off := 0
	count := 0
	for i := 0; i < len(tf.Value) && count < runeIdx; {
		_, sz := utf8.DecodeRuneInString(tf.Value[i:])
		i += sz
		off = i
		count++
	}
	tf.caret = off
	tf.blinkStart = time.Now()
	tf.caretShown = true
}

func (tf *TextField) setValue(s string) {
	// Single line: strip newlines.
	if i := strings.IndexAny(s, "\n\r"); i >= 0 {
		s = s[:i]
	}
	if s == tf.Value {
		return
	}
	tf.Value = s
	if tf.OnChange != nil {
		tf.OnChange(s)
	}
}

func (tf *TextField) insertAtCaret(s string) {
	if s == "" {
		return
	}
	if i := strings.IndexAny(s, "\n\r"); i >= 0 {
		s = s[:i]
	}
	tf.setValue(tf.Value[:tf.caret] + s + tf.Value[tf.caret:])
	tf.caret += len(s)
	tf.blinkStart = time.Now()
	tf.caretShown = true
}

func (tf *TextField) paint(dl *render.DrawList, _ *render.FontAtlas) {
	f := tf.El.Frame
	border := tf.theme.Border
	if tf.focused {
		border = tf.theme.Accent
	}
	dl.AddRoundedRectBorder(f, tf.cfg.Radius, tf.cfg.BorderWidth, tf.theme.Panel, border)

	iconY := f.Y + (f.H-textFieldIconSize)/2
	if tf.cfg.IconStart != "" {
		ix := f.X + textFieldPadX
		tf.sheet.Draw(dl, tf.cfg.IconStart, render.Rect{X: ix, Y: iconY, W: textFieldIconSize, H: textFieldIconSize}, tf.theme.TextDim)
	}
	if tf.cfg.IconEnd != "" {
		ix := f.X + f.W - textFieldPadX - textFieldIconSize
		tf.sheet.Draw(dl, tf.cfg.IconEnd, render.Rect{X: ix, Y: iconY, W: textFieldIconSize, H: textFieldIconSize}, tf.theme.TextDim)
	}

	tx := tf.textLeft()
	tr := tf.textRight()
	_, th := tf.atlas.Measure("Ag")
	ty := f.Y + (f.H-th)/2

	show := tf.displayText()
	col := tf.theme.Text
	if show == "" && !tf.focused {
		show = tf.cfg.Placeholder
		col = tf.theme.TextDim
	}
	if show != "" {
		// Clip text to the inner area between icons.
		dl.PushClip(render.Rect{X: tx, Y: f.Y, W: tr - tx, H: f.H})
		tf.atlas.DrawText(dl, show, tx, ty, col)
		dl.PopClip()
	}

	if tf.focused && tf.caretShown {
		cx := tf.caretX()
		if cx < tx {
			cx = tx
		}
		if cx > tr {
			cx = tr
		}
		dl.AddRect(render.Rect{X: cx, Y: ty, W: textFieldCaretW, H: th}, tf.theme.Accent)
	}
}

func (tf *TextField) onMouse(e *layout.Element, m *input.Mouse) {
	if !e.Frame.Contains(m.X, m.Y) {
		return
	}
	if m.Pressed {
		tf.focused = true
		tf.setCaretFromX(m.X)
		m.Consumed = true
	}
}

// Update advances caret blink; call once per frame.
func (tf *TextField) Update(_ *input.Mouse) {
	if !tf.focused {
		return
	}
	if time.Since(tf.blinkStart) >= textFieldBlink {
		tf.caretShown = !tf.caretShown
		tf.blinkStart = time.Now()
	}
}

// HandleText inserts text-producing characters for this frame.
func (tf *TextField) HandleText(runes []rune) {
	if !tf.focused || len(runes) == 0 {
		return
	}
	tf.insertAtCaret(string(runes))
}

// HandleKeys processes navigation and editing keys for this frame.
func (tf *TextField) HandleKeys(keys []input.KeyEvent) {
	if !tf.focused {
		return
	}
	for _, ev := range keys {
		if ev.Mods.Primary() {
			switch ev.Key {
			case input.KeyA:
				tf.caret = len(tf.Value)
			case input.KeyC:
				if tf.clip != nil && tf.Value != "" {
					tf.clip.Set(tf.Value)
				}
			case input.KeyX:
				if tf.clip != nil && tf.Value != "" {
					tf.clip.Set(tf.Value)
					tf.setValue("")
					tf.caret = 0
				}
			case input.KeyV:
				if tf.clip != nil {
					tf.insertAtCaret(tf.clip.Get())
				}
			}
			continue
		}
		switch ev.Key {
		case input.KeyBackspace:
			if tf.caret > 0 {
				prev := prevRuneOff(tf.Value, tf.caret)
				tf.setValue(tf.Value[:prev] + tf.Value[tf.caret:])
				tf.caret = prev
			}
		case input.KeyDelete:
			if tf.caret < len(tf.Value) {
				next := nextRuneOff(tf.Value, tf.caret)
				tf.setValue(tf.Value[:tf.caret] + tf.Value[next:])
			}
		case input.KeyLeft:
			tf.caret = prevRuneOff(tf.Value, tf.caret)
			tf.blinkStart = time.Now()
			tf.caretShown = true
		case input.KeyRight:
			tf.caret = nextRuneOff(tf.Value, tf.caret)
			tf.blinkStart = time.Now()
			tf.caretShown = true
		case input.KeyHome:
			tf.caret = 0
			tf.blinkStart = time.Now()
			tf.caretShown = true
		case input.KeyEnd:
			tf.caret = len(tf.Value)
			tf.blinkStart = time.Now()
			tf.caretShown = true
		}
	}
}

func prevRuneOff(s string, off int) int {
	if off <= 0 {
		return 0
	}
	_, sz := utf8.DecodeLastRuneInString(s[:off])
	return off - sz
}

func nextRuneOff(s string, off int) int {
	if off >= len(s) {
		return len(s)
	}
	_, sz := utf8.DecodeRuneInString(s[off:])
	return off + sz
}
