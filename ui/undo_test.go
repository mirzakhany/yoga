package ui

import (
	"strings"
	"testing"
	"time"

	"github.com/mirzakhany/yoga/highlight"
	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
)

var (
	keyUndo = []input.KeyEvent{{Key: input.KeyZ, Mods: input.ModCtrl}}
	keyRedo = []input.KeyEvent{{Key: input.KeyZ, Mods: input.ModCtrl | input.ModShift}}
	keyY    = []input.KeyEvent{{Key: input.KeyY, Mods: input.ModCtrl}}
	keyBS   = []input.KeyEvent{{Key: input.KeyBackspace}}
	keyDel  = []input.KeyEvent{{Key: input.KeyDelete}}
)

func typeRunes(h interface{ HandleText([]rune) }, s string) {
	for _, r := range s {
		h.HandleText([]rune{r})
	}
}

func focusedTextInput(t *testing.T) (*TextInput, *input.MemClipboard) {
	t.Helper()
	eng, err := shape.NewEngine(1, false)
	if err != nil {
		t.Skip(err)
	}
	clip := &input.MemClipboard{}
	SetFrameResources(eng, nil, clip)
	t.Cleanup(func() { SetFrameResources(nil, nil, nil) })
	tf := NewTextInput(TextFieldConfig{})
	tf.Focus()
	return tf, clip
}

func TestTextInputUndoWalksBackWordByWord(t *testing.T) {
	tf, _ := focusedTextInput(t)
	var changed string
	tf.OnChange = func(s string) { changed = s }

	typeRunes(tf, "hello big world")
	for _, want := range []string{"hello big ", "hello ", ""} {
		tf.HandleKeys(keyUndo)
		if tf.Value != want || changed != want {
			t.Fatalf("undo: value=%q changed=%q, want %q", tf.Value, changed, want)
		}
	}
	if tf.CanUndo() {
		t.Fatal("CanUndo() = true with history exhausted")
	}
	tf.HandleKeys(keyRedo)
	tf.HandleKeys(keyY)
	if tf.Value != "hello big " || tf.caret != len(tf.Value) {
		t.Fatalf("redo twice: value=%q caret=%d", tf.Value, tf.caret)
	}
	tf.HandleText([]rune{'x'})
	if tf.CanRedo() {
		t.Fatal("a new edit kept the redo history")
	}
}

func TestTextInputUndoGroupsDeletesAndSeparatesPaste(t *testing.T) {
	tf, clip := focusedTextInput(t)
	typeRunes(tf, "abcdef")
	tf.HandleKeys(keyBS)
	tf.HandleKeys(keyBS) // "abcd"
	tf.moveTo(0, false)
	tf.HandleKeys(keyDel)
	tf.HandleKeys(keyDel) // "cd"
	clip.Set("XY")
	tf.Paste() // "XYcd"

	for _, want := range []string{"cd", "abcd", "abcdef", ""} {
		tf.Undo()
		if tf.Value != want {
			t.Fatalf("undo: value=%q, want %q", tf.Value, want)
		}
	}
}

func TestTextInputUndoRestoresSelection(t *testing.T) {
	tf, _ := focusedTextInput(t)
	typeRunes(tf, "hello")
	tf.SelectAll()
	typeRunes(tf, "bye") // replaces the selection: one step
	tf.Undo()
	if tf.Value != "hello" || !tf.HasSelection() {
		t.Fatalf("undo: value=%q selected=%v", tf.Value, tf.HasSelection())
	}
	tf.Undo()
	if tf.Value != "" {
		t.Fatalf("second undo: value=%q", tf.Value)
	}
}

func TestTextInputUndoPauseAndCap(t *testing.T) {
	tf, _ := focusedTextInput(t)
	typeRunes(tf, "ab")
	tf.lastEditAt = time.Now().Add(-2 * undoMergeGap)
	typeRunes(tf, "cd")
	tf.Undo()
	if tf.Value != "ab" {
		t.Fatalf("a pause did not start a step: value=%q", tf.Value)
	}

	tf.resetHistory()
	for range textInputUndoSteps + 20 {
		tf.moveTo(len(tf.Value), false) // break the run: one step each
		tf.HandleText([]rune{'x'})
	}
	if len(tf.undo) != textInputUndoSteps {
		t.Fatalf("undo steps = %d, want %d", len(tf.undo), textInputUndoSteps)
	}
}

func TestTextFieldForgetsHistoryWhenAppChangesValue(t *testing.T) {
	c, _ := textFieldTestEnv(t)
	value := ""
	build := func() *TextInput {
		c.BeginFrame(400, 80, nil, nil)
		el := TextField("f", value).OnChange(func(s string) { value = s }).Width(300).Layout(c)
		el.Calculate(400, 80)
		return c.Widget("f", func() any { return nil }).(*TextInput)
	}
	tf := build()
	tf.Focus()
	typeRunes(tf, "abc")
	tf = build() // the value came back from OnChange: history stays
	if !tf.CanUndo() {
		t.Fatal("history dropped for the field's own edit")
	}
	value = "reset"
	tf = build()
	if tf.CanUndo() {
		t.Fatal("history kept after the app replaced the value")
	}
}

func TestTableCellEditStartsFreshHistory(t *testing.T) {
	_, _ = focusedTextInput(t)
	tbl := NewTable([]TableColumn{{ID: "k", Label: "Key", Kind: TableColEditable}}, nil)
	tbl.SetRows([]TableRow{{ID: "a", Cells: map[string]string{"k": "one"}}, {ID: "b", Cells: map[string]string{"k": "two"}}})
	tbl.StartCellEdit("a", "k")
	tbl.HandleText([]rune{'!'})
	tbl.CommitCellEdit()
	tbl.StartCellEdit("b", "k")
	if tbl.editField.CanUndo() {
		t.Fatal("cell edit inherited the previous cell's history")
	}
	tbl.HandleKeys(keyUndo)
	if tbl.editField.Value != "two" {
		t.Fatalf("undo in a fresh cell changed it: %q", tbl.editField.Value)
	}
}

func newUndoEditor(t *testing.T, content string, opts ...EditorOption) *Editor {
	t.Helper()
	text, err := shape.NewEngine(1, false)
	if err != nil {
		t.Skip(err)
	}
	SetFrameResources(text, render.NewSpriteSheet(text.Atlas), &input.MemClipboard{})
	t.Cleanup(func() { SetFrameResources(nil, nil, nil) })
	ed := NewEditor([]byte(content), highlight.Noop{}, append(opts, WithoutGutter())...)
	t.Cleanup(ed.Close)
	return ed
}

func TestEditorUndoWalksBackWordByWord(t *testing.T) {
	ed := newUndoEditor(t, "")
	typeRunes(ed, "hello big world")
	for _, want := range []string{"hello big ", "hello ", ""} {
		ed.HandleKeys(keyUndo)
		if got := string(ed.Bytes()); got != want {
			t.Fatalf("undo: %q, want %q", got, want)
		}
	}
	ed.HandleKeys(keyY)
	if got := string(ed.Bytes()); got != "hello " {
		t.Fatalf("Ctrl+Y: %q", got)
	}
}

func TestEditorUndoGroupsDeletes(t *testing.T) {
	ed := newUndoEditor(t, "abcdef")
	ed.moveTo(4, false)
	ed.HandleKeys(keyBS)
	ed.HandleKeys(keyBS) // "abef"
	ed.HandleKeys(keyDel)
	ed.HandleKeys(keyDel) // "ab"
	if got := string(ed.Bytes()); got != "ab" {
		t.Fatalf("after deletes: %q", got)
	}
	ed.Undo()
	if got := string(ed.Bytes()); got != "abef" {
		t.Fatalf("undo forward deletes: %q", got)
	}
	ed.Undo()
	if got := string(ed.Bytes()); got != "abcdef" || ed.caret != 4 || ed.CanUndo() {
		t.Fatalf("undo backspaces: %q caret=%d canUndo=%v", got, ed.caret, ed.CanUndo())
	}
}

func TestEditorUndoPauseStartsStep(t *testing.T) {
	ed := newUndoEditor(t, "")
	typeRunes(ed, "ab")
	ed.lastEditAt = time.Now().Add(-2 * undoMergeGap)
	typeRunes(ed, "cd")
	ed.Undo()
	if got := string(ed.Bytes()); got != "ab" {
		t.Fatalf("undo after pause: %q", got)
	}
}

func TestEditorUndoLimits(t *testing.T) {
	ed := newUndoEditor(t, "", WithUndoLimit(3, 0))
	for range 5 {
		ed.replaceSelection("x\n", mergeNone)
	}
	if len(ed.undo) != 3 {
		t.Fatalf("steps = %d, want 3", len(ed.undo))
	}

	ed = newUndoEditor(t, "", WithUndoLimit(0, 10))
	ed.replaceSelection("12345\n", mergeNone)
	ed.replaceSelection("12345\n", mergeNone) // 12 bytes: the first drops
	if len(ed.undo) != 1 || ed.undoBytes != 6 {
		t.Fatalf("steps=%d bytes=%d, want 1 and 6", len(ed.undo), ed.undoBytes)
	}
	big := strings.Repeat("y", 50)
	ed.replaceSelection(big, mergeNone) // over the limit alone: still kept
	if len(ed.undo) != 1 || ed.undoBytes != 50 {
		t.Fatalf("steps=%d bytes=%d, want the newest step only", len(ed.undo), ed.undoBytes)
	}
	ed.Undo()
	if ed.undoBytes != 0 || !ed.CanRedo() {
		t.Fatalf("after undo: bytes=%d canRedo=%v", ed.undoBytes, ed.CanRedo())
	}
	ed.Redo()
	if ed.undoBytes != 50 {
		t.Fatalf("after redo: bytes=%d", ed.undoBytes)
	}
}
