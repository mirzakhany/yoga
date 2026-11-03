package ui

import (
	"testing"

	"github.com/mirzakhany/yoga/input"
)

// press runs one key through the editor's keymap with the platform set.
func pressOn(e *Editor, mac bool, k input.Key, mods input.Mod) {
	defer func(m bool) { editorMac = m }(editorMac)
	editorMac = mac
	e.HandleKeys(key(k, mods))
}

func TestWordMotion(t *testing.T) {
	e := newUndoEditor(t, "foo.bar(baz)  qux\nnext")
	stops := []int{3, 4, 7, 8, 11, 12, 17, 18, 22}
	e.SetSelection(0, 0)
	for _, want := range stops {
		pressOn(e, true, input.KeyRight, input.ModAlt)
		if e.Caret() != want {
			t.Fatalf("⌥→ stopped at %d, want %d", e.Caret(), want)
		}
	}
	// Back from the end: next → line break → qux → ) → baz → ( → bar → . → foo.
	e.SetSelection(22, 22)
	for _, want := range []int{18, 17, 14, 11, 8, 7, 4, 3, 0} {
		pressOn(e, true, input.KeyLeft, input.ModAlt)
		if e.Caret() != want {
			t.Fatalf("⌥← stopped at %d, want %d", e.Caret(), want)
		}
	}
	// Shift extends.
	e.SetSelection(0, 0)
	pressOn(e, true, input.KeyRight, input.ModAlt|input.ModShift)
	if lo, hi := e.Selection(); lo != 0 || hi != 3 {
		t.Fatalf("⌥⇧→ selected %d..%d", lo, hi)
	}
}

func TestDeleteWord(t *testing.T) {
	e := newUndoEditor(t, "alpha beta gamma")
	e.SetSelection(10, 10)
	pressOn(e, true, input.KeyBackspace, input.ModAlt)
	if got := string(e.Bytes()); got != "alpha  gamma" {
		t.Fatalf("⌥⌫: %q", got)
	}
	pressOn(e, false, input.KeyDelete, input.ModCtrl)
	if got := string(e.Bytes()); got != "alpha " {
		t.Fatalf("Ctrl+Del: %q", got)
	}
	pressOn(e, true, input.KeyBackspace, input.ModSuper)
	if got := string(e.Bytes()); got != "" {
		t.Fatalf("⌘⌫: %q", got)
	}
}

func TestSmartHomeAndEnd(t *testing.T) {
	e := newUndoEditor(t, "a\n    indented line\n")
	e.SetSelection(12, 12)
	pressOn(e, false, input.KeyHome, 0)
	if e.Caret() != 6 {
		t.Fatalf("Home went to %d, want first non-blank 6", e.Caret())
	}
	pressOn(e, false, input.KeyHome, 0)
	if e.Caret() != 2 {
		t.Fatalf("second Home went to %d, want column 0 (2)", e.Caret())
	}
	pressOn(e, true, input.KeyLeft, input.ModSuper)
	if e.Caret() != 6 {
		t.Fatalf("⌘← toggled to %d, want 6", e.Caret())
	}
	pressOn(e, true, input.KeyRight, input.ModSuper|input.ModShift)
	if lo, hi := e.Selection(); lo != 6 || hi != 19 {
		t.Fatalf("⌘⇧→ selected %d..%d, want 6..19", lo, hi)
	}
	pressOn(e, false, input.KeyEnd, 0)
	if e.Caret() != 19 || e.HasSelection() {
		t.Fatalf("End at %d", e.Caret())
	}
}

func TestFileStartEnd(t *testing.T) {
	e := newUndoEditor(t, "one\ntwo\nthree")
	e.SetSelection(5, 5)
	pressOn(e, true, input.KeyDown, input.ModSuper)
	if e.Caret() != 13 {
		t.Fatalf("⌘↓ to %d", e.Caret())
	}
	pressOn(e, false, input.KeyHome, input.ModCtrl|input.ModShift)
	if lo, hi := e.Selection(); lo != 0 || hi != 13 {
		t.Fatalf("Ctrl+Shift+Home selected %d..%d", lo, hi)
	}
}

func TestStickyColumn(t *testing.T) {
	e := newUndoEditor(t, "long line here\nab\nanother long one\n")
	e.SetSelection(10, 10)
	e.HandleKeys(key(input.KeyDown, 0))
	if e.Caret() != 17 {
		t.Fatalf("down onto the short line: %d, want its end 17", e.Caret())
	}
	e.HandleKeys(key(input.KeyDown, 0))
	if e.Caret() != 28 {
		t.Fatalf("down again kept column: %d, want 28", e.Caret())
	}
	// A horizontal move resets the remembered column.
	e.HandleKeys(key(input.KeyLeft, 0))
	e.HandleKeys(key(input.KeyUp, 0))
	if e.Caret() != 17 {
		t.Fatalf("up after ←: %d, want 17", e.Caret())
	}
	// So does typing.
	e.SetSelection(1, 1)
	e.HandleKeys(key(input.KeyDown, 0))
	e.HandleText([]rune("xyzw"))
	e.HandleKeys(key(input.KeyUp, 0))
	if e.Caret() != 5 {
		t.Fatalf("up after typing: %d, want the new column 5", e.Caret())
	}
}

func TestMotionAtEveryCursor(t *testing.T) {
	e := newUndoEditor(t, "aa bb\ncc dd\n")
	e.SetSelection(0, 0)
	e.AddCursorAt(6)
	pressOn(e, true, input.KeyRight, input.ModAlt)
	if s := e.Selections(); len(s) != 2 || s[0][0] != 2 || s[1][0] != 8 {
		t.Fatalf("⌥→ at each cursor: %v", s)
	}
	pressOn(e, true, input.KeyBackspace, input.ModAlt)
	if got := string(e.Bytes()); got != " bb\n dd\n" {
		t.Fatalf("⌥⌫ at each cursor: %q", got)
	}
	e.Undo()
	if got := string(e.Bytes()); got != "aa bb\ncc dd\n" {
		t.Fatalf("one undo: %q", got)
	}
}

func TestArrowsCrossLineBreaks(t *testing.T) {
	e := newUndoEditor(t, "ab\ncd\n")
	e.SetSelection(2, 2)
	e.HandleKeys(key(input.KeyRight, 0))
	if e.Caret() != 3 {
		t.Fatalf("→ at a line end went to %d, want the next line 3", e.Caret())
	}
	e.HandleKeys(key(input.KeyLeft, 0))
	if e.Caret() != 2 {
		t.Fatalf("← at a line start went to %d, want 2", e.Caret())
	}
	e.HandleKeys(key(input.KeyLeft, input.ModShift))
	if lo, hi := e.Selection(); lo != 1 || hi != 2 {
		t.Fatalf("⇧← selected %d..%d", lo, hi)
	}
}
