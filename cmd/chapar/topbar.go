package main

import (
	"github.com/mirzakhany/yoga/components"
	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/theme"
)

type TopBar struct {
	workspaceDropdown   *components.Select
	environmentDropdown *components.Select
	searchField         *components.TextField

	themeButton *components.IconButton

	currentTheme *theme.Theme
}

func NewTopBar() *TopBar {
	return &TopBar{
		currentTheme: theme.Current(),
	}
}

func (t *TopBar) Layout() *layout.Element {
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

	t.themeButton = components.NewIconButton("theme", theme.Current().Metrics.ControlHeight, nil).Action(func() { t.ToggleTheme() })

	t.searchField = components.NewTextField(components.TextFieldConfig{Placeholder: "Search..."}).WithIconStart("search")
	t.searchField.El.Style.Width = layout.Px(300)

	leftSide := layout.HStack(theme.Current().Spacing.S,
		components.NewLabel("Chapar", components.LabelBody).El,
		t.workspaceDropdown.El,
	)

	midSide := layout.HStack(theme.Current().Spacing.S,
		t.searchField.El,
	)

	rightSide := layout.HStack(theme.Current().Spacing.S,
		t.themeButton.El,
		t.environmentDropdown.El,
	)

	return layout.New(
		layout.Box().
			Direction(layout.Row).
			Gap(theme.Current().Spacing.S).
			PaddingAll(theme.Current().Spacing.S),
		leftSide,
		layout.Spacer(),
		midSide,
		layout.Spacer(),
		rightSide,
		t.workspaceDropdown.MenuEl(),
		t.environmentDropdown.MenuEl(),
	)
}

func (t *TopBar) Update(m *input.Mouse, kb *input.Keyboard) {
	t.searchField.Update(m)
}

func (t *TopBar) ToggleTheme() {
	if t.currentTheme.Name == "yoga-dark" {
		theme.Use("yoga-light")
	} else {
		theme.Use("yoga-dark")
	}
	t.currentTheme = theme.Current()
}
