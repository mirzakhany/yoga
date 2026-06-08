package main

import (
	"testing"

	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/shape"
)

func TestBuildAppStartup(t *testing.T) {
	text, err := shape.NewEngine(1, false)
	if err != nil {
		t.Fatal(err)
	}
	var clip input.MemClipboard
	app := BuildApp(text, &clip)
	if app.root == nil {
		t.Fatal("root not initialized")
	}
	if app.nav.Selected != int(pageEditor) {
		t.Fatalf("nav selected: got %d want %d", app.nav.Selected, pageEditor)
	}
	app.Layout(800, 600)
}
