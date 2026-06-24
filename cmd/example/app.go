package main

import (
	"github.com/mirzakhany/yoga"
	"github.com/mirzakhany/yoga/components"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/theme"
	"github.com/mirzakhany/yoga/ui"
)

type appPage int

const (
	pageEditor appPage = iota
	pageComponents
)

// AppShell is the root application: a sidebar nav switching between the editor
// workspace and the component gallery. It implements yoga.App.
type AppShell struct {
	page    appPage
	editor  *EditorPage
	gallery *ComponentGallery
	dialogs *components.DialogHost
	toasts  *components.ToastHost
	nav     *components.Navigation
}

var _ yoga.App = (*AppShell)(nil)

// BuildApp assembles the full demo application (retained state, built once).
func BuildApp() *AppShell {
	app := &AppShell{page: pageEditor}
	app.dialogs = components.NewDialogHost()
	app.toasts = components.NewToastHost()
	app.editor = buildEditorPage(app.dialogs, app.toasts)
	app.gallery = buildComponentGallery(app.dialogs, app.toasts)

	const sidebarWidth float32 = 88
	app.nav = components.NewNavigation(components.NavVertical, components.NavIconTop)
	app.nav.El.Style = app.nav.El.Style.W(sidebarWidth).FlexShrink(0)
	app.nav.Add(components.NavItem{ID: "editor", Label: "Editor", Icon: "edit"})
	app.nav.Add(components.NavItem{ID: "gallery", Label: "Gallery", Icon: "code"})
	app.nav.Selected = int(app.page)
	app.nav.OnSelect = func(i int, _ string) { app.page = appPage(i) }

	return app
}

// Body builds the shell each frame. The active page and overlay hosts
// self-register focus/overlays/animation through the context.
func (app *AppShell) Body(c *ui.Ctx) *ui.Element {
	th := theme.Current()
	app.nav.Selected = int(app.page)

	var content *ui.Element
	switch app.page {
	case pageComponents:
		content = app.gallery.Layout(c)
	default:
		content = app.editor.Layout(c)
	}

	row := ui.HStack(app.nav.Layout(c), content).Align(layout.AlignStretch).Grow(1)

	// Overlay hosts self-register (scrim/body for the dialog, toast stack).
	app.dialogs.Layout(c)
	app.toasts.Layout(c)

	// Modal: while a dialog is open, route keyboard to it and swallow it so the
	// page widgets behind the scrim do not also receive it.
	if app.dialogs.Open {
		if kb := c.Keyboard(); kb != nil {
			app.dialogs.HandleKeys(kb.Keys)
			app.dialogs.HandleText(kb.Chars)
			kb.Keys = nil
			kb.Chars = nil
		}
		app.dialogs.Update(c.Mouse())
	}

	return ui.VStack(row).Grow(1).BgPtr(&th.Background)
}

// ClearColor tracks the live theme background.
func (app *AppShell) ClearColor() render.Color { return theme.Current().Background }

// Close releases editor resources (Closer capability).
func (app *AppShell) Close() { app.editor.close() }
