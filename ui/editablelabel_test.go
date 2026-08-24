package ui

import (
	"testing"

	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/shape"
)

func editableLabelTestEnv(t *testing.T) (*Ctx, *shape.Engine) {
	t.Helper()
	eng, err := shape.NewEngine(1, false)
	if err != nil {
		t.Fatal(err)
	}
	SetFrameResources(eng, nil, nil)
	c := New(eng, NewFocusScope(), nil)
	return c, eng
}

func layoutEditableLabel(t *testing.T, c *Ctx, id, value string, onSave func(string)) (*editableLabelState, *layout.Element) {
	t.Helper()
	c.BeginFrame(400, 80, nil, nil)
	n := EditableLabel(id, value).Width(200)
	if onSave != nil {
		n = n.OnSave(onSave)
	}
	el := n.Layout(c)
	el.Calculate(400, 80)
	st := c.Widget(id, func() any {
		field := NewTextInput(TextFieldConfig{Height: editableLabelHeight(c.Theme())})
		return &editableLabelState{field: field}
	}).(*editableLabelState)
	return st, el
}

func clickEditableLabel(st *editableLabelState, el *layout.Element) {
	st.el = el
	insideY := el.Frame.Y + el.Frame.H/2
	insideX := el.Frame.X + el.Frame.W/2
	el.OnMouse(el, &input.Mouse{X: insideX, Y: insideY, Pressed: true, Down: true})
	el.OnMouse(el, &input.Mouse{X: insideX, Y: insideY, Down: false, Released: true})
}

func TestEditableLabelClickEntersEdit(t *testing.T) {
	c, _ := editableLabelTestEnv(t)
	st, el := layoutEditableLabel(t, c, "el", "Title", nil)

	if st.editing {
		t.Fatal("should start in label mode")
	}
	clickEditableLabel(st, el)
	if !st.editing {
		t.Fatal("click should enter edit mode")
	}
	if st.field.Value != "Title" {
		t.Fatalf("draft value: got %q want %q", st.field.Value, "Title")
	}
}

func TestEditableLabelEnterSaves(t *testing.T) {
	c, _ := editableLabelTestEnv(t)
	var saved string
	st, el := layoutEditableLabel(t, c, "el", "Title", func(s string) { saved = s })

	clickEditableLabel(st, el)
	st.field.HandleText([]rune{'!'})
	st.HandleKeys([]input.KeyEvent{{Key: input.KeyEnter}})

	if st.editing {
		t.Fatal("Enter should exit edit mode")
	}
	if saved != "!" {
		t.Fatalf("OnSave: got %q want %q", saved, "!")
	}
}

func TestEditableLabelEscapeCancels(t *testing.T) {
	c, _ := editableLabelTestEnv(t)
	var saved string
	st, el := layoutEditableLabel(t, c, "el", "Title", func(s string) { saved = s })

	clickEditableLabel(st, el)
	st.field.HandleText([]rune{'x'})
	st.HandleKeys([]input.KeyEvent{{Key: input.KeyEscape}})

	if st.editing {
		t.Fatal("Escape should exit edit mode")
	}
	if saved != "" {
		t.Fatalf("Escape should not call OnSave, got %q", saved)
	}
}

func TestEditableLabelBlurCancels(t *testing.T) {
	c, _ := editableLabelTestEnv(t)
	var saved string
	st, el := layoutEditableLabel(t, c, "el", "Title", func(s string) { saved = s })

	clickEditableLabel(st, el)
	st.field.HandleText([]rune{'x'})
	st.Blur()

	if st.editing {
		t.Fatal("Blur should exit edit mode")
	}
	if saved != "" {
		t.Fatalf("Blur should not call OnSave, got %q", saved)
	}
	_ = el
}

func TestEditableLabelHeightStableInEdit(t *testing.T) {
	c, _ := editableLabelTestEnv(t)
	st, el := layoutEditableLabel(t, c, "el", "Title", nil)
	labelH := el.Frame.H

	clickEditableLabel(st, el)

	c.BeginFrame(400, 80, nil, nil)
	editEl := EditableLabel("el", "Title").Width(200).Layout(c)
	editEl.Calculate(400, 80)
	if editEl.Frame.H != labelH {
		t.Fatalf("height changed in edit mode: label=%v edit=%v", labelH, editEl.Frame.H)
	}
}

func TestEditableLabelHoverSetsTextCursor(t *testing.T) {
	c, _ := editableLabelTestEnv(t)
	_, el := layoutEditableLabel(t, c, "el", "Title", nil)
	m := &input.Mouse{
		X: el.Frame.X + el.Frame.W/2,
		Y: el.Frame.Y + el.Frame.H/2,
	}
	el.OnMouse(el, m)
	if m.Cursor != CursorText {
		t.Fatalf("hover cursor: got %v want CursorText", m.Cursor)
	}
}
