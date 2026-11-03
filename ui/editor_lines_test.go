package ui

import (
	"testing"

	"github.com/mirzakhany/yoga/highlight"
	"github.com/mirzakhany/yoga/input"
)

func linesCase(t *testing.T, doc string, sels [][2]int, run func(e *Editor), want string, wantSels [][2]int) {
	t.Helper()
	e := newUndoEditor(t, doc)
	e.SetSelection(sels[0][0], sels[0][1])
	for _, s := range sels[1:] {
		e.extra = append(e.extra, cursor{caret: s[1], anchor: s[0]})
	}
	e.normalizeCursors()
	run(e)
	if got := string(e.Bytes()); got != want {
		t.Fatalf("text %q, want %q", got, want)
	}
	if wantSels != nil {
		got := e.Selections()
		if len(got) != len(wantSels) {
			t.Fatalf("selections %v, want %v", got, wantSels)
		}
		for i := range got {
			if got[i] != wantSels[i] {
				t.Fatalf("selections %v, want %v", got, wantSels)
			}
		}
	}
	e.Undo()
	if got := string(e.Bytes()); got != doc {
		t.Fatalf("one undo gave %q, want the original %q", got, doc)
	}
}

func TestDuplicateLines(t *testing.T) {
	down := func(e *Editor) { pressOn(e, true, input.KeyDown, input.ModAlt|input.ModShift) }
	up := func(e *Editor) { pressOn(e, false, input.KeyUp, input.ModAlt|input.ModShift) }
	linesCase(t, "a\nb\nc\n", [][2]int{{2, 2}}, down, "a\nb\nb\nc\n", [][2]int{{4, 4}})
	linesCase(t, "a\nb\nc\n", [][2]int{{2, 2}}, up, "a\nb\nb\nc\n", [][2]int{{2, 2}})
	// The last line, without a newline.
	linesCase(t, "a\nb", [][2]int{{3, 3}}, down, "a\nb\nb", [][2]int{{5, 5}})
	// A selection over two lines, and a second cursor further down.
	linesCase(t, "a\nb\nc\nd\n", [][2]int{{0, 3}, {6, 6}}, down, "a\nb\na\nb\nc\nd\nd\n", [][2]int{{4, 7}, {12, 12}})
	// A selection ending at column 0 does not take that line.
	linesCase(t, "a\nb\nc\n", [][2]int{{0, 2}}, down, "a\na\nb\nc\n", nil)
}

func TestMoveLines(t *testing.T) {
	down := func(e *Editor) { pressOn(e, true, input.KeyDown, input.ModAlt) }
	up := func(e *Editor) { pressOn(e, false, input.KeyUp, input.ModAlt) }
	linesCase(t, "a\nb\nc\n", [][2]int{{0, 1}}, down, "b\na\nc\n", [][2]int{{2, 3}})
	linesCase(t, "a\nb\nc\n", [][2]int{{4, 4}}, up, "a\nc\nb\n", [][2]int{{2, 2}})
	// Onto and off the last line without a newline.
	linesCase(t, "a\nb", [][2]int{{0, 0}}, down, "b\na", [][2]int{{2, 2}})
	linesCase(t, "a\nb", [][2]int{{2, 2}}, up, "b\na", [][2]int{{0, 0}})
	// Two cursors on neighbouring lines move together.
	linesCase(t, "a\nb\nc\nd\n", [][2]int{{0, 0}, {2, 2}}, down, "c\na\nb\nd\n", [][2]int{{2, 2}, {4, 4}})
	// At the top, up does nothing (and so undoes nothing).
	e := newUndoEditor(t, "a\nb\n")
	MoveUp := func() { pressOn(e, true, input.KeyUp, input.ModAlt) }
	MoveUp()
	if got := string(e.Bytes()); got != "a\nb\n" {
		t.Fatalf("move up at top: %q", got)
	}
}

func TestDeleteLines(t *testing.T) {
	del := func(e *Editor) { pressOn(e, true, input.KeyK, input.ModSuper|input.ModShift) }
	linesCase(t, "one\ntwo\nthree\n", [][2]int{{5, 5}}, del, "one\nthree\n", [][2]int{{5, 5}})
	linesCase(t, "one\ntwo", [][2]int{{6, 6}}, del, "one", [][2]int{{2, 2}})
	linesCase(t, "a\nb\nc\nd\n", [][2]int{{0, 0}, {4, 4}}, del, "b\nd\n", [][2]int{{0, 0}, {2, 2}})
}

func TestToggleComment(t *testing.T) {
	toggle := func(e *Editor) { e.Path = "x.go"; pressOn(e, true, input.KeySlash, input.ModSuper) }
	linesCase(t, "func f() {\n\tx := 1\n\n\ty := 2\n}\n", [][2]int{{12, 25}}, toggle,
		"func f() {\n\t// x := 1\n\n\t// y := 2\n}\n", nil)
	linesCase(t, "\t// x\n\t//y\n", [][2]int{{0, 8}}, toggle, "\tx\n\ty\n", nil)
	// Mixed: some lines commented, so all get commented.
	linesCase(t, "// a\nb\n", [][2]int{{0, 6}}, toggle, "// // a\n// b\n", nil)
	// The caret keeps its place in the text.
	linesCase(t, "abc\n", [][2]int{{1, 1}}, toggle, "// abc\n", [][2]int{{4, 4}})

	py := func(e *Editor) { e.Path = "x.py"; e.ToggleComment() }
	linesCase(t, "x = 1\n", [][2]int{{0, 0}}, py, "# x = 1\n", nil)
	html := func(e *Editor) { e.Path = "x.html"; e.ToggleComment() }
	linesCase(t, "<p>\n", [][2]int{{0, 0}}, html, "<!-- <p> -->\n", nil)
	// Toggling back.
	linesCase(t, "<!-- <p> -->\n", [][2]int{{0, 0}}, html, "<p>\n", nil)

	own := func(e *Editor) { e.SetLineComment(highlight.Comment{Line: ";;"}); e.ToggleComment() }
	linesCase(t, "x\n", [][2]int{{0, 0}}, own, ";; x\n", nil)
}
