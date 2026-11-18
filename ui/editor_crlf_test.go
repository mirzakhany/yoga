package ui

import (
	"strings"
	"testing"

	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/text"
)

func TestPieceTableLinesDropCR(t *testing.T) {
	pt := text.New([]byte("ab\r\ncd\r\n\r\nlast\r"))
	for i, want := range []string{"ab", "cd", "", "last\r"} {
		if got := pt.Line(i); got != want || pt.LineLen(i) != len(want) {
			t.Fatalf("line %d: %q (%d), want %q", i, got, pt.LineLen(i), want)
		}
	}
	if pt.LineStart(1) != 4 {
		t.Fatalf("line 1 starts at %d", pt.LineStart(1))
	}
}

// crlf runs keys on src (| marks the caret) and shows the result the same way.
func crlf(t *testing.T, src string, keys ...input.KeyEvent) string {
	t.Helper()
	at := strings.Index(src, "|")
	e := newTestEditor(t, []byte(strings.Replace(src, "|", "", 1)))
	e.SetAutoIndent(AutoIndentKeep)
	e.SetSelection(at, at)
	e.HandleKeys(keys)
	got := string(e.Bytes())
	return got[:e.Caret()] + "|" + got[e.Caret():]
}

func TestEditorCRLF(t *testing.T) {
	key := func(k input.Key) input.KeyEvent { return input.KeyEvent{Key: k} }
	for _, tc := range []struct {
		name, src string
		keys      []input.KeyEvent
		want      string
	}{
		{"enter writes CRLF", "a\r\n  b|c\r\n", []input.KeyEvent{key(input.KeyEnter)}, "a\r\n  b\r\n  |c\r\n"},
		{"enter in an LF file", "a\n  b|c\n", []input.KeyEvent{key(input.KeyEnter)}, "a\n  b\n  |c\n"},
		{"end stops before CR", "|ab\r\ncd", []input.KeyEvent{key(input.KeyEnd)}, "ab|\r\ncd"},
		{"right crosses the break", "ab|\r\ncd", []input.KeyEvent{key(input.KeyRight)}, "ab\r\n|cd"},
		{"left crosses the break", "ab\r\n|cd", []input.KeyEvent{key(input.KeyLeft)}, "ab|\r\ncd"},
		{"backspace joins", "ab\r\n|cd", []input.KeyEvent{key(input.KeyBackspace)}, "ab|cd"},
		{"delete joins", "ab|\r\ncd", []input.KeyEvent{key(input.KeyDelete)}, "ab|cd"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := crlf(t, tc.src, tc.keys...); got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestEditorCRLFWords(t *testing.T) {
	b := []byte("ab  \r\ncd")
	if got := wordRightIn(b, 2); got != 4 {
		t.Fatalf("word right to the line end: %d", got)
	}
	if got := wordRightIn(b, 4); got != 6 {
		t.Fatalf("word right over the break: %d", got)
	}
	if got := wordLeftIn(b, 6); got != 4 {
		t.Fatalf("word left over the break: %d", got)
	}
}

func TestEditorCRLFPasteAndLines(t *testing.T) {
	e := newTestEditor(t, []byte("one\r\ntwo"))
	e.SetSelection(3, 3)
	e.replaceSelection(e.toEOL("\nx\ny"), mergeNone)
	if got := string(e.Bytes()); got != "one\r\nx\r\ny\r\ntwo" {
		t.Fatalf("paste: %q", got)
	}

	e = newTestEditor(t, []byte("one\r\ntwo"))
	e.SetSelection(6, 6) // on the last line, which has no break
	e.DuplicateLines(1)
	if got := string(e.Bytes()); got != "one\r\ntwo\r\ntwo" {
		t.Fatalf("duplicate: %q", got)
	}

	e = newTestEditor(t, []byte("one\r\ntwo"))
	e.SetSelection(6, 6)
	e.MoveLines(-1)
	if got := string(e.Bytes()); got != "two\r\none" {
		t.Fatalf("move up: %q", got)
	}

	e = newTestEditor(t, []byte("one\r\ntwo"))
	e.SetSelection(6, 6)
	e.DeleteLines()
	if got := string(e.Bytes()); got != "one" {
		t.Fatalf("delete last line: %q", got)
	}

	e = newTestEditor(t, []byte("one\r\ntwo\r\n"))
	e.Path = "x.go"
	e.SetSelection(0, 7)
	e.ToggleComment()
	if got := string(e.Bytes()); got != "// one\r\n// two\r\n" {
		t.Fatalf("comment: %q", got)
	}
}
