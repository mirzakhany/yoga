package main

import (
	"fmt"

	"github.com/mirzakhany/yoga/theme"
	"github.com/mirzakhany/yoga/ui"
)

type EnvironmentsPage struct {
	query        string
	environments *ui.Tree
	nextEnv      int
}

func NewEnvironmentsPage() *EnvironmentsPage {
	env := &EnvironmentsPage{nextEnv: 4}
	root := &ui.TreeNode{
		Label: "Environments",
		Data:  "environments",
		Children: []*ui.TreeNode{
			{Label: "Environment 1", Data: "environment1", Leaf: true},
			{Label: "Environment 2", Data: "environment2", Leaf: true},
			{Label: "Environment 3", Data: "environment3", Leaf: true},
		},
	}
	env.environments = ui.NewTree(root)
	env.environments.Background = &theme.Current().Panel
	return env
}

func (e *EnvironmentsPage) Layout(c *ui.Ctx) ui.View {
	return ui.Splitter("env-page-split", ui.Horizontal, e.sideBar(c), e.mainContent(c)).
		Sizes(300, 0).
		Grow(1)
}

func (e *EnvironmentsPage) sideBar(c *ui.Ctx) ui.View {
	th := c.Theme()
	return ui.Column(
		ui.Strong("Environments").Margin(th.Spacing.S),
		ui.Row(
			ui.Spacer(),
			ui.Button("import", ui.Text("Import")).Secondary(),
			ui.Button("new", ui.Text("New")).Primary().IconStart("add").OnClick(func() {
				n := e.nextEnv
				e.nextEnv++
				e.environments.AddChild(nil, &ui.TreeNode{
					Label: fmt.Sprintf("Environment %d", n),
					Data:  fmt.Sprintf("environment%d", n),
					Leaf:  true,
				})
			}),
		).Gap(th.Spacing.S).MarginTop(th.Spacing.S).MarginRight(th.Spacing.S),
		ui.TextField("env-search", e.query).
			Placeholder("Search...").
			IconStart("search").
			OnChange(func(s string) { e.query = s }).
			Margin(th.Spacing.S),
		ui.ViewOf(e.environments).Grow(1),
	).Gap(th.Spacing.S).Background(ui.TokenChrome).Grow(1)
}

func (e *EnvironmentsPage) mainContent(c *ui.Ctx) ui.View {
	th := c.Theme()
	return ui.Column(ui.Text("Environment")).Gap(th.Spacing.S)
}
