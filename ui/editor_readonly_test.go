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
