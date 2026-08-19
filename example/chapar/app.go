package main

import (
	"github.com/mirzakhany/yoga/theme"
	"github.com/mirzakhany/yoga/ui"
)

// ChaparApp is the demo application.
type ChaparApp struct {
	topBar        *TopBar
	navigationBar *NavigationBar
	footer        *Footer
}

func BuildChaparApp() *ChaparApp {
	envPage := NewEnvironmentsPage()
	pages := []Page{
		NewPage(0, "environments", "Envs", "code", envPage.Layout),
	}
	navigationBar := NewNavigationBar()
	navigationBar.SetPages(pages)
	return &ChaparApp{
		topBar:        NewTopBar(),
		navigationBar: navigationBar,
		footer:        NewFooter(),
	}
}

func (app *ChaparApp) Body(c *ui.Ctx) ui.View {
	currentPage := app.navigationBar.CurrentPage()
	th := theme.Current()
	return ui.Column(
		app.topBar.Layout(c),
		ui.HLine(1, th.Border),
		ui.Row(
			app.navigationBar.Layout(c),
			ui.VLine(1, th.Border),
			ui.ViewOf(currentPage.Layout(c)).Grow(1),
		).Align(ui.AlignStretch).Grow(1),
		ui.HLine(1, th.Border),
		app.footer.Layout(c),
	).Grow(1)
}
