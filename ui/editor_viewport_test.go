package ui

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/mirzakhany/yoga/highlight"
	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
)

func newTestCtx(t *testing.T) *Ctx {
	t.Helper()
	text, err := shape.NewEngine(1, false)
	if err != nil {
		t.Skip(err)
	}
	SetFrameResources(text, render.NewSpriteSheet(text.Atlas), &input.MemClipboard{})
	c := New(text, NewFocusScope(), nil)
	c.SetIcons(render.NewSpriteSheet(text.Atlas))
	c.SetClipboard(&input.MemClipboard{})
	return c
}

func bigJSONDoc(lines int) []byte {
	var b strings.Builder
	b.WriteString("{\n  \"items\": [\n")
	for i := 0; i < lines; i++ {
		fmt.Fprintf(&b, "    {\"id\": %d, \"name\": \"item-%06d\", \"ok\": %v},\n", i, i, i%2 == 0)
	}
	b.WriteString("    {}\n  ]\n}\n")
	return []byte(b.String())
}

// frameUntilTokens lays out repeatedly until the editor has picked up tokens.
func frameUntilTokens(t *testing.T, c *Ctx, ed *Editor) {
	t.Helper()
	body := func(cc *Ctx) View { return Column(ViewOf(ed).Grow(1)).Grow(1) }
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		root := BuildFrame(c, body, 1000, 600, &input.Mouse{}, &input.Keyboard{})
		layout.Paint(root, &render.DrawList{}, c.Text())
		c.EndFrame()
		if len(ed.tokens) > 0 {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("editor never received tokens")
}

// coloredInVisibleRows reports how many of the rows on screen have at least one
// non-default colored byte.
func coloredInVisibleRows(ed *Editor) (colored, total int) {
	first := int(ed.ScrollPx / ed.lineH)
	rows := int(ed.contentViewport().H/ed.lineH) + 1
	for r := first; r < first+rows && r < ed.pt.LineCount(); r++ {
		total++
		start := ed.pt.LineStart(r)
		end := start + ed.pt.LineLen(r)
		for off := start; off < end; off++ {
			if ed.colorAt(off) != highlight.ClassDefault {
				colored++
				break
			}
		}
	}
	return colored, total
}

// TestEditorHighlightsVisibleRowsAfterScroll is the integration contract: the
// editor asks the highlighter only for what is on screen, so scrolling to a new
// part of a large document must still colour it.
func TestEditorHighlightsVisibleRowsAfterScroll(t *testing.T) {
	orig := highlight.MaxBytes
	highlight.MaxBytes = 1 << 30
	defer func() { highlight.MaxBytes = orig }()

	c := newTestCtx(t)
	src := bigJSONDoc(40000)
	ed := NewEditor(src, highlight.NewJSON())
	defer ed.Close()

	frameUntilTokens(t, c, ed)

	colored, total := coloredInVisibleRows(ed)
	if total == 0 {
		t.Fatal("no visible rows")
	}
	if colored < total/2 {
		t.Errorf("at the top: only %d of %d visible rows coloured", colored, total)
	}
	// The whole point: the editor holds tokens for a window, not the document.
	// Compare against a highlighter that was never given a range.
	whole := highlight.NewJSON()
	defer whole.Close()
	whole.Update(src)
	var wholeCount int
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if toks, ok := whole.Poll(); ok {
			wholeCount = len(toks)
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if wholeCount == 0 {
		t.Fatal("whole-document highlighter produced nothing to compare against")
	}
	topTokens := len(ed.tokens)
	if topTokens > wholeCount/10 {
		t.Errorf("editor holds %d tokens; the whole document has %d — the range is not being scoped",
			topTokens, wholeCount)
	}
	t.Logf("editor holds %d tokens; whole document has %d (%.1f%%)",
		topTokens, wholeCount, 100*float64(topTokens)/float64(wholeCount))

	// Scroll deep into the document and let the editor re-request.
	for _, frac := range []float64{0.25, 0.5, 0.9} {
		ed.ScrollPx = float32(float64(ed.pt.LineCount())*frac) * ed.lineH
		frameUntilVisibleColoured(t, c, ed, frac)
	}
}

func frameUntilVisibleColoured(t *testing.T, c *Ctx, ed *Editor, frac float64) {
	t.Helper()
	body := func(cc *Ctx) View { return Column(ViewOf(ed).Grow(1)).Grow(1) }
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		root := BuildFrame(c, body, 1000, 600, &input.Mouse{}, &input.Keyboard{})
		layout.Paint(root, &render.DrawList{}, c.Text())
		c.EndFrame()
		colored, total := coloredInVisibleRows(ed)
		if total > 0 && colored >= total/2 {
			t.Logf("scrolled to %.0f%%: %d/%d visible rows coloured, %d tokens held",
				frac*100, colored, total, len(ed.tokens))
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	colored, total := coloredInVisibleRows(ed)
	t.Fatalf("after scrolling to %.0f%%: only %d of %d visible rows coloured", frac*100, colored, total)
}

// TestEditorNoopHighlighterStillWorks guards the fallback path.
func TestEditorNoopHighlighterStillWorks(t *testing.T) {
	c := newTestCtx(t)
	ed := NewEditor(bigJSONDoc(100), highlight.Noop{})
	defer ed.Close()

	body := func(cc *Ctx) View { return Column(ViewOf(ed).Grow(1)).Grow(1) }
	root := BuildFrame(c, body, 1000, 600, &input.Mouse{}, &input.Keyboard{})
	layout.Paint(root, &render.DrawList{}, c.Text())
	if len(ed.tokens) != 0 {
		t.Errorf("Noop highlighter produced %d tokens", len(ed.tokens))
	}
}
