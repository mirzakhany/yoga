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
	files   *ui.FileDialog
	toasts  *ui.ToastHost
}

var _ yoga.App = (*AppShell)(nil)

func BuildApp() *AppShell {
	app := &AppShell{page: pageEditor}
	app.dialogs = ui.NewDialogHost()
	app.files = ui.NewFileDialog()
	app.toasts = ui.NewToastHost()
	app.editor = buildEditorPage(app.files, app.toasts)
	app.gallery = buildComponentGallery(app.dialogs, app.files, app.toasts)
	return app
}

func (app *AppShell) Body(c *ui.Ctx) ui.View {
	var content ui.View
	switch app.page {
	case pageComponents:
		content = app.gallery.Layout(c)
	default:
		content = app.editor.Layout(c)
	}
	return ui.Column(
		ui.Row(
			ui.Nav("shell-nav", ui.NavVertical, ui.NavIconTop,
				ui.NavItem{ID: "editor", Label: "Editor", Icon: "edit"},
				ui.NavItem{ID: "gallery", Label: "Components", Icon: "code"},
			).Selected(int(app.page)).OnSelectItem(func(i int, _ string) {
				app.page = appPage(i)
			}).Width(88),
			content,
		).Align(ui.AlignStretch).Grow(1),
		app.dialogs,
		app.files,
		app.toasts,
	).Grow(1).Background(ui.TokenSurface)
}

func (app *AppShell) ClearColor() render.Color { return theme.Current().Background }

func (app *AppShell) Close() { app.editor.close() }
