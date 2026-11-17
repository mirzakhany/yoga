package ui

import (
	"testing"

	"github.com/mirzakhany/yoga/icons"
	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/shape"
)

func textFieldTestEnv(t *testing.T) (*Ctx, *shape.Engine) {
	t.Helper()
	eng, err := shape.NewEngine(1, false)
	if err != nil {
		t.Fatal(err)
	}
	SetFrameResources(eng, nil, nil)
	c := New(eng, NewFocusScope(), nil)
	return c, eng
}

func layoutTextField(t *testing.T, c *Ctx, id, value string, password bool) (*TextInput, *layout.Element) {
	t.Helper()
	c.BeginFrame(400, 80, nil, nil)
	n := TextField(id, value).Width(300)
	if password {
		n = n.Password(true)
	}
	el := n.Layout(c)
	el.Calculate(400, 80)
	tf := c.Widget(id, func() any { return NewTextInput(TextFieldConfig{}) }).(*TextInput)
	return tf, el
}

func clickField(tf *TextInput, el *layout.Element, x float32) {
	// Press: place caret / start drag.
	tf.onMouse(el, &input.Mouse{
		X: x, Y: el.Frame.Y + el.Frame.H/2,
		Pressed: true, Down: true,
	})
	// Release: collapse pending selection when caret == anchor.
	tf.onMouse(el, &input.Mouse{
		X: x, Y: el.Frame.Y + el.Frame.H/2,
		Down: false, Released: true,
	})
	tf.Focus()
}

func TestTextFieldClickThenTypeDoesNotReplaceFirstChar(t *testing.T) {
	c, _ := textFieldTestEnv(t)
	tf, el := layoutTextField(t, c, "tf", "", false)

	clickField(tf, el, el.Frame.X+el.Frame.W/2)
	tf.HandleText([]rune{'a'})
	if tf.Value != "a" {
		t.Fatalf("after first char: got %q want %q", tf.Value, "a")
	}
	if tf.hasSelection() {
		t.Fatalf("first insert must not leave a selection (caret=%d selAnchor=%d)", tf.caret, tf.selAnchor)
	}
	tf.HandleText([]rune{'b'})
	if tf.Value != "ab" {
		t.Fatalf("after second char: got %q want %q (first char was replaced)", tf.Value, "ab")
	}
	if tf.hasSelection() {
		t.Fatal("expected no selection after typing")
	}
}

func TestTextFieldTabFocusThenType(t *testing.T) {
	c, _ := textFieldTestEnv(t)
	tf, _ := layoutTextField(t, c, "tf", "", false)

	tf.Focus() // Tab / DefaultFocus path: no mouse, selAnchor stays -1
	tf.HandleText([]rune{'h', 'i'})
	if tf.Value != "hi" {
		t.Fatalf("got %q want %q", tf.Value, "hi")
	}
	if tf.hasSelection() {
		t.Fatal("expected no selection")
	}
}

func TestTextFieldDragSelectThenTypeReplaces(t *testing.T) {
	c, _ := textFieldTestEnv(t)
	tf, el := layoutTextField(t, c, "tf", "hello", false)
	tf.Value = "hello"
	tf.caret = len(tf.Value)
	tf.selAnchor = -1

	// Select all via drag from left to right of the text area.
	x0 := tf.textLeft() + 1
	x1 := tf.textLeft() + 80
	tf.onMouse(el, &input.Mouse{X: x0, Y: el.Frame.Y + el.Frame.H/2, Pressed: true, Down: true})
	tf.onMouse(el, &input.Mouse{X: x1, Y: el.Frame.Y + el.Frame.H/2, Down: true})
	tf.onMouse(el, &input.Mouse{X: x1, Y: el.Frame.Y + el.Frame.H/2, Down: false, Released: true})
	tf.Focus()

	if !tf.hasSelection() {
		t.Fatalf("drag should create a selection (caret=%d selAnchor=%d value=%q)", tf.caret, tf.selAnchor, tf.Value)
	}
	tf.HandleText([]rune{'x'})
	if tf.Value != "x" {
		t.Fatalf("typing over selection: got %q want %q", tf.Value, "x")
	}
	if tf.hasSelection() {
		t.Fatal("selection should collapse after replace")
	}
}

func TestTextFieldPasswordClickThenType(t *testing.T) {
	c, _ := textFieldTestEnv(t)
	tf, el := layoutTextField(t, c, "pass", "", true)

	clickField(tf, el, el.Frame.X+el.Frame.W/2)
	tf.HandleText([]rune{'s', 'e', 'c'})
	if tf.Value != "sec" {
		t.Fatalf("password click+type: got %q want %q", tf.Value, "sec")
	}
	if tf.hasSelection() {
		t.Fatal("expected no selection")
	}
}

func TestTextFieldClickRepositionThenType(t *testing.T) {
	c, _ := textFieldTestEnv(t)
	tf, el := layoutTextField(t, c, "tf", "ab", false)
	tf.Value = "ab"
	tf.caret = 2
	tf.selAnchor = -1

	// Click near the start of the text (after padding).
	clickField(tf, el, tf.textLeft()+1)
	tf.HandleText([]rune{'Z'})
	if tf.Value != "Zab" && tf.Value != "aZb" && tf.Value != "abZ" {
		// Offset mapping may land at 0 or 1; either insert position is fine
		// as long as we did not replace an existing character via selection.
		t.Fatalf("unexpected value %q after reposition insert", tf.Value)
	}
	if len(tf.Value) != 3 {
		t.Fatalf("insert should add one rune, got %q", tf.Value)
	}
	if tf.hasSelection() {
		t.Fatal("click reposition must not leave a selection after insert")
	}
}

func TestTextFieldKeepsMinWidthBesideSpacer(t *testing.T) {
	c, _ := textFieldTestEnv(t)
	c.BeginFrame(400, 80, nil, nil)
	row := Row(
		TextField("tf", "").Placeholder("Filter rows…").IconStart(icons.Search),
		Spacer(),
		Button("add", Text("Add Row")),
	).Gap(8)
	el := row.Layout(c)
	el.Calculate(400, 80)
	if len(el.Children) < 3 {
		t.Fatalf("row children: got %d want 3", len(el.Children))
	}
	field, spacer, btn := el.Children[0], el.Children[1], el.Children[2]
	if field.Frame.W < 120 {
		t.Fatalf("text field crushed by spacer: width %v", field.Frame.W)
	}
	if spacer.Frame.W <= 0 {
		t.Fatalf("spacer should take leftover space, got %v", spacer.Frame.W)
	}
	if btn.Frame.X <= field.Frame.X {
		t.Fatalf("button should sit to the right of the field")
	}
}

func TestTextFieldStretchesInColumn(t *testing.T) {
	c, _ := textFieldTestEnv(t)
	c.BeginFrame(400, 80, nil, nil)
	col := Column(TextField("tf", "").Placeholder("Name"))
	el := col.Layout(c)
	el.Calculate(400, 80)
	if len(el.Children) == 0 {
		t.Fatal("column has no children")
	}
	field := el.Children[0]
	if field.Frame.W != 400 {
		t.Fatalf("column stretch: got %v want 400", field.Frame.W)
	}
}

func TestTextFieldExplicitWidthWinsOverMin(t *testing.T) {
	c, _ := textFieldTestEnv(t)
	c.BeginFrame(400, 80, nil, nil)
	el := TextField("tf", "").Placeholder("Add tag...").Width(80).Layout(c)
	el.Calculate(400, 80)
	if el.Frame.W != 80 {
		t.Fatalf("explicit width: got %v want 80", el.Frame.W)
	}
}

func TestTextFieldControlledValueClampsSelAnchor(t *testing.T) {
	c, _ := textFieldTestEnv(t)
	c.BeginFrame(400, 80, nil, nil)
	el := TextField("tf", "hello").Width(300).Layout(c)
	el.Calculate(400, 80)
	tf := c.Widget("tf", func() any { return NewTextInput(TextFieldConfig{}) }).(*TextInput)
	tf.selAnchor = 4
	tf.caret = 5

	// Controlled value shrinks; layout sync should clear an out-of-range anchor.
	c.BeginFrame(400, 80, nil, nil)
	el = TextField("tf", "hi").Width(300).Layout(c)
	el.Calculate(400, 80)
	if tf.selAnchor != -1 {
		t.Fatalf("selAnchor after shrink: got %d want -1", tf.selAnchor)
	}
	if tf.caret > len(tf.Value) {
		t.Fatalf("caret not clamped: %d value=%q", tf.caret, tf.Value)
	}
}

// TestTextFieldWordAndLineKeys checks the editor's word and line chords in a
// text field, on both keymaps: ⌥ and ⌘ on macOS, Ctrl elsewhere.
func TestTextFieldWordAndLineKeys(t *testing.T) {
	defer func(m bool) { editorMac = m }(editorMac)
	const val = "foo.bar  baz qux"
	tests := []struct {
		name    string
		mac     bool
		caret   int
		key     input.Key
		mods    input.Mod
		value   string
		wantLo  int
		wantCar int
	}{
		{"mac alt+left", true, 16, input.KeyLeft, input.ModAlt, val, -1, 13},
		{"mac alt+left over punctuation", true, 4, input.KeyLeft, input.ModAlt, val, -1, 3},
		{"mac alt+right past spaces", true, 7, input.KeyRight, input.ModAlt, val, -1, 12},
		{"mac alt+shift+left selects", true, 16, input.KeyLeft, input.ModAlt | input.ModShift, val, 16, 13},
		{"mac alt+backspace", true, 12, input.KeyBackspace, input.ModAlt, "foo.bar   qux", -1, 9},
		{"mac alt+delete", true, 0, input.KeyDelete, input.ModAlt, ".bar  baz qux", -1, 0},
		{"mac cmd+left", true, 9, input.KeyLeft, input.ModSuper, val, -1, 0},
		{"mac cmd+shift+right", true, 9, input.KeyRight, input.ModSuper | input.ModShift, val, 9, 16},
		{"mac cmd+backspace", true, 9, input.KeyBackspace, input.ModSuper, "baz qux", -1, 0},
		{"mac ctrl+left is the line", true, 9, input.KeyLeft, input.ModCtrl, val, -1, 0},
		{"ctrl+left", false, 16, input.KeyLeft, input.ModCtrl, val, -1, 13},
		{"ctrl+right", false, 0, input.KeyRight, input.ModCtrl, val, -1, 3},
		{"ctrl+backspace", false, 16, input.KeyBackspace, input.ModCtrl, "foo.bar  baz ", -1, 13},
		{"ctrl+delete", false, 9, input.KeyDelete, input.ModCtrl, "foo.bar   qux", -1, 9},
		{"alt+left elsewhere is a char", false, 16, input.KeyLeft, input.ModAlt, val, -1, 15},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			editorMac = tt.mac
			c, _ := textFieldTestEnv(t)
			tf, _ := layoutTextField(t, c, "tf", val, false)
			tf.Value, tf.caret, tf.selAnchor = val, tt.caret, -1
			tf.Focus()
			tf.HandleKeys([]input.KeyEvent{{Key: tt.key, Mods: tt.mods}})
			if tf.Value != tt.value || tf.caret != tt.wantCar || tf.selAnchor != tt.wantLo {
				t.Fatalf("got %q caret %d anchor %d; want %q caret %d anchor %d",
					tf.Value, tf.caret, tf.selAnchor, tt.value, tt.wantCar, tt.wantLo)
			}
		})
	}
}

func TestTextFieldWordKeysEdgeCases(t *testing.T) {
	defer func(m bool) { editorMac = m }(editorMac)
	editorMac = true
	c, _ := textFieldTestEnv(t)
	tf, _ := layoutTextField(t, c, "tf", "", false)
	tf.Focus()
	press := func(key input.Key, mods input.Mod) {
		tf.HandleKeys([]input.KeyEvent{{Key: key, Mods: mods}})
	}

	// Multibyte runes are one word; a selection is deleted whole.
	tf.Value, tf.caret, tf.selAnchor = "héllo wörld", len("héllo wörld"), -1
	press(input.KeyBackspace, input.ModAlt)
	if tf.Value != "héllo " {
		t.Fatalf("value %q", tf.Value)
	}
	tf.selAnchor, tf.caret = 0, len("hé")
	press(input.KeyBackspace, input.ModAlt)
	if tf.Value != "llo " || tf.caret != 0 {
		t.Fatalf("selection delete: %q caret %d", tf.Value, tf.caret)
	}

	// Each word delete is its own undo step.
	tf.Value, tf.caret, tf.selAnchor = "one two", 7, -1
	tf.undo = nil
	press(input.KeyBackspace, input.ModAlt)
	press(input.KeyBackspace, input.ModAlt)
	if tf.Value != "" {
		t.Fatalf("value %q", tf.Value)
	}
	tf.Undo()
	if tf.Value != "one " {
		t.Fatalf("after undo %q", tf.Value)
	}

	// A password is one word.
	pw, _ := layoutTextField(t, c, "pw", "", true)
	pw.Focus()
	pw.Value, pw.caret, pw.selAnchor = "hunter two", 10, -1
	pw.HandleKeys([]input.KeyEvent{{Key: input.KeyLeft, Mods: input.ModAlt}})
	if pw.caret != 0 {
		t.Fatalf("password word-left caret %d", pw.caret)
	}
}
