package main

import (
	"time"

	"github.com/mirzakhany/yoga"
	"github.com/mirzakhany/yoga/components"
	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/theme"
)

type appPage int

const (
	pageEditor appPage = iota
	pageComponents
)

// AppShell is the root scene with sidebar navigation.
type AppShell struct {
	root    *layout.Element
	page    appPage
	editor  *EditorPage
	gallery *ComponentGallery
	dialogs *components.DialogHost
	toasts  *components.ToastHost
	content *layout.Element
	nav     *components.Navigation
	lastW   float32
	lastH   float32
}

var _ yoga.Scene = (*AppShell)(nil)

// BuildApp assembles the full demo application.
func BuildApp() *AppShell {
	th := theme.Current()
	app := &AppShell{page: pageEditor}
	app.dialogs = components.NewDialogHost()
	app.toasts = components.NewToastHost()
	app.editor = buildEditorPage(app.dialogs, app.toasts)
	app.editor.relayoutRoot = app.relayout
	app.gallery = buildComponentGallery(app.dialogs, app.toasts)

	app.content = layout.New(layout.Box().FlexGrow(1))
	const sidebarWidth float32 = 88
	app.nav = components.NewNavigation(components.NavVertical, components.NavIconTop)
	app.nav.El.Style = app.nav.El.Style.W(sidebarWidth).FlexShrink(0)
	app.nav.Add(components.NavItem{ID: "editor", Label: "Editor", Icon: "edit"})
	app.nav.Add(components.NavItem{ID: "gallery", Label: "Gallery", Icon: "code"})
	app.nav.Selected = int(app.page)
	app.mountPage()

	row := layout.New(layout.Box().Direction(layout.Row).FlexGrow(1), app.nav.El, app.content)
	children := []*layout.Element{row}
	children = append(children, app.editor.overlayEls()...)
	children = append(children, app.gallery.overlayEls()...)
	children = append(children, app.dialogs.ScrimEl(), app.dialogs.El, app.toasts.El)
	app.root = layout.New(layout.Box().Direction(layout.Column), children...).WithBackgroundPtr(&th.Background)

	app.nav.OnSelect = func(i int, _ string) {
		app.setPage(appPage(i))
	}

	return app
}

func (app *AppShell) setPage(p appPage) {
	app.page = p
	app.nav.Selected = int(p)
	app.mountPage()
	app.relayout()
}

func (app *AppShell) mountPage() {
	switch app.page {
	case pageComponents:
		app.content.Children = []*layout.Element{app.gallery.root}
	default:
		app.content.Children = []*layout.Element{app.editor.root}
	}
	app.content.MarkDirty()
}

func (app *AppShell) relayout() {
	if app.root == nil {
		return
	}
	app.root.MarkDirty()
	if app.lastW > 0 && app.lastH > 0 {
		app.root.Calculate(app.lastW, app.lastH)
	}
}

func (app *AppShell) Root() *layout.Element { return app.root }

func (app *AppShell) ClearColor() render.Color { return theme.Current().Background }

func (app *AppShell) Layout(w, h float32) {
	app.lastW, app.lastH = w, h
	components.SetViewport(w, h)
	app.root.Calculate(w, h)
	app.dialogs.Position(w, h)
	app.toasts.Position(w, h)
}

func (app *AppShell) Update(m *input.Mouse, kb *input.Keyboard) {
	layout.Dispatch(app.root, m)

	if app.dialogs.Open {
		if kb != nil {
			app.dialogs.HandleKeys(kb.Keys)
			app.dialogs.HandleText(kb.Chars)
		}
		app.dialogs.Update(m)
		return
	}

	switch app.page {
	case pageComponents:
		app.gallery.update(m, kb)
	default:
		app.editor.update(m, kb)
	}
}

func (app *AppShell) AnimationWait() (time.Duration, bool) {
	var soonest time.Duration
	var active bool
	if app.page == pageEditor {
		if d, ok := app.editor.animationWait(); ok {
			soonest, active = d, true
		}
	}
	if app.page == pageComponents {
		if d, ok := app.gallery.animationWait(); ok {
			if !active || d < soonest {
				soonest, active = d, true
			}
		}
	}
	if d, ok := app.toasts.AnimationWait(); ok {
		if !active || d < soonest {
			soonest, active = d, true
		}
	}
	return soonest, active
}

func (app *AppShell) Close() {
	app.editor.close()
}
