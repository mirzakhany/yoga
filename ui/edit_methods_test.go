package ui

import (
	"testing"

	"github.com/yogaui/yoga/input"
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

func TestPasswordFieldNeverCopies(t *testing.T) {
	clip := &input.MemClipboard{}
	SetFrameResources(nil, nil, clip)
	defer SetFrameResources(nil, nil, nil)

	tf := NewTextInput(TextFieldConfig{Password: true})
	tf.Value = "hunter2"
	tf.Focus()
	clip.Set("before")

	// Nothing selected: Copy and Cut would otherwise take the whole value.
	tf.HandleKeys([]input.KeyEvent{{Key: input.KeyC, Mods: input.ModCtrl}})
	tf.HandleKeys([]input.KeyEvent{{Key: input.KeyX, Mods: input.ModCtrl}})
	tf.SelectAll()
	if tf.Copy() {
		t.Fatal("Copy reported a copy")
	}
	tf.Cut()
	if clip.Get() != "before" || tf.Value != "hunter2" {
		t.Fatalf("clip=%q value=%q", clip.Get(), tf.Value)
	}

	// Paste still works.
	tf.Paste()
	if tf.Value != "before" {
		t.Fatalf("paste into password: %q", tf.Value)
	}
}

func TestPasswordDoubleClickSelectsAll(t *testing.T) {
	c, _ := textFieldTestEnv(t)
	tf, el := layoutTextField(t, c, "pw", "two words", true)
	x := el.Frame.X + 12
	clickField(tf, el, x)
	tf.onMouse(el, &input.Mouse{X: x, Y: el.Frame.Y + el.Frame.H/2, Pressed: true, Down: true})
	if lo, hi := tf.selRange(); lo != 0 || hi != len("two words") {
		t.Fatalf("double-click selected %d..%d, want the whole password", lo, hi)
	}
}
