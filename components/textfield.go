package components

import (
	"strings"
	"time"
	"unicode/utf8"

	"github.com/mirzakhany/yoga"
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
	cfg   TextFieldConfig

	Value    string
	OnChange func(value string)

	focused    bool
	caret      int // byte offset
	caretShown bool
	blinkStart time.Time
	// scrollX shifts the text left so the caret stays inside the viewport when
	// the content is wider than the field.
	scrollX float32
}

// NewTextField builds a text field with the given configuration.
func NewTextField(cfg TextFieldConfig) *TextField {
	th := theme.Current()
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

func (tf *TextField) padX() float32 { return theme.Current().Spacing.MNudge }

func (tf *TextField) iconSize() float32 { return theme.Current().Metrics.IconSizeSM }

func (tf *TextField) iconGap() float32 { return theme.Current().Spacing.SNudge }

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

// caretOffsetX is the caret's x position relative to the start of the text
// (before applying scrollX).
func (tf *TextField) caretOffsetX() float32 {
	th := theme.Current()
	style := th.Typography.Body
	tw, _ := yoga.Text().MeasureAt(tf.displayPrefixForCaret(), style.Size)
	return tw
}

func (tf *TextField) caretX() float32 {
	return tf.textLeft() - tf.scrollX + tf.caretOffsetX()
}

// ensureCaretVisible adjusts scrollX so the caret sits inside the text
// viewport. Call when the frame is valid (paint time).
func (tf *TextField) ensureCaretVisible() {
	th := theme.Current()
	style := th.Typography.Body
	viewW := tf.textRight() - tf.textLeft()
	if viewW <= 0 {
		tf.scrollX = 0
		return
	}
	fullW, _ := yoga.Text().MeasureAt(tf.displayText(), style.Size)
	caretPad := th.Stroke.Thick
	maxScroll := f32max(0, fullW+caretPad-viewW)
	tf.scrollX = clampf(tf.scrollX, 0, maxScroll)

	cx := tf.caretOffsetX()
	if cx-tf.scrollX < 0 {
		tf.scrollX = cx
	} else if cx-tf.scrollX > viewW-caretPad {
		tf.scrollX = cx - viewW + caretPad
	}
	tf.scrollX = clampf(tf.scrollX, 0, maxScroll)
}

func (tf *TextField) setCaretFromX(px float32) {
	s := tf.displayText()
	th := theme.Current()
	style := th.Typography.Body
	x0 := tf.textLeft() - tf.scrollX
	// Shape at the same logical size used for painting so click-to-caret
	// matches glyph positions when the theme body size is not the default.
	ln := yoga.Text().LineAt(s, style.Size)
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
		tf.clampCaret()
		return
	}
	tf.Value = s
	tf.clampCaret()
	if tf.OnChange != nil {
		tf.OnChange(s)
	}
}

func (tf *TextField) clampCaret() {
	if tf.caret < 0 {
		tf.caret = 0
	}
	if tf.caret > len(tf.Value) {
		tf.caret = len(tf.Value)
	}
}

func (tf *TextField) insertAtCaret(s string) {
	if s == "" {
		return
	}
	if i := strings.IndexAny(s, "\n\r"); i >= 0 {
		s = s[:i]
	}
	if s == "" {
		return
	}
	tf.clampCaret()
	tf.setValue(tf.Value[:tf.caret] + s + tf.Value[tf.caret:])
	tf.caret += len(s)
	tf.blinkStart = time.Now()
	tf.caretShown = true
}

func (tf *TextField) paint(dl *render.DrawList, _ *shape.Engine) {
	th := theme.Current()
	text := yoga.Text()
	sheet := yoga.Icons()
	f := tf.El.Frame
	border := th.Border
	if tf.focused {
		border = th.FocusRing
	}
	dl.AddRoundedRectBorder(f, tf.cfg.Radius, tf.cfg.BorderWidth, th.Chrome, border)

	iconSz := tf.iconSize()
	iconY := f.Y + (f.H-iconSz)/2
	if tf.cfg.IconStart != "" {
		ix := f.X + tf.padX()
		sheet.Draw(dl, tf.cfg.IconStart, render.Rect{X: ix, Y: iconY, W: iconSz, H: iconSz}, th.ForegroundMuted)
	}
	if tf.cfg.IconEnd != "" {
		ix := f.X + f.W - tf.padX() - iconSz
		sheet.Draw(dl, tf.cfg.IconEnd, render.Rect{X: ix, Y: iconY, W: iconSz, H: iconSz}, th.ForegroundMuted)
	}

	tx := tf.textLeft()
	tr := tf.textRight()
	style := th.Typography.Body
	_, lh := text.MeasureAt("Ag", style.Size)
	ty := f.Y + (f.H-lh)/2

	// Keep the caret inside the viewport by scrolling the text horizontally.
	tf.ensureCaretVisible()

	show := tf.displayText()
	col := th.Foreground
	if show == "" && !tf.focused {
		show = tf.cfg.Placeholder
		col = th.ForegroundMuted
	}
	if show != "" {
		dl.PushClip(render.Rect{X: tx, Y: f.Y, W: tr - tx, H: f.H})
		text.DrawStringTopAt(dl, show, tx-tf.scrollX, ty, col, style.Size)
		dl.PopClip()
	}

	if tf.focused && tf.caretShown {
		cx := clampf(tf.caretX(), tx, tr)
		dl.AddRect(render.Rect{X: cx, Y: ty, W: th.Stroke.Thick, H: lh}, th.Accent)
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

// ── Builder/modifier methods ─────────────────────────────────────────────────

// Changed sets the OnChange callback.
func (tf *TextField) Changed(fn func(string)) *TextField { tf.OnChange = fn; return tf }

// WithIconStart sets the leading icon (name must exist in the sprite sheet).
func (tf *TextField) WithIconStart(name string) *TextField { tf.cfg.IconStart = name; return tf }

// WithIconEnd sets the trailing icon.
func (tf *TextField) WithIconEnd(name string) *TextField { tf.cfg.IconEnd = name; return tf }

// AsPassword enables password masking (displays bullets instead of characters).
func (tf *TextField) AsPassword() *TextField { tf.cfg.Password = true; return tf }

// HandleKeys processes navigation and editing keys for this frame.
func (tf *TextField) HandleKeys(keys []input.KeyEvent) {
	if !tf.focused {
		return
	}
	clip := yoga.Clipboard()
	for _, ev := range keys {
		if ev.Mods.Primary() {
			switch ev.Key {
			case input.KeyA:
				tf.caret = len(tf.Value)
			case input.KeyC:
				if clip != nil && tf.Value != "" {
					clip.Set(tf.Value)
				}
			case input.KeyX:
				if clip != nil && tf.Value != "" {
					clip.Set(tf.Value)
					tf.setValue("")
					tf.caret = 0
				}
			case input.KeyV:
				if clip != nil {
					tf.insertAtCaret(clip.Get())
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
