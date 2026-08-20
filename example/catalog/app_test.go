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

func setupCatalog(t *testing.T) (*CatalogApp, *ui.Ctx) {
	t.Helper()
	text, err := shape.NewEngine(1, false)
	if err != nil {
		t.Fatal(err)
	}
	clip := &input.MemClipboard{}
	sheet := render.NewSpriteSheet(text.Atlas)
	yoga.SetResources(text, sheet, clip)

	app := BuildCatalog()
	c := ui.New(text, ui.NewFocusScope(), nil)
	c.SetIcons(sheet)
	c.SetClipboard(clip)
	return app, c
}

func TestBuildCatalogStartup(t *testing.T) {
	app, c := setupCatalog(t)
	if app.pageID != "buttons" {
		t.Fatalf("pageID: got %q want %q", app.pageID, "buttons")
	}
	if len(catalogPages) != 13 {
		t.Fatalf("catalogPages: got %d want 13", len(catalogPages))
	}

	root := ui.BuildFrame(c, app.Body, 1100, 720, &input.Mouse{}, &input.Keyboard{})
	if root == nil {
		t.Fatal("nil root")
	}
	sidebar := findSidebar(root)
	if sidebar == nil {
		t.Fatal("sidebar not found")
	}
	if sidebar.Frame.W != sidebarWidth {
		t.Fatalf("sidebar width: got %v want %v", sidebar.Frame.W, sidebarWidth)
	}
}

func TestCatalogPageSwitch(t *testing.T) {
	app, c := setupCatalog(t)
	app.pageID = "typography"
	root := ui.BuildFrame(c, app.Body, 1100, 720, &input.Mouse{}, &input.Keyboard{})
	if root == nil {
		t.Fatal("nil root")
	}
	app.pageID = "feedback"
	root2 := ui.BuildFrame(c, app.Body, 1100, 720, &input.Mouse{}, &input.Keyboard{})
	if root2 == nil {
		t.Fatal("nil root after page switch")
	}
}

func TestCatalogIconsPageKeepsShell(t *testing.T) {
	app, c := setupCatalog(t)
	app.pageID = "icons"
	root := ui.BuildFrame(c, app.Body, 1100, 720, &input.Mouse{}, &input.Keyboard{})
	if root == nil {
		t.Fatal("nil root")
	}
	sidebar := findSidebar(root)
	if sidebar == nil {
		t.Fatal("icons page hid the sidebar")
	}
	if sidebar.Frame.W != sidebarWidth {
		t.Fatalf("sidebar width: got %v want %v", sidebar.Frame.W, sidebarWidth)
	}
	if sidebar.Frame.X != 0 {
		t.Fatalf("sidebar x: got %v want 0", sidebar.Frame.X)
	}
	// Content must stay finite — fr-intrinsic used to report ~1e9 and blow the scroll pane.
	if root.Frame.H > 10_000 || root.Frame.W > 10_000 {
		t.Fatalf("root frame exploded: %+v", root.Frame)
	}
}

func findSidebar(e *layout.Element) *layout.Element {
	if e.Frame.W == sidebarWidth && e.Frame.H > 200 {
		return e
	}
	for _, ch := range e.Children {
		if found := findSidebar(ch); found != nil {
			return found
		}
	}
	return nil
}
