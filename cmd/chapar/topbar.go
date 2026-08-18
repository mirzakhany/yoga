package main

import (
	"github.com/mirzakhany/yoga/theme"
	"github.com/mirzakhany/yoga/ui"
)

type TopBar struct {
	workspaceDropdown   *ui.Select
	environmentDropdown *ui.Select
	query               string
	currentTheme        *theme.Theme
}

func NewTopBar() *TopBar {
	t := &TopBar{currentTheme: theme.Current()}
	t.workspaceDropdown = ui.NewSelect(150, []ui.SelectOption{
		{Label: "Workspace 1", Value: "workspace1"},
		{Label: "Workspace 2", Value: "workspace2"},
		{Label: "Workspace 3", Value: "workspace3"},
	})
	t.environmentDropdown = ui.NewSelect(150, []ui.SelectOption{
		{Label: "Environment 1", Value: "environment1"},
		{Label: "Environment 2", Value: "environment2"},
		{Label: "Environment 3", Value: "environment3"},
	})
	return t
}

func (t *TopBar) Layout(c *ui.Ctx) ui.View {
	sp := theme.Current().Spacing.S
	left := ui.Row(
		ui.Strong("Chapar"),
		ui.ViewOf(t.workspaceDropdown),
	).Gap(sp)
	mid := ui.Row(
		ui.TextField("chapar-search", t.query).
			Placeholder("Search...").
			IconStart("search").
			OnChange(func(s string) { t.query = s }).
			Width(300),
	).Gap(sp)
	right := ui.Row(
		ui.IconButton("theme", "theme").OnClick(t.ToggleTheme),
		ui.ViewOf(t.environmentDropdown),
	).Gap(sp)
	return ui.Row(left, ui.Spacer(), mid, ui.Spacer(), right).
		Gap(sp).Padding(sp).MarginRight(10).MarginLeft(10)
}

func (t *TopBar) ToggleTheme() {
	if t.currentTheme.Name == "yoga-dark" {
		theme.Use("yoga-light")
	} else {
		theme.Use("yoga-dark")
	}
	t.currentTheme = theme.Current()
}
