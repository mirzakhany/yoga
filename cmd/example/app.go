package main

import (
	"github.com/mirzakhany/yoga"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/theme"
	"github.com/mirzakhany/yoga/ui"
)

type appPage int

const (
	pageEditor appPage = iota
	pageComponents
)

type AppShell struct {
	page    appPage
	editor  *EditorPage
	gallery *ComponentGallery
	dialogs *ui.DialogHost
	toasts  *ui.ToastHost
	nav     *ui.Navigation
}

var _ yoga.App = (*AppShell)(nil)

func BuildApp() *AppShell {
	app := &AppShell{page: pageEditor}
	app.dialogs = ui.NewDialogHost()
	app.toasts = ui.NewToastHost()
	app.editor = buildEditorPage(app.dialogs, app.toasts)
	app.gallery = buildComponentGallery(app.dialogs, app.toasts)

	app.nav = ui.NewNavigation(ui.NavVertical, ui.NavIconTop)
	app.nav.Add(ui.NavItem{ID: "editor", Label: "Editor", Icon: "edit"})
	app.nav.Add(ui.NavItem{ID: "gallery", Label: "Components", Icon: "code"})
	app.nav.Selected = int(app.page)
	app.nav.OnSelect = func(i int, _ string) { app.page = appPage(i) }
	return app
}

func (app *AppShell) Body(c *ui.Ctx) ui.View {
	app.nav.Selected = int(app.page)
	var content ui.View
	switch app.page {
	case pageComponents:
		content = app.gallery.Layout(c)
	default:
		content = app.editor.Layout(c)
	}
	app.dialogs.Layout(c)
	app.toasts.Layout(c)
	return ui.Column(
		ui.Row(ui.ViewOf(app.nav).Width(88), content).Align(ui.AlignStretch).Grow(1),
	).Grow(1).Background(ui.TokenSurface)
}

func (app *AppShell) ClearColor() render.Color { return theme.Current().Background }

func (app *AppShell) Close() { app.editor.close() }
