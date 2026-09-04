package main

import (
	"github.com/mirzakhany/yoga"
	"github.com/mirzakhany/yoga/icons"
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
}

var _ yoga.App = (*AppShell)(nil)

func BuildApp() *AppShell {
	app := &AppShell{page: pageEditor}
	app.editor = buildEditorPage()
	app.gallery = buildComponentGallery()
	app.gallery.OnWordWrap = func(v bool) { app.editor.setWordWrap(v) }
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
				ui.NavItem{ID: "editor", Label: "Editor", Icon: icons.Pencil},
				ui.NavItem{ID: "gallery", Label: "Components", Icon: icons.Code},
			).Selected(int(app.page)).OnSelectItem(func(i int, _ string) {
				app.page = appPage(i)
			}).Width(88),
			content,
		).Align(ui.AlignStretch).Grow(1),
	).Grow(1).Background(ui.TokenSurface)
}

func (app *AppShell) Close() { app.editor.close() }
