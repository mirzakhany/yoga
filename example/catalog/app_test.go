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
	if len(catalogPages) != 19 {
		t.Fatalf("catalogPages: got %d want 19", len(catalogPages))
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

func TestCatalogImagesPageKeepsShell(t *testing.T) {
	app, c := setupCatalog(t)
	app.pageID = "images"
	root := ui.BuildFrame(c, app.Body, 1100, 720, &input.Mouse{}, &input.Keyboard{})
	if root == nil {
		t.Fatal("nil root")
	}
	sidebar := findSidebar(root)
	if sidebar == nil {
		t.Fatal("images page hid the sidebar")
	}
	if root.Frame.H > 10_000 || root.Frame.W > 10_000 {
		t.Fatalf("root frame exploded: %+v", root.Frame)
	}
}

func TestCatalogSVGPageKeepsShell(t *testing.T) {
	app, c := setupCatalog(t)
	app.pageID = "svg"
	root := ui.BuildFrame(c, app.Body, 1100, 720, &input.Mouse{}, &input.Keyboard{})
	if root == nil {
		t.Fatal("nil root")
	}
	sidebar := findSidebar(root)
	if sidebar == nil {
		t.Fatal("svg page hid the sidebar")
	}
	if root.Frame.H > 10_000 || root.Frame.W > 10_000 {
		t.Fatalf("root frame exploded: %+v", root.Frame)
	}
}

func TestCatalogNavPageDoesNotShrinkChrome(t *testing.T) {
	app, c := setupCatalog(t)

	app.pageID = "buttons"
	rootButtons := ui.BuildFrame(c, app.Body, 1100, 720, &input.Mouse{}, &input.Keyboard{})
	topButtons := findTopBar(rootButtons)
	if topButtons == nil {
		t.Fatal("buttons: top bar not found")
	}

	app.pageID = "nav"
	rootNav := ui.BuildFrame(c, app.Body, 1100, 720, &input.Mouse{}, &input.Keyboard{})
	topNav := findTopBar(rootNav)
	if topNav == nil {
		t.Fatal("nav: top bar not found")
	}

	// Tall scroll content used to inflate ScrollView's flex basis, overflow the
	// body column, and shrink the top bar (~10px), shifting the whole shell up.
	if topNav.Frame.H < topButtons.Frame.H-0.5 {
		t.Fatalf("nav shrunk top bar: buttons=%v nav=%v", topButtons.Frame.H, topNav.Frame.H)
	}
	if topNav.Frame.Y > topButtons.Frame.Y+0.5 {
		t.Fatalf("nav moved top bar down: buttons Y=%v nav Y=%v", topButtons.Frame.Y, topNav.Frame.Y)
	}
	sideButtons := findSidebar(rootButtons)
	sideNav := findSidebar(rootNav)
	if sideButtons == nil || sideNav == nil {
		t.Fatal("sidebar missing")
	}
	if sideNav.Frame.Y < sideButtons.Frame.Y-0.5 {
		t.Fatalf("nav shifted sidebar up: buttons Y=%v nav Y=%v", sideButtons.Frame.Y, sideNav.Frame.Y)
	}
}

func findTopBar(e *layout.Element) *layout.Element {
	// Top bar is a full-width row under the root column, shorter than the body.
	if e.Frame.Y == 0 && e.Frame.W > 800 && e.Frame.H > 20 && e.Frame.H < 80 && len(e.Children) >= 3 {
		return e
	}
	for _, ch := range e.Children {
		if found := findTopBar(ch); found != nil {
			return found
		}
	}
	return nil
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
