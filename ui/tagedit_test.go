package ui

import (
	"testing"

	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
	"github.com/mirzakhany/yoga/theme"
)

func TestTagEditEnterAfterTyping(t *testing.T) {
	text, err := shape.NewEngine(1, false)
	if err != nil {
		t.Fatal(err)
	}
	sheet := render.NewSpriteSheet(text.Atlas)
	SetFrameResources(text, sheet, nil)

	te := NewTagEdit(300)
	te.Focus()
	te.field.caret = len("ui")
	te.field.setValue("ui")

	te.HandleKeys([]input.KeyEvent{{Key: input.KeyEnter}})
	if tags := te.Tags(); len(tags) != 1 || tags[0] != "ui" {
		t.Fatalf("tags after enter: %v", tags)
	}
	if te.field.Value != "" {
		t.Fatalf("field value: %q", te.field.Value)
	}
	if te.field.caret != 0 {
		t.Fatalf("caret after enter: %d", te.field.caret)
	}

	te.field.setValue("go")
	te.field.caret = 2
	te.HandleKeys([]input.KeyEvent{{Key: input.KeyEnter}})
	te.HandleText([]rune{'\r'})
	if te.field.caret != 0 {
		t.Fatalf("caret after enter+\\r: %d", te.field.caret)
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
	te := NewTagEdit(width)
	for _, tag := range []string{"ui", "yoga", "jjj", "jjjd", "hh", "ddd", "ee", "t", "vv"} {
		te.addTag(tag)
	}

	te.El.Calculate(width, 400)

	wrapped := false
	contentRight := te.content.Frame.X + te.content.Frame.W
	for i := 0; i < len(te.Tags()); i++ {
		ch := te.content.Children[i]
		if ch.Frame.Y > te.content.Frame.Y+1 {
			wrapped = true
		}
		if ch.Frame.X+ch.Frame.W > contentRight+0.5 {
			t.Fatalf("chip %d overflows content: %+v", i, ch.Frame)
		}
	}
	if !wrapped {
		t.Fatal("expected tags to wrap to multiple rows")
	}

	field := te.content.Children[len(te.content.Children)-1]
	if field.Frame.X+field.Frame.W > contentRight+0.5 {
		t.Fatalf("input overflows content: %+v", field.Frame)
	}

	chipH := th.Metrics.ControlHeight - th.Spacing.S
	singleRowH := chipH + 2*th.Spacing.S
	if te.El.Frame.H <= singleRowH {
		t.Fatalf("height should grow with wrapped rows: got %.1f", te.El.Frame.H)
	}
}
