package ui

import (
	"testing"
)

func TestSelectLayoutRegistersMenuOnlyWhenOpen(t *testing.T) {
	c := New(nil, NewFocusScope(), nil)
	n := Select("s", []SelectOption{{Label: "A", Value: "a"}, {Label: "B", Value: "b"}}).Width(120)

	c.BeginFrame(200, 200, nil, nil)
	n.Layout(c)
	if got := len(c.Overlays()); got != 0 {
		t.Fatalf("closed select should register no overlay, got %d", got)
	}

	st := c.Widget("s", func() any { return &selectState{} }).(*selectState)
	if st.menu == nil {
		t.Fatal("menu not allocated")
	}
	st.menu.OpenAt(0, 0)
	c.BeginFrame(200, 200, nil, nil)
	n.Layout(c)
	if got := len(c.Overlays()); got != 1 {
		t.Fatalf("open select should register its menu overlay, got %d", got)
	}
}

func TestDialogLayoutRegistersScrimAndBodyWhenOpen(t *testing.T) {
	d := NewDialogHost()
	d.ShowError("oops", "bad", nil)

	c := New(nil, NewFocusScope(), nil)
	c.BeginFrame(400, 300, nil, nil)
	d.Layout(c)
	if got := len(c.Overlays()); got != 2 {
		t.Fatalf("open dialog should register scrim+body (2 overlays), got %d", got)
	}
}
