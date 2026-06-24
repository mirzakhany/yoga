package components

import (
	"testing"

	"github.com/mirzakhany/yoga/ui"
)

func TestSelectLayoutRegistersMenuOnlyWhenOpen(t *testing.T) {
	s := NewSelect(120, []SelectOption{{Label: "A", Value: "a"}, {Label: "B", Value: "b"}})

	c := ui.New(nil, ui.NewFocusScope(), nil)

	// Closed: no overlay registered.
	c.BeginFrame(200, 200, nil, nil)
	s.Layout(c)
	if got := len(c.Overlays()); got != 0 {
		t.Fatalf("closed select should register no overlay, got %d", got)
	}

	// Open: menu self-registers as an overlay.
	s.menu.OpenAt(0, 0)
	c.BeginFrame(200, 200, nil, nil)
	s.Layout(c)
	if got := len(c.Overlays()); got != 1 {
		t.Fatalf("open select should register its menu overlay, got %d", got)
	}
}

func TestDialogLayoutRegistersScrimAndBodyWhenOpen(t *testing.T) {
	d := NewDialogHost()
	d.ShowError("oops", "bad", nil) // opens the dialog

	c := ui.New(nil, ui.NewFocusScope(), nil)
	c.BeginFrame(400, 300, nil, nil)
	d.Layout(c)
	if got := len(c.Overlays()); got != 2 {
		t.Fatalf("open dialog should register scrim+body (2 overlays), got %d", got)
	}
}
