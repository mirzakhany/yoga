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

func TestGalleryDoesNotOverlap(t *testing.T) {
	text, err := shape.NewEngine(1, false)
	if err != nil {
		t.Fatal(err)
	}
	clip := &input.MemClipboard{}
	sheet := render.NewSpriteSheet(text.Atlas)
	yoga.SetResources(text, sheet, clip)

	app := BuildApp()
	app.page = pageComponents
	c := ui.New(text, ui.NewFocusScope(), nil)
	c.SetIcons(sheet)
	c.SetClipboard(clip)
	root := ui.BuildFrame(c, app.Body, 800, 500, &input.Mouse{}, &input.Keyboard{})
	col := findTallestColumn(root)
	if col == nil || len(col.Children) < 8 {
		t.Fatalf("gallery column not found: children=%d", childCount(col))
	}
	var prev *layout.Element
	for i, ch := range col.Children {
		if ch.Frame.H <= 0 {
			continue
		}
		if prev != nil && ch.Frame.Y < prev.Frame.Y+prev.Frame.H-0.5 {
			t.Fatalf("gallery children overlap at %d: prev=%v child=%v", i, prev.Frame, ch.Frame)
		}
		prev = ch
	}
}

func findTallestColumn(e *layout.Element) *layout.Element {
	best := e
	var walk func(*layout.Element)
	walk = func(n *layout.Element) {
		if n == nil {
			return
		}
		if len(n.Children) > len(best.Children) {
			best = n
		}
		for _, ch := range n.Children {
			walk(ch)
		}
	}
	walk(e)
	return best
}

func childCount(e *layout.Element) int {
	if e == nil {
		return 0
	}
	return len(e.Children)
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
