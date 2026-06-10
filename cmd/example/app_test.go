package main

import (
	"testing"

	"github.com/mirzakhany/yoga"
	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
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
	if app.root == nil {
		t.Fatal("root not initialized")
	}
	if app.nav.Selected != int(pageEditor) {
		t.Fatalf("nav selected: got %d want %d", app.nav.Selected, pageEditor)
	}
	app.Layout(800, 600)
	navW := app.nav.El.Frame.W
	if navW != 88 {
		t.Fatalf("nav width: got %v want 88", navW)
	}
	app.editor.relayout()
	if app.nav.El.Frame.W != navW {
		t.Fatalf("nav width after editor relayout: got %v want %v", app.nav.El.Frame.W, navW)
	}
	if app.nav.El.Frame.X != 0 {
		t.Fatalf("nav x after editor relayout: got %v", app.nav.El.Frame.X)
	}
	if app.editor.root.Frame.X <= 0 {
		t.Fatalf("editor should sit right of nav, got x=%v", app.editor.root.Frame.X)
	}
}
