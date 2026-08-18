package main

import (
	"github.com/mirzakhany/yoga/theme"
	"github.com/mirzakhany/yoga/ui"
)

type TopBar struct {
	workspace    string
	environment  string
	query        string
	currentTheme *theme.Theme
}

func NewTopBar() *TopBar {
	return &TopBar{
		workspace:    "workspace1",
		environment:  "environment1",
		currentTheme: theme.Current(),
	}
}

func (t *TopBar) Layout(c *ui.Ctx) ui.View {
	sp := theme.Current().Spacing.S
	workspaces := []ui.SelectOption{
		{Label: "Workspace 1", Value: "workspace1"},
		{Label: "Workspace 2", Value: "workspace2"},
		{Label: "Workspace 3", Value: "workspace3"},
	}
	environments := []ui.SelectOption{
		{Label: "Environment 1", Value: "environment1"},
		{Label: "Environment 2", Value: "environment2"},
		{Label: "Environment 3", Value: "environment3"},
	}
	left := ui.Row(
		ui.Strong("Chapar"),
		ui.Select("workspace", workspaces).
			Width(150).
			Selected(optionIndex(t.workspace, workspaces)).
			OnChange(func(v string) { t.workspace = v }),
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
		ui.Select("environment", environments).
			Width(150).
			Selected(optionIndex(t.environment, environments)).
			OnChange(func(v string) { t.environment = v }),
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

func optionIndex(v string, opts []ui.SelectOption) int {
	for i, o := range opts {
		if o.Value == v {
			return i
		}
	}
	return 0
}
