package ui

import (
	"strings"
	"testing"

	"github.com/mirzakhany/yoga/input"
)

func TestEditorAutoIndent(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mode   AutoIndent
		spaces bool
		src    string // | marks the caret
		want   string
	}{
		{"none", AutoIndentNone, false, "\tfoo|", "\tfoo\n|"},
		{"default is none", AutoIndentDefault, false, "\tfoo|", "\tfoo\n|"},
		{"keep", AutoIndentKeep, false, "\tfoo|", "\tfoo\n\t|"},
		{"keep ignores brackets", AutoIndentKeep, false, "  f() {|", "  f() {\n  |"},
		{"bracket adds a level", AutoIndentBrackets, false, "\tif x {|", "\tif x {\n\t\t|"},
		{"bracket with spaces", AutoIndentBrackets, true, "  a: [|", "  a: [\n      |"},
		{"colon", AutoIndentBrackets, true, "key:|", "key:\n    |"},
		{"split pair", AutoIndentBrackets, false, "\tf() {|  }", "\tf() {\n\t\t|\n\t}"},
		{"mismatched closer stays", AutoIndentBrackets, false, "(|]", "(\n\t|]"},
		{"mid line", AutoIndentBrackets, false, "\tab|cd", "\tab\n\t|cd"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			at := strings.Index(tc.src, "|")
			e := newTestEditor(t, []byte(strings.Replace(tc.src, "|", "", 1)))
			e.SetTabWidth(4)
			e.SetIndentSpaces(tc.spaces)
			e.SetAutoIndent(tc.mode)
			e.SetSelection(at, at)
			e.HandleKeys([]input.KeyEvent{{Key: input.KeyEnter}})
			got := string(e.Bytes())
			got = got[:e.Caret()] + "|" + got[e.Caret():]
			if got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestEditorAutoIndentFollowsDefaults(t *testing.T) {
	old := EditorDefaults.AutoIndent
	defer func() { EditorDefaults.AutoIndent = old }()
	EditorDefaults.AutoIndent = AutoIndentKeep
	e := newTestEditor(t, []byte("  x"))
	e.SetSelection(3, 3)
	e.HandleKeys([]input.KeyEvent{{Key: input.KeyEnter}})
	if got := string(e.Bytes()); got != "  x\n  " {
		t.Fatalf("got %q", got)
	}
}

func TestEditorPerEditorTabWidth(t *testing.T) {
	e := newTestEditor(t, []byte("\tx"))
	def := e.TabWidth()
	wide := e.shapedLine(0).Width
	e.SetTabWidth(2)
	if e.TabWidth() != 2 {
		t.Fatalf("TabWidth %d, want 2", e.TabWidth())
	}
	if w := e.shapedLine(0).Width; w >= wide {
		t.Fatalf("a 2-column tab should shape narrower than %d columns: %v >= %v", def, w, wide)
	}
	e.SetIndentSpaces(true)
	e.SetSelection(0, 0)
	e.HandleKeys([]input.KeyEvent{{Key: input.KeyTab}})
	if got := string(e.Bytes()); got != "  \tx" {
		t.Fatalf("Tab inserted %q, want two spaces", got)
	}
	e.SetTabWidth(0)
	if e.TabWidth() != def {
		t.Fatalf("SetTabWidth(0) should follow the engine (%d), got %d", def, e.TabWidth())
	}
}
