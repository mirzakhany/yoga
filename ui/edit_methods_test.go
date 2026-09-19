package ui

import (
	"testing"

	"github.com/mirzakhany/yoga/highlight"
	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
)

// The public edit methods back the right-click menu, which acts on a field
// whether or not it has keyboard focus.
func TestTextInputEditMethodsWorkWithoutFocus(t *testing.T) {
	clip := &input.MemClipboard{}
	SetFrameResources(nil, nil, clip)
	defer SetFrameResources(nil, nil, nil)

	tf := NewTextInput(TextFieldConfig{})
	var changed string
	tf.OnChange = func(s string) { changed = s }
	tf.Value = "hello world"

	if tf.HasSelection() {
		t.Fatal("HasSelection() = true before any selection")
	}
	// Nothing selected: Copy takes the whole value.
	if !tf.Copy() || clip.Get() != "hello world" {
		t.Fatalf("Copy with no selection: clipboard = %q", clip.Get())
	}

	tf.selAnchor, tf.caret = 6, 11
	if !tf.Copy() || clip.Get() != "world" {
		t.Fatalf("Copy selection: clipboard = %q", clip.Get())
	}
	tf.Cut()
	if tf.Value != "hello " || changed != "hello " || clip.Get() != "world" {
		t.Fatalf("Cut: value=%q changed=%q clip=%q", tf.Value, changed, clip.Get())
	}

	clip.Set("there\nsecond line")
	tf.Paste()
	if tf.Value != "hello there" || tf.caret != len("hello there") {
		t.Fatalf("Paste: value=%q caret=%d", tf.Value, tf.caret)
	}

	tf.SelectAll()
	if lo, hi := tf.selRange(); lo != 0 || hi != len(tf.Value) || !tf.HasSelection() {
		t.Fatalf("SelectAll: range %d..%d", lo, hi)
	}
	tf.Cut()
	if tf.Value != "" || clip.Get() != "hello there" {
		t.Fatalf("Cut all: value=%q clip=%q", tf.Value, clip.Get())
	}
	if tf.Copy() {
		t.Fatal("Copy of an empty field reported a copy")
	}
}

func TestTextInputEditMethodsIgnoredWhenDisabled(t *testing.T) {
	clip := &input.MemClipboard{}
	SetFrameResources(nil, nil, clip)
	defer SetFrameResources(nil, nil, nil)

	tf := NewTextInput(TextFieldConfig{})
	tf.Value = "keep"
	tf.disabled = true
	clip.Set("x")

	tf.SelectAll()
	tf.Cut()
	tf.Paste()
	if tf.Copy() || tf.Value != "keep" || tf.HasSelection() || clip.Get() != "x" {
		t.Fatalf("disabled field changed: value=%q clip=%q", tf.Value, clip.Get())
	}
}

func TestEditorEditMethods(t *testing.T) {
	text, err := shape.NewEngine(1, false)
	if err != nil {
		t.Skip(err)
	}
	clip := &input.MemClipboard{}
	SetFrameResources(text, render.NewSpriteSheet(text.Atlas), clip)
	defer SetFrameResources(nil, nil, nil)

	ed := NewEditor([]byte("hello world"), highlight.Noop{}, WithoutGutter())
	defer ed.Close()

	if ed.HasSelection() || ed.CanUndo() || ed.CanRedo() {
		t.Fatal("fresh editor reports a selection or history")
	}
	// Unlike TextInput, Copy with nothing selected leaves the clipboard alone.
	clip.Set("prev")
	if ed.Copy() || clip.Get() != "prev" {
		t.Fatalf("Copy with no selection: clipboard = %q", clip.Get())
	}

	ed.SelectAll()
	if !ed.HasSelection() {
		t.Fatal("SelectAll left no selection")
	}
	if !ed.Copy() || clip.Get() != "hello world" {
		t.Fatalf("Copy: clipboard = %q", clip.Get())
	}
	ed.Cut()
	if got := string(ed.Bytes()); got != "" || !ed.CanUndo() {
		t.Fatalf("Cut: content=%q canUndo=%v", got, ed.CanUndo())
	}

	clip.Set("pasted")
	ed.Paste()
	if got := string(ed.Bytes()); got != "pasted" {
		t.Fatalf("Paste: content=%q", got)
	}

	ed.Undo()
	if !ed.CanRedo() {
		t.Fatal("CanRedo() = false after Undo")
	}
	ed.Undo()
	if got := string(ed.Bytes()); got != "hello world" || ed.CanUndo() {
		t.Fatalf("after undoing both: content=%q canUndo=%v", got, ed.CanUndo())
	}
}

func TestEditorReadOnlyEditMethods(t *testing.T) {
	text, err := shape.NewEngine(1, false)
	if err != nil {
		t.Skip(err)
	}
	clip := &input.MemClipboard{}
	SetFrameResources(text, render.NewSpriteSheet(text.Atlas), clip)
	defer SetFrameResources(nil, nil, nil)

	ed := NewEditor([]byte("hello"), highlight.Noop{}, WithReadOnly(), WithoutGutter())
	defer ed.Close()

	clip.Set("pasted")
	ed.Paste()
	ed.SelectAll()
	ed.Cut()
	if got := string(ed.Bytes()); got != "hello" || clip.Get() != "hello" {
		t.Fatalf("read-only: content=%q clip=%q", got, clip.Get())
	}
	if ed.CanUndo() || ed.CanRedo() {
		t.Fatal("read-only editor reports undo history")
	}
}
