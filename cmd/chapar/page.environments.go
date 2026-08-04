package main

import (
	"github.com/mirzakhany/yoga/components"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/theme"
	"github.com/mirzakhany/yoga/ui"
)

type EnvironmentsPage struct {
	splitter *components.Splitter

	importButton *components.Button
	newButton    *components.Button
	searchField  *components.TextField
	environments *components.Tree
	label        *components.Label

	root *ui.Element
}

// NewEnvironmentsPage builds the retained element tree once. Every component
// holds state across frames; Layout(c) only re-registers focus/overlays and
// returns this stable tree. Rebuilding the tree (or the page) per frame would
// reset hover/focus/drag state before input is ever applied.
func NewEnvironmentsPage() *EnvironmentsPage {
	th := theme.Current()
	env := &EnvironmentsPage{}

	env.importButton = components.NewButton("Import").Secondary()
	env.newButton = components.NewButton("New").Primary().IconStart("add")
	env.searchField = components.NewTextField(components.TextFieldConfig{Placeholder: "Search...", IconStart: "search"})

	root := &components.TreeNode{Label: "Environments", Data: "environments"}
	env.environments = components.NewTree(root)
	env.environments.Loader = func(n *components.TreeNode) []*components.TreeNode {
		// Only the root has children; items are leaves so they don't expand
		// (and don't re-trigger the loader → no infinite recursion).
		if n != root {
			return nil
		}
		return []*components.TreeNode{
			{Label: "Environment 1", Data: "environment1", Leaf: true},
			{Label: "Environment 2", Data: "environment2", Leaf: true},
			{Label: "Environment 3", Data: "environment3", Leaf: true},
		}
	}
	env.environments.SetRoot(root)
	// Tree fills its panel with the same slightly-lighter color as the left
	// pane so they read as one surface. Pointer tracks live theme switches.
	env.environments.Background = &theme.Current().Panel

	env.label = components.NewLabel("Environment", components.LabelBody)

	actions := ui.HStack(
		ui.Spacer(),
		env.importButton.El,
		env.newButton.El,
	).Gap(th.Spacing.S).
		MarginTop(th.Spacing.S).
		MarginRight(th.Spacing.S)

	left := ui.VStack(
		components.NewLabel("Environments", components.LabelStrong).El.
			MarginTop(th.Spacing.S).
			MarginBottom(th.Spacing.S).
			MarginLeft(th.Spacing.S).
			MarginRight(th.Spacing.S),
		actions,
		env.searchField.El.
			MarginTop(th.Spacing.S).
			MarginBottom(th.Spacing.S).
			MarginLeft(th.Spacing.S).
			MarginRight(th.Spacing.S),
		env.environments.El,
	).Gap(th.Spacing.S).BgPtr(&theme.Current().Panel)

	right := ui.VStack(
		env.label.El,
	).Gap(th.Spacing.S)

	env.splitter = components.NewSplitter(components.Horizontal,
		components.SplitSection{El: left, Size: 240},
		components.SplitSection{El: right, Size: 0},
	)

	env.root = layout.New(layout.Box().Direction(layout.Column).FlexGrow(1),
		env.splitter.El,
	)
	return env
}

// Layout registers the page's interactive components with the frame's focus
// scope each frame, then returns the retained tree (re-solved by the runtime).
func (e *EnvironmentsPage) Layout(c *ui.Ctx) *ui.Element {
	e.importButton.Layout(c)
	e.newButton.Layout(c)
	e.searchField.Layout(c)
	e.environments.Layout(c)
	return e.root
}
