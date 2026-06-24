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

var _ yoga.Scene = (*ChaparApp)(nil)

type ChaparApp struct {
	ui *layout.Host

	topBar  *TopBar
	sidebar *Sidebar
	footer  *Footer
}

func BuildChaparApp() *ChaparApp {
	app := &ChaparApp{}
	return app
}

func (app *ChaparApp) Body() *layout.Element {
	app.topBar = NewTopBar()
	app.sidebar = NewSidebar()
	app.footer = NewFooter()

	return layout.VStack(0,
		app.topBar.Layout(),
		HLine(),
		layout.New(layout.Box().Direction(layout.Row).Gap(0).FlexGrow(1),
			app.sidebar.Layout(),
			layout.Spacer(),
		),
		HLine(),
		app.footer.Layout(),
	).Grow(1)
}

func (app *ChaparApp) Close() {
}

func (app *ChaparApp) Update(m *input.Mouse, kb *input.Keyboard) {
	root := app.ui.Root()
	components.SetViewport(root.Frame.W, root.Frame.H)
	layout.Dispatch(root, m)
	app.topBar.Update(m, kb)
}

func (app *ChaparApp) Attach(host *layout.Host) {
	app.ui = host
}

func (app *ChaparApp) AnimationWait() (time.Duration, bool) {
	return 30 * time.Millisecond, false
}

func (app *ChaparApp) ClearColor() render.Color { return theme.Current().Background }
