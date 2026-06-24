package main

import (
	"github.com/mirzakhany/yoga/components"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/theme"
	"github.com/mirzakhany/yoga/ui"
)

// ChaparApp is the demo application. It implements yoga.App: a single Body(c)
// builds the element tree each frame from retained component state.
type ChaparApp struct {
	topBar  *TopBar
	sidebar *Sidebar
	footer  *Footer
}

// BuildChaparApp constructs the retained component state once. Components
// (search box, dropdowns) survive every rebuild; Body only re-wires elements.
func BuildChaparApp() *ChaparApp {
	return &ChaparApp{
		topBar:  NewTopBar(),
		sidebar: NewSidebar(),
		footer:  NewFooter(),
	}
}

// Body builds the workspace: top bar, sidebar, footer. No manual host, no
// Attach, no Update, no AnimationWait — the runtime drives all of it.
func (app *ChaparApp) Body(c *ui.Ctx) *ui.Element {
	return ui.VStack(
		app.topBar.Layout(c),
		components.HLine(1, theme.Current().Border),
		ui.HStack(app.sidebar.Layout(c), ui.Spacer()).Align(layout.AlignStretch).Grow(1),
		components.HLine(1, theme.Current().Border),
		app.footer.Layout(c),
	).Grow(1)
}

// ClearColor keeps the GPU clear color in sync with the live theme.
func (app *ChaparApp) ClearColor() render.Color { return theme.Current().Background }
