package ui

import (
	"strings"
	"testing"
	"time"

	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
)

// Spans cover the text exactly: joining them with the newlines between source
// lines gives it back, spacing and all, and no line is wider than the limit.
func TestWrapSpansKeepSpacing(t *testing.T) {
	_, text := newParagraphCtx(t)
	s := "Traceback:\n    File \"x.py\", line 3\n" + strings.Repeat("word  ", 30) + "\n\n" + strings.Repeat("x", 80)
	spans := wrapSpans(text, s, 13, 0, 150)
	var b strings.Builder
	for i, sp := range spans {
		if i > 0 && spans[i-1].hi < sp.lo {
			b.WriteString(s[spans[i-1].hi:sp.lo])
		}
		line := s[sp.lo:sp.hi]
		b.WriteString(line)
		if w, _ := text.MeasureAt(strings.TrimRight(line, " "), 13); w > 150 {
			t.Fatalf("line %q is %v wide, max 150", line, w)
		}
	}
	if b.String() != s {
		t.Fatalf("spans lost text:\n got %q\nwant %q", b.String(), s)
	}
	if got := s[spans[1].lo:spans[1].hi]; got != `    File "x.py", line 3` {
		t.Fatalf("indented line = %q", got)
	}
}

type paragraphHarness struct {
	t    *testing.T
	c    *Ctx
	clip *input.MemClipboard
	body func(*Ctx) View
	root *layout.Element
}

func newParagraphHarness(t *testing.T, s string) *paragraphHarness {
	c := newTestCtx(t)
	clip := &input.MemClipboard{}
	c.SetClipboard(clip)
	h := &paragraphHarness{t: t, c: c, clip: clip}
	h.body = func(*Ctx) View { return Column(Paragraph(s).Selectable("p")).Padding(10) }
	h.frame(&input.Mouse{}, &input.Keyboard{})
	return h
}

// frame builds, paints and routes one frame of input, as the app loop does.
func (h *paragraphHarness) frame(m *input.Mouse, kb *input.Keyboard) {
	h.root = BuildFrame(h.c, h.body, 600, 400, m, kb)
	layout.Paint(h.root, &render.DrawList{}, h.c.Text())
	layout.Dispatch(h.root, m)
	h.c.Focus().HandleMouse(m)
	h.c.Focus().Route(kb)
	m.EndFrame()
	kb.EndFrame()
	h.c.EndFrame()
}

func (h *paragraphHarness) sel() *paragraphSelection {
	return h.c.Widget("p", nil).(*paragraphState).sel
}

// point returns the screen position of byte off, on its line's text baseline band.
func (h *paragraphHarness) point(off int) (float32, float32) {
	p := h.sel()
	for i, sp := range p.spans {
		if off >= sp.lo && off <= sp.hi {
			w, _ := h.c.Text().MeasureAtWeight(p.text[sp.lo:off], p.size, p.weight)
			return p.lineX(i) + w, p.top + float32(i)*p.lineH + p.lineH/2
		}
	}
	h.t.Fatalf("offset %d not on any line", off)
	return 0, 0
}

// drag presses at from, moves to to and releases. It is a fresh click, not
// the second of a double-click, however soon it follows the last one.
func (h *paragraphHarness) drag(from, to int) {
	h.sel().lastClick = time.Time{}
	m := &input.Mouse{}
	x, y := h.point(from)
	m.SetPos(x, y)
	m.SetButton(true)
	h.frame(m, &input.Keyboard{})
	x, y = h.point(to)
	m.SetPos(x, y)
	h.frame(m, &input.Keyboard{})
	m.SetButton(false)
	h.frame(m, &input.Keyboard{})
}

func (h *paragraphHarness) press(k input.Key) {
	kb := &input.Keyboard{}
	kb.PressKey(k, input.ModSuper)
	h.frame(&input.Mouse{}, kb)
}

func TestSelectableParagraphDragAndCopy(t *testing.T) {
	s := "Error: scripting is disabled\n    at line 3"
	h := newParagraphHarness(t, s)
	h.drag(7, 16)
	if !h.sel().Focused() {
		t.Fatal("paragraph not focused after the click")
	}
	h.press(input.KeyC)
	if got := h.clip.Get(); got != "scripting" {
		t.Fatalf("copied %q, want %q", got, "scripting")
	}

	// A drag down onto the next line takes the newline and indentation with it.
	h.drag(7, strings.Index(s, "at"))
	h.press(input.KeyC)
	if got := h.clip.Get(); got != "scripting is disabled\n    " {
		t.Fatalf("copied %q across lines", got)
	}

	h.press(input.KeyA)
	h.press(input.KeyC)
	if got := h.clip.Get(); got != s {
		t.Fatalf("select all copied %q", got)
	}
}

func TestSelectableParagraphDoubleClickSelectsWord(t *testing.T) {
	h := newParagraphHarness(t, "status_code is not available")
	m := &input.Mouse{}
	x, y := h.point(3)
	m.SetPos(x, y)
	for range 2 {
		m.SetButton(true)
		h.frame(m, &input.Keyboard{})
		m.SetButton(false)
		h.frame(m, &input.Keyboard{})
	}
	h.press(input.KeyC)
	if got := h.clip.Get(); got != "status_code" {
		t.Fatalf("double-click copied %q", got)
	}
}

// New text drops a selection made in the old one.
func TestSelectableParagraphResetsOnNewText(t *testing.T) {
	h := newParagraphHarness(t, "first text")
	h.drag(0, 5)
	h.body = func(*Ctx) View { return Column(Paragraph("other").Selectable("p")).Padding(10) }
	h.frame(&input.Mouse{}, &input.Keyboard{})
	if h.sel().HasSelection() {
		t.Fatal("selection survived a text change")
	}
}
