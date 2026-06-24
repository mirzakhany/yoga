package main

import (
	"testing"

	"github.com/mirzakhany/yoga"
	"github.com/mirzakhany/yoga/input"
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
	if app.nav.Selected != int(pageEditor) {
		t.Fatalf("nav selected: got %d want %d", app.nav.Selected, pageEditor)
	}

	c := ui.New(text, ui.NewFocusScope(), nil)
	mouse := &input.Mouse{}
	kb := &input.Keyboard{}
	root := ui.BuildFrame(c, app.Body, 800, 600, mouse, kb)
	if root == nil {
		t.Fatal("nil root")
	}

	navW := app.nav.El.Frame.W
	if navW != 88 {
		t.Fatalf("nav width: got %v want 88", navW)
	}
	if app.nav.El.Frame.X != 0 {
		t.Fatalf("nav x: got %v want 0", app.nav.El.Frame.X)
	}
	if app.editor.root.Frame.X <= 0 {
		t.Fatalf("editor should sit right of nav, got x=%v", app.editor.root.Frame.X)
	}
}
