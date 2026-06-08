package components

import (
	"strings"
	"time"
	"unicode/utf8"

	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
	"github.com/mirzakhany/yoga/theme"
)

const textFieldBlink = 500 * time.Millisecond

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
	text *shape.Engine
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
func NewTextField(text *shape.Engine, th *theme.Theme, sheet *render.SpriteSheet, clip input.Clipboard, cfg TextFieldConfig) *TextField {
	if cfg.Radius <= 0 {
		cfg.Radius = th.Radius.Medium
	}
	if cfg.BorderWidth <= 0 {
		cfg.BorderWidth = th.Stroke.Thin
	}
	style := th.Typography.Body
	if cfg.Height <= 0 {
		cfg.Height = style.LineHeight + th.Spacing.S*2
	}
	tf := &TextField{
		theme:      th,
		text:       text,
		sheet:      sheet,
		clip:       clip,
		cfg:        cfg,
		blinkStart: time.Now(),
		caretShown: true,
	}
	padX := th.Spacing.MNudge
	tf.El = layout.New(layout.Box().H(cfg.Height).PaddingXY(padX, 0))
	tf.El.Paint = tf.paint
	tf.El.OnMouse = tf.onMouse
	return tf
}

// Focus grants keyboard focus to the field.
func (tf *TextField) Focus() {
	tf.focused = true
	tf.blinkStart = time.Now()
	tf.caretShown = true
}

// Focused reports whether the field has keyboard focus.
func (tf *TextField) Focused() bool { return tf.focused }

// Blur removes keyboard focus.
func (tf *TextField) Blur() { tf.focused = false }

// CapturesTab reports that plain Tab should move focus rather than insert text.
func (tf *TextField) CapturesTab() bool { return false }

// FocusOnClick reports that clicking the field should grant focus.
func (tf *TextField) FocusOnClick() bool { return true }

// FocusEl returns the element used for click-to-focus hit testing.
func (tf *TextField) FocusEl() *layout.Element { return tf.El }

func (tf *TextField) displayText() string {
	if tf.cfg.Password {
		return strings.Repeat("•", utf8.RuneCountInString(tf.Value))
	}
	return tf.Value
}

func (tf *TextField) padX() float32 { return tf.theme.Spacing.MNudge }

func (tf *TextField) iconSize() float32 { return tf.theme.Metrics.IconSizeSM }

func (tf *TextField) iconGap() float32 { return tf.theme.Spacing.SNudge }

func (tf *TextField) textLeft() float32 {
	x := tf.El.Frame.X + tf.padX()
	if tf.cfg.IconStart != "" {
		x += tf.iconSize() + tf.iconGap()
	}
	return x
}

func (tf *TextField) textRight() float32 {
	x := tf.El.Frame.X + tf.El.Frame.W - tf.padX()
	if tf.cfg.IconEnd != "" {
		x -= tf.iconSize() + tf.iconGap()
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
	style := tf.theme.Typography.Body
	tw, _ := tf.text.MeasureAt(tf.displayPrefixForCaret(), style.Size)
	return tf.textLeft() + tw
}

func (tf *TextField) setCaretFromX(px float32) {
	s := tf.displayText()
	x0 := tf.textLeft()
	ln := tf.text.Line(s)
	tf.caret = ln.ByteForX(px - x0)
	if tf.cfg.Password {
		// Map display byte offset back to Value byte offset by rune count.
		runes := 0
		for i := 0; i < tf.caret && i < len(s); {
			_, sz := utf8.DecodeRuneInString(s[i:])
			i += sz
			runes++
		}
		off := 0
		for count := 0; count < runes && off < len(tf.Value); count++ {
			_, sz := utf8.DecodeRuneInString(tf.Value[off:])
			off += sz
		}
		tf.caret = off
	}
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

func (tf *TextField) paint(dl *render.DrawList, _ *shape.Engine) {
	f := tf.El.Frame
	border := tf.theme.Border
	if tf.focused {
		border = tf.theme.FocusRing
	}
	dl.AddRoundedRectBorder(f, tf.cfg.Radius, tf.cfg.BorderWidth, tf.theme.Chrome, border)

	iconSz := tf.iconSize()
	iconY := f.Y + (f.H-iconSz)/2
	if tf.cfg.IconStart != "" {
		ix := f.X + tf.padX()
		tf.sheet.Draw(dl, tf.cfg.IconStart, render.Rect{X: ix, Y: iconY, W: iconSz, H: iconSz}, tf.theme.ForegroundMuted)
	}
	if tf.cfg.IconEnd != "" {
		ix := f.X + f.W - tf.padX() - iconSz
		tf.sheet.Draw(dl, tf.cfg.IconEnd, render.Rect{X: ix, Y: iconY, W: iconSz, H: iconSz}, tf.theme.ForegroundMuted)
	}

	tx := tf.textLeft()
	tr := tf.textRight()
	style := tf.theme.Typography.Body
	_, lh := tf.text.MeasureAt("Ag", style.Size)
	ty := f.Y + (f.H-lh)/2

	show := tf.displayText()
	col := tf.theme.Foreground
	if show == "" && !tf.focused {
		show = tf.cfg.Placeholder
		col = tf.theme.ForegroundMuted
	}
	if show != "" {
		dl.PushClip(render.Rect{X: tx, Y: f.Y, W: tr - tx, H: f.H})
		tf.text.DrawStringTopAt(dl, show, tx, ty, col, style.Size)
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
		dl.AddRect(render.Rect{X: cx, Y: ty, W: tf.theme.Stroke.Thick, H: lh}, tf.theme.Accent)
	}
}

func (tf *TextField) onMouse(e *layout.Element, m *input.Mouse) {
	if !e.Frame.Contains(m.X, m.Y) {
		return
	}
	if m.Pressed {
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
