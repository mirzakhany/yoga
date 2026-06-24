package main

import (
	"github.com/mirzakhany/yoga/components"
	"github.com/mirzakhany/yoga/theme"
	"github.com/mirzakhany/yoga/ui"
)

type TopBar struct {
	workspaceDropdown   *components.Select
	environmentDropdown *components.Select
	searchField         *components.TextField
	themeButton         *components.IconButton
	label               *components.Label

	currentTheme *theme.Theme
}

// NewTopBar constructs the retained widgets once so their state (search text,
// dropdown selection) survives every rebuild.
func NewTopBar() *TopBar {
	t := &TopBar{currentTheme: theme.Current()}

	t.workspaceDropdown = components.NewSelect(150, []components.SelectOption{
		{Label: "Workspace 1", Value: "workspace1"},
		{Label: "Workspace 2", Value: "workspace2"},
		{Label: "Workspace 3", Value: "workspace3"},
	})
	t.environmentDropdown = components.NewSelect(150, []components.SelectOption{
		{Label: "Environment 1", Value: "environment1"},
		{Label: "Environment 2", Value: "environment2"},
		{Label: "Environment 3", Value: "environment3"},
	})
	t.themeButton = components.NewIconButton("theme", theme.Current().Metrics.ControlHeight, nil).
		Action(func() { t.ToggleTheme() })
	t.searchField = components.NewTextField(components.TextFieldConfig{Placeholder: "Search..."}).
		WithIconStart("search")
	t.label = components.NewLabel("Chapar", components.LabelBody)

	return t
}

func (t *TopBar) Layout(c *ui.Ctx) *ui.Element {
	sp := theme.Current().Spacing.S

	left := ui.HStack(
		t.label.Layout(c),
		t.workspaceDropdown.Layout(c),
	).Gap(sp)

	mid := ui.HStack(
		t.searchField.Layout(c).Width(300),
	).Gap(sp)

	right := ui.HStack(
		t.themeButton.Layout(c),
		t.environmentDropdown.Layout(c),
	).Gap(sp)

	// Dropdown menus self-register as overlays from their Layout(c); no manual
	// MenuEl mounting.
	return ui.HStack(left, ui.Spacer(), mid, ui.Spacer(), right).
		Gap(sp).
		Padding(sp)
}

func (t *TopBar) ToggleTheme() {
	if t.currentTheme.Name == "yoga-dark" {
		theme.Use("yoga-light")
	} else {
		theme.Use("yoga-dark")
	}
	t.currentTheme = theme.Current()
}
