package ui

import (
	"testing"

	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
)

func inlineEditor(t *testing.T, content string) *Editor {
	t.Helper()
	e := newTestEditor(t, []byte(content))
	e.Focus()
	e.SetSelection(len(content), len(content))
	return e
}

func TestEditorInlineSuggestionTabAccepts(t *testing.T) {
	e := inlineEditor(t, "func add(a, b int) int {\n\t")
	var accepted string
	e.OnInlineAccept = func(s string) { accepted = s }
	e.SetInlineSuggestion("return a + b\n}")
	if got := e.InlineSuggestion(); got != "return a + b\n}" {
		t.Fatalf("suggestion: %q", got)
	}
	e.HandleKeys([]input.KeyEvent{{Key: input.KeyTab}})
	if got := string(e.Bytes()); got != "func add(a, b int) int {\n\treturn a + b\n}" {
		t.Fatalf("after Tab: %q", got)
	}
	if accepted != "return a + b\n}" || e.InlineSuggestion() != "" {
		t.Fatalf("accepted %q, still showing %q", accepted, e.InlineSuggestion())
	}
	e.Undo()
	if got := string(e.Bytes()); got != "func add(a, b int) int {\n\t" {
		t.Fatalf("one undo should drop the whole suggestion: %q", got)
	}
}

func TestEditorInlineSuggestionTabWithoutOneIndents(t *testing.T) {
	e := inlineEditor(t, "x")
	e.HandleKeys([]input.KeyEvent{{Key: input.KeyTab}})
	if got := string(e.Bytes()); got != "x\t" {
		t.Fatalf("Tab with no suggestion: %q", got)
	}
}

func TestEditorInlineSuggestionTypingThroughKeepsRest(t *testing.T) {
	e := inlineEditor(t, "fmt.")
	e.SetInlineSuggestion("Println")
	typeInEditor(e, "Pri")
	if got := e.InlineSuggestion(); got != "ntln" {
		t.Fatalf("after typing its start: %q", got)
	}
	typeInEditor(e, "x")
	if got := e.InlineSuggestion(); got != "" {
		t.Fatalf("typing something else should drop it: %q", got)
	}
}

func TestEditorInlineSuggestionDroppedByCaretAndEscape(t *testing.T) {
	e := inlineEditor(t, "abc")
	e.SetInlineSuggestion("def")
	e.HandleKeys([]input.KeyEvent{{Key: input.KeyLeft}})
	if e.InlineSuggestion() != "" {
		t.Fatal("moving the caret should drop the suggestion")
	}
	e.HandleKeys([]input.KeyEvent{{Key: input.KeyRight}})
	e.SetInlineSuggestion("def")
	e.HandleKeys([]input.KeyEvent{{Key: input.KeyEscape}})
	if e.InlineSuggestion() != "" {
		t.Fatal("Escape should drop the suggestion")
	}
	e.SetInlineSuggestion("def")
	e.SetText("other")
	if e.InlineSuggestion() != "" {
		t.Fatal("replacing the text should drop the suggestion")
	}
}

func TestEditorInlineSuggestionReadOnly(t *testing.T) {
	e := inlineEditor(t, "abc")
	e.readOnly = true
	e.SetInlineSuggestion("def")
	if e.InlineSuggestion() != "" {
		t.Fatal("a read-only editor shows no suggestion")
	}
}

func TestEditorInlineSuggestionPaints(t *testing.T) {
	for _, wrap := range []bool{false, true} {
		e := inlineEditor(t, "a := f(\nnext line")
		e.SetSoftWrap(wrap)
		e.SetSelection(len("a := f("), len("a := f("))
		c := New(frameText(), NewFocusScope(), nil)
		c.BeginFrame(900, 600, nil, nil)
		e.Layout(c)
		root := layout.New(layout.Box(), e.host)
		root.Calculate(900, 600)
		e.Update(nil)

		plain := &render.DrawList{}
		e.paint(plain, frameText())
		e.SetInlineSuggestion("x,\n\ty)")
		ghost := &render.DrawList{}
		e.paint(ghost, frameText())
		if len(ghost.Vertices) <= len(plain.Vertices) {
			t.Fatalf("wrap=%v: ghost text added no geometry (%d vs %d vertices)", wrap, len(ghost.Vertices), len(plain.Vertices))
		}
	}
}
