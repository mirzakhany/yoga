package ui

import (
	"testing"

	"github.com/mirzakhany/yoga/highlight"
	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
)

func TestEditorReadOnlyIgnoresEditsButCopies(t *testing.T) {
	text, err := shape.NewEngine(1, false)
	if err != nil {
		t.Skip(err)
	}
	clip := &input.MemClipboard{}
	SetFrameResources(text, render.NewSpriteSheet(text.Atlas), clip)

	ed := NewEditor([]byte("hello world"), highlight.Noop{}, WithReadOnly(), WithoutGutter())
	defer ed.Close()
	if !ed.ReadOnly() {
		t.Fatal("ReadOnly() = false")
	}

	ed.HandleText([]rune("x("))
	ed.HandleText([]rune{'('})
	ed.HandleKeys([]input.KeyEvent{{Key: input.KeyEnter}, {Key: input.KeyBackspace}, {Key: input.KeyTab}})

	// Select all and copy works; cut and paste leave the text alone.
	clip.Set("pasted")
	ed.HandleKeys([]input.KeyEvent{{Key: input.KeyV, Mods: input.ModCtrl}})
	ed.HandleKeys([]input.KeyEvent{{Key: input.KeyA, Mods: input.ModCtrl}})
	ed.HandleKeys([]input.KeyEvent{{Key: input.KeyX, Mods: input.ModCtrl}})
	if got := string(ed.Bytes()); got != "hello world" {
		t.Fatalf("content changed: %q", got)
	}
	if got := clip.Get(); got != "hello world" {
		t.Fatalf("clipboard = %q, want the selection", got)
	}
	ed.Undo()
	if got := string(ed.Bytes()); got != "hello world" || ed.Modified() {
		t.Fatalf("after undo: %q modified=%v", got, ed.Modified())
	}
	if ed.gutterW != 0 {
		t.Fatalf("gutterW = %v, want 0", ed.gutterW)
	}
}

func TestEditorRevisionCountsEdits(t *testing.T) {
	e := newTestEditor(t, []byte("a"))
	r0 := e.Revision()
	e.SetText("ab")
	e.Undo()
	if e.Revision() != r0+2 {
		t.Fatalf("revision %d, want %d", e.Revision(), r0+2)
	}
}

func TestApplyTextKeepsCaretOutsideTheChange(t *testing.T) {
	e := newTestEditor(t, []byte("package a\n\nfunc  f( ) {}\n\nvar x = 1\n"))
	tail := len("package a\n\nfunc  f( ) {}\n\nvar ")
	e.SetSelection(tail, tail)
	e.ApplyText("package a\n\nfunc f() {}\n\nvar x = 1\n")
	if got := string(e.Bytes()); got != "package a\n\nfunc f() {}\n\nvar x = 1\n" {
		t.Fatalf("text %q", got)
	}
	if want := tail - 2; e.Caret() != want {
		t.Fatalf("caret %d, want %d", e.Caret(), want)
	}
	e.Undo()
	if string(e.Bytes()) != "package a\n\nfunc  f( ) {}\n\nvar x = 1\n" {
		t.Fatal("ApplyText should undo in one step")
	}
}

// A formatter that only appends (gofmt adding the final newline) or only
// trims leaves one side a prefix of the other; ApplyText must not index past
// the shorter text.
func TestApplyTextPrefixEdits(t *testing.T) {
	for _, tc := range []struct{ from, to string }{
		{"package main", "package main\n"},
		{"package main\n\n", "package main\n"},
		{"", "package main\n"},
		{"package main\n", ""},
		{"héllo", "héllo wörld"},
	} {
		e := newTestEditor(t, []byte(tc.from))
		e.SetSelection(len(tc.from), len(tc.from))
		e.ApplyText(tc.to)
		if got := string(e.Bytes()); got != tc.to {
			t.Fatalf("%q -> %q: got %q", tc.from, tc.to, got)
		}
		e.Undo()
		if got := string(e.Bytes()); got != tc.from {
			t.Fatalf("%q -> %q: undo gave %q", tc.from, tc.to, got)
		}
	}
}
