package ui

import (
	"strings"
	"testing"

	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/layout"
)

// layoutTextArea lays out a TextArea of the given width twice, so its height
// settles on the wrap of that width.
func layoutTextArea(t *testing.T, c *Ctx, n *Node) (*TextInput, *layout.Element) {
	t.Helper()
	var el *layout.Element
	for range 2 {
		c.BeginFrame(400, 400, nil, nil)
		el = Column(n).Width(300).Layout(c)
		el.Calculate(400, 400)
	}
	tf := c.Widget(n.id, func() any { return NewTextInput(TextFieldConfig{}) }).(*TextInput)
	return tf, tf.host
}

func TestTextAreaShiftEnterInsertsNewlineEnterSubmits(t *testing.T) {
	c, _ := textFieldTestEnv(t)
	var submitted string
	tf, _ := layoutTextArea(t, c, TextArea("ta", "").OnSubmit(func(v string) { submitted = v }))
	tf.OnChange = func(v string) {}
	tf.Focus()
	tf.HandleText([]rune("one"))
	tf.HandleKeys([]input.KeyEvent{{Key: input.KeyEnter, Mods: input.ModShift}})
	tf.HandleText([]rune("two"))
	if tf.Value != "one\ntwo" {
		t.Fatalf("value: got %q want %q", tf.Value, "one\ntwo")
	}
	tf.HandleKeys([]input.KeyEvent{{Key: input.KeyEnter}})
	if submitted != "one\ntwo" {
		t.Fatalf("submitted: got %q", submitted)
	}
}

func TestTextAreaEnterInsertsNewlineWithoutSubmit(t *testing.T) {
	c, _ := textFieldTestEnv(t)
	tf, _ := layoutTextArea(t, c, TextArea("ta", ""))
	tf.Focus()
	tf.HandleText([]rune("a"))
	tf.HandleKeys([]input.KeyEvent{{Key: input.KeyEnter}})
	tf.HandleText([]rune("b"))
	if tf.Value != "a\nb" {
		t.Fatalf("value: got %q", tf.Value)
	}
}

func TestTextAreaHeightFollowsRows(t *testing.T) {
	c, _ := textFieldTestEnv(t)
	_, el := layoutTextArea(t, c, TextArea("ta", "").Rows(2, 4))
	two := el.Frame.H
	_, el = layoutTextArea(t, c, TextArea("ta", "a\nb\nc").Rows(2, 4))
	three := el.Frame.H
	_, el = layoutTextArea(t, c, TextArea("ta", strings.Repeat("x\n", 20)).Rows(2, 4))
	capped := el.Frame.H
	if !(two < three && three < capped) {
		t.Fatalf("heights should grow 2 < 3 < 4 rows: %v %v %v", two, three, capped)
	}
	_, el = layoutTextArea(t, c, TextArea("ta", strings.Repeat("x\n", 40)).Rows(2, 4))
	if el.Frame.H != capped {
		t.Fatalf("height past max rows: got %v want %v", el.Frame.H, capped)
	}
}

func TestTextAreaWrapsLongLines(t *testing.T) {
	c, _ := textFieldTestEnv(t)
	tf, _ := layoutTextArea(t, c, TextArea("ta", strings.Repeat("word ", 60)).Width(200))
	if n := len(tf.lines()); n < 3 {
		t.Fatalf("a long line at 200px should wrap, got %d lines", n)
	}
}

func TestTextAreaUpDownKeepColumn(t *testing.T) {
	c, _ := textFieldTestEnv(t)
	tf, _ := layoutTextArea(t, c, TextArea("ta", "abcdef\nxy\nabcdef"))
	tf.Focus()
	tf.moveTo(4, false) // abcd|ef
	tf.HandleKeys([]input.KeyEvent{{Key: input.KeyDown}})
	if tf.caret != len("abcdef\nxy") {
		t.Fatalf("down onto a short line ends it: caret %d", tf.caret)
	}
	tf.HandleKeys([]input.KeyEvent{{Key: input.KeyDown}})
	if want := len("abcdef\nxy\nabcd"); tf.caret != want {
		t.Fatalf("down again returns to the column: caret %d want %d", tf.caret, want)
	}
	tf.HandleKeys([]input.KeyEvent{{Key: input.KeyHome}})
	if want := len("abcdef\nxy\n"); tf.caret != want {
		t.Fatalf("home goes to the line start: caret %d want %d", tf.caret, want)
	}
	tf.HandleKeys([]input.KeyEvent{{Key: input.KeyUp, Mods: input.ModShift}})
	if !tf.hasSelection() {
		t.Fatal("shift+up should select")
	}
}

func TestTextAreaClickPlacesCaretOnLine(t *testing.T) {
	c, _ := textFieldTestEnv(t)
	tf, el := layoutTextArea(t, c, TextArea("ta", "first\nsecond"))
	y := tf.textTop() + tf.lineHeight()*1.5
	tf.onMouse(el, &input.Mouse{X: el.Frame.X + el.Frame.W - 2, Y: y, Pressed: true, Down: true})
	tf.onMouse(el, &input.Mouse{X: el.Frame.X + el.Frame.W - 2, Y: y, Released: true})
	if tf.caret != len(tf.Value) {
		t.Fatalf("click past the end of line 2: caret %d want %d", tf.caret, len(tf.Value))
	}
}

func TestTextAreaPasteKeepsLines(t *testing.T) {
	c, _ := textFieldTestEnv(t)
	tf, _ := layoutTextArea(t, c, TextArea("ta", ""))
	tf.Focus()
	tf.edit(mergeNone, "", func() { tf.insertAtCaret("a\r\nb\rc") })
	if tf.Value != "a\nb\nc" {
		t.Fatalf("value: got %q", tf.Value)
	}
}
