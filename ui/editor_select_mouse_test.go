package ui

import (
	"testing"
	"time"

	"github.com/mirzakhany/yoga/input"
)

// pointAt is the window point over offset off's left edge, mid-row.
func pointAt(ed *Editor, off int) (float32, float32) {
	f := ed.viewport.Frame
	ln := ed.lineOf(off)
	x := f.X + ed.gutterW + ed.textPad - ed.ScrollX + ed.caretXInLine(ln, off) + 1
	y := f.Y + float32(ln)*ed.lineH - ed.ScrollPx + ed.lineH/2
	return x, y
}

func (h *findHarness) clickMods(x, y float32, mods input.Mod) {
	h.frame(&input.Mouse{X: x, Y: y, Mods: mods}, nil)
	h.frame(&input.Mouse{X: x, Y: y, Down: true, Pressed: true, Mods: mods}, nil)
	h.frame(&input.Mouse{X: x, Y: y, Released: true, Mods: mods}, nil)
	time.Sleep(doubleClickInterval) // keep clicks from counting as double
}

func TestShiftClickExtends(t *testing.T) {
	h := newFindHarness(t, "alpha beta\ngamma delta\nepsilon\n")
	ed := h.ed
	x, y := pointAt(ed, 2)
	h.clickMods(x, y, 0)
	x, y = pointAt(ed, 14)
	h.clickMods(x, y, input.ModShift)
	if lo, hi := ed.Selection(); lo != 2 || hi != 14 {
		t.Fatalf("⇧-click selected %d..%d, want 2..14", lo, hi)
	}
	// A second ⇧-click moves the same selection's end.
	x, y = pointAt(ed, 25)
	h.clickMods(x, y, input.ModShift)
	if lo, hi := ed.Selection(); lo != 2 || hi != 25 {
		t.Fatalf("second ⇧-click selected %d..%d, want 2..25", lo, hi)
	}
	x, y = pointAt(ed, 1)
	h.clickMods(x, y, input.ModShift)
	if lo, hi := ed.Selection(); lo != 1 || hi != 2 {
		t.Fatalf("⇧-click before the anchor selected %d..%d, want 1..2", lo, hi)
	}
}

func TestBoxSelection(t *testing.T) {
	h := newFindHarness(t, "abcdef\nab\nabcdefgh\n")
	ed := h.ed
	ax, ay := pointAt(ed, 1)    // line 0, column 1
	bx, by := pointAt(ed, 10+4) // line 2, column 4
	mods := input.ModAlt | input.ModShift
	h.frame(&input.Mouse{X: ax, Y: ay, Mods: mods}, nil)
	h.frame(&input.Mouse{X: ax, Y: ay, Down: true, Pressed: true, Mods: mods}, nil)
	h.frame(&input.Mouse{X: bx, Y: by, Down: true, Mods: mods}, nil)
	h.frame(&input.Mouse{X: bx, Y: by, Released: true, Mods: mods}, nil)
	got := ed.Selections()
	want := [][2]int{{1, 4}, {8, 9}, {11, 14}}
	if len(got) != 3 || got[0] != want[0] || got[1] != want[1] || got[2] != want[2] {
		t.Fatalf("box selected %v, want %v (the short line to its end)", got, want)
	}
	ed.HandleText([]rune("X"))
	if s := string(ed.Bytes()); s != "aXef\naX\naXefgh\n" {
		t.Fatalf("typing at the box: %q", s)
	}
	// Pasting three lines gives one to each caret.
	ed.Undo()
	clip := &input.MemClipboard{}
	SetFrameResources(h.c.Text(), nil, clip)
	clip.Set("1\n2\n3")
	ed.Paste()
	if s := string(ed.Bytes()); s != "a1ef\na2\na3efgh\n" {
		t.Fatalf("paste per caret: %q", s)
	}
}

func TestAddCursorKeepsColumn(t *testing.T) {
	e := newUndoEditor(t, "abcdef\nab\nabcdef\n")
	e.SetSelection(5, 5)
	e.AddCursorLine(1)
	e.AddCursorLine(1)
	got := e.Selections()
	if len(got) != 3 || got[1][0] != 9 || got[2][0] != 15 {
		t.Fatalf("carets %v, want 5, 9 (short line end), 15", got)
	}
}

func TestEscapeOrder(t *testing.T) {
	e := newUndoEditor(t, "aa aa aa\n")
	e.SetSelection(0, 2)
	e.AddNextOccurrence()
	e.openSearch(false)
	e.search.focused = false // keys go to the text
	e.HandleKeys(key(input.KeyEscape, 0))
	if e.Cursors() != 1 || !e.search.open {
		t.Fatalf("first Escape: %d cursors, find open %v", e.Cursors(), e.search.open)
	}
	e.HandleKeys(key(input.KeyEscape, 0))
	if e.search.open || !e.HasSelection() {
		t.Fatal("second Escape should close find and keep the selection")
	}
	e.HandleKeys(key(input.KeyEscape, 0))
	if e.HasSelection() {
		t.Fatal("third Escape should clear the selection")
	}
}

func TestLineOpUndoRestoresCursors(t *testing.T) {
	e := newUndoEditor(t, "a\nb\nc\nd\n")
	e.SetSelection(0, 0)
	e.AddCursorAt(4)
	e.DuplicateLines(1)
	e.Undo()
	if s := e.Selections(); len(s) != 2 || s[0][0] != 0 || s[1][0] != 4 {
		t.Fatalf("after undo %v, want carets 0 and 4", s)
	}
	e.Redo()
	if s := e.Selections(); len(s) != 2 || s[0][0] != 2 || s[1][0] != 8 {
		t.Fatalf("after redo %v, want carets on the copies 2 and 8", s)
	}
}
