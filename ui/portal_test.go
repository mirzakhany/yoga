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

func TestSelectOnChangeUpdatesCaller(t *testing.T) {
	c := New(nil, NewFocusScope(), nil)
	got := "none"
	opts := []SelectOption{
		{Label: "None", Value: "none"},
		{Label: "Text", Value: "text"},
		{Label: "JSON", Value: "json"},
	}
	c.BeginFrame(200, 200, nil, nil)
	n := Select("body-type", opts).Selected(optionIndexForTest(got, opts)).OnChange(func(v string) { got = v })
	n.Layout(c)
	st := c.Widget("body-type", func() any { return &selectState{} }).(*selectState)
	if st.menu == nil || len(st.menu.items) < 3 {
		t.Fatal("select menu items missing")
	}
	st.menu.items[2].OnSelect()
	if got != "json" {
		t.Fatalf("OnChange: got %q want json", got)
	}
}

func optionIndexForTest(v string, opts []SelectOption) int {
	for i, o := range opts {
		if o.Value == v {
			return i
		}
	}
	return 0
}
