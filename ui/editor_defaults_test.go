package ui

import (
	"testing"

	"github.com/mirzakhany/yoga/highlight"
	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
)

// withEditorDefaults runs the test with EditorDefaults restored afterwards.
func withEditorDefaults(t *testing.T) {
	t.Helper()
	text, err := shape.NewEngine(1, false)
	if err != nil {
		t.Skip(err)
	}
	SetFrameResources(text, render.NewSpriteSheet(text.Atlas), &input.MemClipboard{})
	saved := EditorDefaults
	t.Cleanup(func() { EditorDefaults = saved })
}

func typeEditor(ed *Editor, s string) {
	for _, r := range s {
		ed.HandleText([]rune{r})
	}
}

func TestEditorAutoCloseBracketsOff(t *testing.T) {
	withEditorDefaults(t)
	EditorDefaults.AutoCloseBrackets = false
	ed := NewEditor(nil, highlight.Noop{})
	defer ed.Close()
	typeEditor(ed, "f(")
	if got := string(ed.Bytes()); got != "f(" {
		t.Fatalf("got %q, want %q", got, "f(")
	}

	EditorDefaults.AutoCloseBrackets = true
	ed2 := NewEditor(nil, highlight.Noop{})
	defer ed2.Close()
	typeEditor(ed2, "f(")
	if got := string(ed2.Bytes()); got != "f()" {
		t.Fatalf("auto-close on: got %q, want %q", got, "f()")
	}
}

func TestEditorAutoCloseQuotes(t *testing.T) {
	withEditorDefaults(t)
	for _, tc := range []struct {
		on        bool
		typed     string
		want      string
		wantCaret int
	}{
		{false, `"a`, `"a`, 2},
		{true, `"a`, `"a"`, 2},
		{true, `"a"`, `"a"`, 3},     // steps over the closing quote
		{true, `don't`, `don't`, 5}, // apostrophe after a letter
		{true, `{"k": "v"}`, `{"k": "v"}`, 10},
	} {
		EditorDefaults.AutoCloseQuotes = tc.on
		EditorDefaults.AutoCloseBrackets = true
		ed := NewEditor(nil, highlight.Noop{})
		typeEditor(ed, tc.typed)
		if got := string(ed.Bytes()); got != tc.want || ed.caret != tc.wantCaret {
			t.Errorf("on=%v typed %q: got %q caret %d, want %q caret %d", tc.on, tc.typed, got, ed.caret, tc.want, tc.wantCaret)
		}
		ed.Close()
	}
}

func TestEditorAutoCloseQuotesWrapsSelection(t *testing.T) {
	withEditorDefaults(t)
	EditorDefaults.AutoCloseQuotes = true
	ed := NewEditor([]byte("word"), highlight.Noop{})
	defer ed.Close()
	ed.SelectAll()
	ed.HandleText([]rune{'"'})
	if got := string(ed.Bytes()); got != `"word"` {
		t.Fatalf("got %q", got)
	}
}

func TestEditorTabInsertsSpaces(t *testing.T) {
	withEditorDefaults(t)
	EditorDefaults.IndentSpaces = true
	ed := NewEditor(nil, highlight.Noop{})
	defer ed.Close()
	tabW := ed.tabW
	typeEditor(ed, "a")
	ed.HandleKeys([]input.KeyEvent{{Key: input.KeyTab}})
	want := "a"
	for i := 1; i < tabW; i++ {
		want += " "
	}
	if got := string(ed.Bytes()); got != want {
		t.Fatalf("got %q, want %q", got, want)
	}

	EditorDefaults.IndentSpaces = false
	ed.HandleKeys([]input.KeyEvent{{Key: input.KeyTab}})
	if got := string(ed.Bytes()); got != want+"\t" {
		t.Fatalf("tabs: got %q", got)
	}
}

func TestEditorFollowsLiveDefaults(t *testing.T) {
	withEditorDefaults(t)
	EditorDefaults.LineNumbers = true
	EditorDefaults.SoftWrap = false
	ed := NewEditor([]byte("x"), highlight.Noop{})
	defer ed.Close()
	pinned := NewEditor([]byte("x"), highlight.Noop{}, WithSoftWrap(false), WithoutGutter())
	defer pinned.Close()
	if ed.gutterW == 0 {
		t.Fatal("gutter hidden with LineNumbers on")
	}

	EditorDefaults.LineNumbers = false
	EditorDefaults.SoftWrap = true
	ed.followDefaults()
	pinned.followDefaults()
	if ed.gutterW != 0 || !ed.SoftWrap {
		t.Fatalf("open editor: gutterW=%v SoftWrap=%v, want 0 true", ed.gutterW, ed.SoftWrap)
	}
	if pinned.gutterW != 0 || pinned.SoftWrap {
		t.Fatalf("pinned editor: gutterW=%v SoftWrap=%v, want 0 false", pinned.gutterW, pinned.SoftWrap)
	}

	EditorDefaults.LineNumbers = true
	pinned.followDefaults()
	ed.followDefaults()
	if ed.gutterW == 0 || pinned.gutterW != 0 {
		t.Fatalf("gutterW ed=%v pinned=%v", ed.gutterW, pinned.gutterW)
	}
}
