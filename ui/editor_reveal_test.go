package ui

import (
	"strings"
	"testing"

	"github.com/mirzakhany/yoga/layout"
)

// editorFrames lays the editor out at 900x600 for n frames.
func editorFrames(e *Editor, n int) {
	c := New(frameText(), NewFocusScope(), nil)
	for range n {
		c.BeginFrame(900, 600, nil, nil)
		root := layout.New(layout.Box(), e.Layout(c))
		root.Calculate(900, 600)
		c.EndFrame()
	}
}

func TestEditorCenterCaretBeforeLayout(t *testing.T) {
	e := newTestEditor(t, []byte(strings.Repeat("line\n", 400)))
	off := e.pt.LineStart(300)
	e.SetSelection(off, off)
	e.CenterCaret()
	editorFrames(e, 3)

	_, clientH, _, _ := e.scrollMetrics()
	if clientH <= 0 {
		t.Fatalf("editor not laid out")
	}
	want := 300*e.lineH + e.lineH/2 - clientH/2
	if d := e.ScrollPx - want; d > 1 || d < -1 {
		t.Fatalf("ScrollPx %v, want caret row centered at %v", e.ScrollPx, want)
	}
}

func TestEditorScrollPastEnd(t *testing.T) {
	withEditorDefaults(t)
	e := newTestEditor(t, []byte(strings.Repeat("line\n", 99)))
	editorFrames(e, 2)
	textH := 100 * e.lineH
	if e.ContentHeight != textH {
		t.Fatalf("ContentHeight %v, want %v without ScrollPastEnd", e.ContentHeight, textH)
	}

	EditorDefaults.ScrollPastEnd = true
	editorFrames(e, 2)
	want := textH + e.viewport.Frame.H - e.lineH
	if e.ContentHeight != want {
		t.Fatalf("ContentHeight %v, want %v with ScrollPastEnd", e.ContentHeight, want)
	}
	e.ScrollPx = 1e9
	e.clampScroll()
	_, clientH, _, _ := e.scrollMetrics()
	if top := e.ScrollPx + clientH - e.pastEndH; top < textH-e.lineH {
		t.Fatalf("cannot scroll the last line to the top: ScrollPx=%v", e.ScrollPx)
	}
}
