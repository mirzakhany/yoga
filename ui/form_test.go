package ui

import (
	"testing"

	"github.com/mirzakhany/yoga/shape"
)

func TestFormNumberAcceptsTypedValueWithinRange(t *testing.T) {
	text, err := shape.NewEngine(1, false)
	if err != nil {
		t.Fatal(err)
	}
	SetFrameResources(text, nil, nil)
	c := New(text, NewFocusScope(), nil)

	value := 14.0
	frame := func() {
		c.BeginFrame(400, 200, nil, nil)
		Form("f", FormNumber("size", "Size", "", value, 10, 22, 1, func(v float64) { value = v })).Layout(c)
		c.EndFrame()
	}
	frame()
	tf := c.Widget("size", nil).(*TextInput)
	tf.focused = true
	tf.selAnchor, tf.caret = 0, len(tf.Value)

	for _, r := range "16" {
		tf.HandleText([]rune{r})
		frame()
	}
	if value != 16 {
		t.Fatalf("value = %v, want 16", value)
	}
	if tf.Value != "16" {
		t.Fatalf("field text = %q, want %q", tf.Value, "16")
	}

	// Empty and out-of-range drafts survive while editing without committing.
	tf.selAnchor, tf.caret = 0, len(tf.Value)
	tf.HandleText([]rune{'3'})
	frame()
	if tf.Value != "3" || value != 16 {
		t.Fatalf("draft: text %q value %v, want %q and 16", tf.Value, value, "3")
	}

	// Blur discards the invalid draft and shows the committed value again.
	tf.Blur()
	frame()
	if tf.Value != "16" {
		t.Fatalf("after blur text = %q, want %q", tf.Value, "16")
	}
}
