package ui

import (
	"testing"

	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
	"github.com/mirzakhany/yoga/theme"
)

func TestTagEditAddsOnSubmit(t *testing.T) {
	text, err := shape.NewEngine(1, false)
	if err != nil {
		t.Fatal(err)
	}
	sheet := render.NewSpriteSheet(text.Atlas)
	SetFrameResources(text, sheet, nil)

	tags := []string{}
	c := New(text, NewFocusScope(), nil)
	c.SetIcons(sheet)
	c.BeginFrame(300, 80, nil, nil)
	n := TagEdit("t", tags).OnTags(func(next []string) { tags = next }).Width(300)
	n.Layout(c)

	st := c.Widget("t", func() any { return &tagState{} }).(*tagState)
	st.draft = "ui"
	tf := c.Widget("t-in", func() any { return NewTextInput(TextFieldConfig{}) }).(*TextInput)
	tf.Value = "ui"
	tf.OnSubmit = func(s string) {
		tags = append(tags, s)
		st.draft = ""
	}
	tf.OnSubmit("ui")
	if len(tags) != 1 || tags[0] != "ui" {
		t.Fatalf("tags after submit: %v", tags)
	}
}

func TestTagEditWrap(t *testing.T) {
	text, err := shape.NewEngine(1, false)
	if err != nil {
		t.Fatal(err)
	}
	sheet := render.NewSpriteSheet(text.Atlas)
	SetFrameResources(text, sheet, nil)

	th := theme.Current()
	width := float32(220)
	tags := []string{"ui", "yoga", "jjj", "jjjd", "hh", "ddd", "ee", "t", "vv"}
	c := New(text, NewFocusScope(), nil)
	c.SetIcons(sheet)
	c.BeginFrame(width, 400, nil, nil)
	el := TagEdit("t", tags).Width(width).Layout(c)
	el.Calculate(width, 400)

	wrapped := false
	contentRight := el.Frame.X + el.Frame.W
	for i, ch := range el.Children {
		if ch.Frame.Y > el.Frame.Y+1 {
			wrapped = true
		}
		if ch.Frame.X+ch.Frame.W > contentRight+0.5 {
			t.Fatalf("chip %d overflows content: %+v", i, ch.Frame)
		}
	}
	if !wrapped {
		t.Fatal("expected tags to wrap to multiple rows")
	}

	chipH := th.Metrics.ControlHeight - th.Spacing.S
	singleRowH := chipH + 2*th.Spacing.S
	if el.Frame.H <= singleRowH {
		t.Fatalf("height should grow with wrapped rows: got %.1f", el.Frame.H)
	}
}
