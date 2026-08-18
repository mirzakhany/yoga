package main

import (
	"testing"

	"github.com/mirzakhany/yoga"
	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
	"github.com/mirzakhany/yoga/ui"
)

func TestBuildAppStartup(t *testing.T) {
	text, err := shape.NewEngine(1, false)
	if err != nil {
		t.Fatal(err)
	}
	clip := &input.MemClipboard{}
	sheet := render.NewSpriteSheet(text.Atlas)
	yoga.SetResources(text, sheet, clip)

	app := BuildApp()
	if app.page != pageEditor {
		t.Fatalf("page: got %d want %d", app.page, pageEditor)
	}

	c := ui.New(text, ui.NewFocusScope(), nil)
	c.SetIcons(sheet)
	c.SetClipboard(clip)
	mouse := &input.Mouse{}
	kb := &input.Keyboard{}
	root := ui.BuildFrame(c, app.Body, 800, 600, mouse, kb)
	if root == nil {
		t.Fatal("nil root")
	}
	nav := findNav(root)
	if nav == nil {
		t.Fatal("nav strip not found")
	}
	if nav.Frame.W != 88 {
		t.Fatalf("nav width: got %v want 88", nav.Frame.W)
	}
	if nav.Frame.X != 0 {
		t.Fatalf("nav x: got %v want 0", nav.Frame.X)
	}
}

func findNav(e *layout.Element) *layout.Element {
	if e.Frame.W == 88 && e.Frame.H > 100 {
		return e
	}
	for _, ch := range e.Children {
		if found := findNav(ch); found != nil {
			return found
		}
	}
	return nil
}
