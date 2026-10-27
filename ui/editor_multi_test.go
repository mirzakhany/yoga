package ui

import (
	"testing"

	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/render"
)

func key(k input.Key, mods input.Mod) []input.KeyEvent {
	return []input.KeyEvent{{Key: k, Mods: mods}}
}

func TestMultiCursorTypingAndUndo(t *testing.T) {
	e := newTestEditor(t, []byte("a\nb\nc\n"))
	e.SetSelection(0, 0)
	e.AddCursorLine(1)
	e.AddCursorLine(1)
	if e.Cursors() != 3 {
		t.Fatalf("cursors %d, want 3", e.Cursors())
	}
	e.HandleText([]rune("x"))
	e.HandleText([]rune("y"))
	if got := string(e.Bytes()); got != "xya\nxyb\nxyc\n" {
		t.Fatalf("typed %q", got)
	}
	e.HandleKeys(key(input.KeyBackspace, 0))
	if got := string(e.Bytes()); got != "xa\nxb\nxc\n" {
		t.Fatalf("backspace %q", got)
	}
	e.Undo()
	if got := string(e.Bytes()); got != "xya\nxyb\nxyc\n" {
		t.Fatalf("one undo should restore every cursor's edit: %q", got)
	}
	if e.Cursors() != 1 {
		t.Fatal("undo should leave one caret")
	}
}

func TestAddNextOccurrenceAndReplace(t *testing.T) {
	e := newTestEditor(t, []byte("foo bar foo baz foo"))
	e.SetSelection(1, 1)
	e.AddNextOccurrence() // selects the word
	e.AddNextOccurrence()
	e.AddNextOccurrence()
	if e.Cursors() != 3 {
		t.Fatalf("cursors %d, want 3: %v", e.Cursors(), e.Selections())
	}
	e.HandleText([]rune("qux"))
	if got := string(e.Bytes()); got != "qux bar qux baz qux" {
		t.Fatalf("got %q", got)
	}
	e.HandleKeys(key(input.KeyEscape, 0))
	if e.Cursors() != 1 {
		t.Fatal("escape should drop the extra cursors")
	}
}

func TestSelectAllOccurrencesAndMove(t *testing.T) {
	e := newTestEditor(t, []byte("ab ab ab"))
	e.SetSelection(0, 2)
	e.SelectAllOccurrences()
	if s := e.Selections(); len(s) != 3 || s[2] != [2]int{6, 8} {
		t.Fatalf("selections %v", s)
	}
	e.HandleKeys(key(input.KeyRight, 0))
	if s := e.Selections(); len(s) != 3 || s[0] != [2]int{3, 3} || s[1] != [2]int{6, 6} {
		t.Fatalf("after right: %v", s)
	}
	// Cursors that meet merge.
	e.HandleKeys(key(input.KeyEnd, 0))
	if e.Cursors() != 1 {
		t.Fatalf("cursors on one line moved to its end should merge: %v", e.Selections())
	}
}

func TestMultiCursorCopyPasteSplitsLines(t *testing.T) {
	e := newTestEditor(t, []byte("one\ntwo\n"))
	clip := &input.MemClipboard{}
	SetFrameResources(frameText(), render.NewSpriteSheet(frameText().Atlas), clip)
	e.SetSelection(0, 3)
	e.extra = []cursor{{caret: 7, anchor: 4}}
	e.HandleKeys(key(input.KeyC, input.ModSuper))
	if got := clip.Get(); got != "one\ntwo" {
		t.Fatalf("copied %q", got)
	}
	e.SetText("A\nB\n")
	e.SetSelection(1, 1)
	e.AddCursorLine(1)
	e.HandleKeys(key(input.KeyV, input.ModSuper))
	if got := string(e.Bytes()); got != "Aone\nBtwo\n" {
		t.Fatalf("paste %q", got)
	}
}

func TestEditorFiltersAndUndoGroup(t *testing.T) {
	e := newTestEditor(t, []byte("hello world"))
	var keys []input.Key
	e.TextFilter = func(r []rune) bool { return r[0] == 'x' } // eat x
	e.KeyFilter = func(ev input.KeyEvent) bool {
		keys = append(keys, ev.Key)
		return ev.Key == input.KeyBackspace
	}
	e.SetSelection(5, 5)
	e.HandleText([]rune("x"))
	e.HandleText([]rune("!"))
	e.HandleKeys(key(input.KeyBackspace, 0))
	if got := string(e.Bytes()); got != "hello! world" {
		t.Fatalf("filters: %q", got)
	}
	if len(keys) != 1 {
		t.Fatalf("key filter saw %v", keys)
	}
	e.UndoGroup(func() {
		e.Replace(0, 5, "HELLO")
		e.Replace(7, 12, "WORLD")
	})
	if got := string(e.Bytes()); got != "HELLO! WORLD" {
		t.Fatalf("replace: %q", got)
	}
	e.Undo()
	if got := string(e.Bytes()); got != "hello! world" {
		t.Fatalf("one undo for the group: %q", got)
	}
}
