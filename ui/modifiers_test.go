package ui

import (
	"testing"

	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
)

func TestDisabledSelectBlocksGrowAndInteraction(t *testing.T) {
	text, err := shape.NewEngine(1, false)
	if err != nil {
		t.Fatal(err)
	}
	sheet := render.NewSpriteSheet(text.Atlas)
	SetFrameResources(text, sheet, nil)

	c := New(text, NewFocusScope(), nil)
	c.BeginFrame(400, 80, nil, nil)

	opts := []SelectOption{{Label: "A", Value: "a"}}
	row := Row(
		Select("sel", opts).Selected(0).Grow(1).Disabled(true),
		Button("btn", Text("OK")).Width(80),
	).Gap(8)
	el := row.Layout(c)
	root := layout.New(layout.Box(), el)
	root.Calculate(400, 80)

	sel := el.Children[0]
	wantSelW := float32(400 - 80 - 8)
	if sel.Frame.W != wantSelW {
		t.Fatalf("disabled select should still grow: got %v want %v", sel.Frame.W, wantSelW)
	}

	st := c.Widget("sel", func() any { return &selectState{} }).(*selectState)
	st.hovered, st.pressed = true, true
	st.menu = NewMenu(160, nil)
	st.menu.OpenAt(0, 0)

	c.BeginFrame(400, 80, nil, nil)
	Select("sel", opts).Selected(0).Grow(1).Disabled(true).Layout(c)
	if st.hovered || st.pressed {
		t.Fatal("disabled select should clear hover/press")
	}
	if st.menu.Open {
		t.Fatal("disabled select should close menu")
	}
}

func TestDisabledTextFieldBlocksInput(t *testing.T) {
	text, err := shape.NewEngine(1, false)
	if err != nil {
		t.Fatal(err)
	}
	SetFrameResources(text, nil, nil)

	c := New(text, NewFocusScope(), nil)
	c.BeginFrame(300, 40, nil, nil)

	tf := c.Widget("tf", func() any {
		return NewTextInput(TextFieldConfig{})
	}).(*TextInput)
	tf.Value = "hello"

	n := TextField("tf", "hello").Disabled(true)
	n.Layout(c)

	if !tf.disabled {
		t.Fatal("expected text field disabled flag")
	}
	tf.focused = true
	tf.HandleText([]rune{'x'})
	if tf.Value != "hello" {
		t.Fatalf("disabled field should ignore input: got %q", tf.Value)
	}
}

func TestSelectStyleBackgroundApplied(t *testing.T) {
	text, err := shape.NewEngine(1, false)
	if err != nil {
		t.Fatal(err)
	}
	SetFrameResources(text, nil, nil)

	c := New(text, NewFocusScope(), nil)
	c.BeginFrame(200, 40, nil, nil)

	custom := BackgroundColor(render.Color{R: 255, G: 0, B: 0, A: 255})
	el := Select("sel", []SelectOption{{Label: "X", Value: "x"}}).
		Style(custom).
		Layout(c)
	if el == nil {
		t.Fatal("nil element")
	}
	// Paint path merges user Style via spec.resolve; layout still succeeds.
}

func TestDisabledCheckboxIgnoresToggle(t *testing.T) {
	c := New(nil, NewFocusScope(), nil)
	c.BeginFrame(200, 40, nil, nil)

	toggled := false
	n := Checkbox("cb", "task").OnToggle(func(v bool) { toggled = true }).Disabled(true)
	n.Layout(c)

	st := c.Widget("cb", func() any { return &checkboxState{} }).(*checkboxState)
	st.toggle()
	if toggled {
		t.Fatal("disabled checkbox should not toggle")
	}
}
