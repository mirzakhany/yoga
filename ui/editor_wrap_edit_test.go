package ui

import (
	"strings"
	"testing"

	"github.com/mirzakhany/yoga/highlight"
	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
)

// TestEditorSoftWrapEnterAtEnd types Enter on the last line of a wrapped
// document. The edit adds a line before the next layout re-wraps, and keeping
// the caret visible must not index the stale wrap tables.
func TestEditorSoftWrapEnterAtEnd(t *testing.T) {
	c := newTestCtx(t)
	doc := strings.Repeat("{\"key\": \"a fairly long value that wraps\"},\n", 24) + "}"
	ed := NewEditor([]byte(doc), highlight.Noop{}, WithSoftWrap(true))
	defer ed.Close()
	body := func(cc *Ctx) View { return Column(ViewOf(ed).Grow(1)).Grow(1) }
	frame := func() {
		root := BuildFrame(c, body, 300, 200, &input.Mouse{X: -1, Y: -1}, &input.Keyboard{})
		layout.Paint(root, &render.DrawList{}, c.Text())
		c.EndFrame()
	}
	frame()
	frame()
	ed.moveTo(ed.pt.Len(), false)
	ed.HandleKeys([]input.KeyEvent{{Key: input.KeyEnter}})
	ed.HandleText([]rune{'x'})
	ed.HandleKeys([]input.KeyEvent{{Key: input.KeyUp}, {Key: input.KeyEnter}})
	frame()
	if got := ed.pt.LineCount(); got != 27 {
		t.Fatalf("line count = %d, want 27", got)
	}
	// The deferred reveal scrolled the caret into view on that layout.
	_, clientH, _, _ := ed.scrollMetrics()
	top := float32(ed.rowOfByte(ed.caret)) * ed.lineH
	if top < ed.ScrollPx || top+ed.lineH > ed.ScrollPx+clientH {
		t.Fatalf("caret row at %v not in view [%v, %v]", top, ed.ScrollPx, ed.ScrollPx+clientH)
	}
}
