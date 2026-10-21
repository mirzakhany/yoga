package main

import (
	"fmt"
	"strconv"

	"github.com/mirzakhany/yoga/icons"
	"github.com/mirzakhany/yoga/theme"
	"github.com/mirzakhany/yoga/ui"
)

// environment is one set of variables, tinted with a categorical hue so it
// is recognizable in the tree, the top bar and its own header.
type environment struct {
	id   string
	name string
	hue  ui.Hue
	vars *ui.Table
	node *ui.TreeNode
}

type EnvironmentsPage struct {
	query        string
	environments *ui.Tree
	envs         []*environment
	current      *environment
	nextEnv      int
	nextVar      int
}

// envHues cycles new environments through the categorical hues.
var envHues = []ui.Hue{ui.HueRed, ui.HueOrange, ui.HueGreen, ui.HueBlue, ui.HuePurple, ui.HueTeal, ui.HuePink, ui.HueYellow}

func NewEnvironmentsPage() *EnvironmentsPage {
	e := &EnvironmentsPage{}
	root := &ui.TreeNode{Label: "Environments"}
	e.environments = ui.NewTree(root)
	e.environments.Background = &theme.Current().Panel
	// A ring in the environment's hue in place of the icon, and the
	// variable count on the right.
	e.environments.Leading = func(n *ui.TreeNode) ui.View {
		env, _ := n.Data.(*environment)
		if env == nil {
			return nil
		}
		th := theme.Current()
		return ui.Icon(icons.Circle, th.Metrics.TreeIconSize, th.Categorical[env.hue])
	}
	e.environments.Trailing = func(n *ui.TreeNode) ui.View {
		env, _ := n.Data.(*environment)
		if env == nil {
			return nil
		}
		return ui.Caption(strconv.Itoa(len(env.vars.Rows))).Style(ui.Spec{}.TextColor(ui.TokenForegroundMuted))
	}
	e.environments.OnActivate = func(n *ui.TreeNode) {
		if env, ok := n.Data.(*environment); ok {
			e.selectEnv(env)
		}
	}

	e.addEnv("Production", [][3]string{
		{"baseUrl", "https://api.example.com", ""},
		{"apiKey", "sk_live_51HxQ2", "secret"},
		{"timeout", "30s", ""},
	})
	e.addEnv("Staging", [][3]string{
		{"baseUrl", "https://staging.example.com", ""},
		{"apiKey", "sk_test_4eC39H", "secret"},
	})
	e.addEnv("Local", [][3]string{
		{"baseUrl", "http://localhost:8080", ""},
	})
	e.selectEnv(e.envs[0])
	return e
}

// addEnv adds an environment with vars given as name, value, flags rows
// (flags "secret" masks the value).
func (e *EnvironmentsPage) addEnv(name string, vars [][3]string) *environment {
	env := &environment{
		id:   fmt.Sprintf("environment%d", e.nextEnv+1),
		name: name,
		hue:  envHues[e.nextEnv%len(envHues)],
	}
	e.nextEnv++
	env.vars = newVarsTable()
	for _, v := range vars {
		e.addVar(env, v[0], v[1], v[2])
	}
	env.node = &ui.TreeNode{Label: name, Data: env, Leaf: true}
	e.envs = append(e.envs, env)
	e.environments.AddChild(nil, env.node)
	return env
}

func (e *EnvironmentsPage) addVar(env *environment, name, value, flags string) {
	e.nextVar++
	env.vars.AddRow(ui.TableRow{
		ID:    fmt.Sprintf("var-%d", e.nextVar),
		Cells: map[string]string{"key": name, "val": value, "flags": flags},
	})
}

// newVarsTable lists an environment's variables. Values flagged secret are
// masked, with a per-cell eye to show one; the delete button only shows on
// the hovered row.
func newVarsTable() *ui.Table {
	t := ui.NewTable([]ui.TableColumn{
		{ID: "key", Label: "Variable", Kind: ui.TableColEditable, Sortable: true},
		{ID: "val", Label: "Value", Kind: ui.TableColSecret,
			SecretWhen: func(r ui.TableRow) bool { return r.Cells["flags"] == "secret" }},
		{ID: "flags", Label: "Flags", Kind: ui.TableColBadges, Width: 120},
		{ID: "act", Kind: ui.TableColActions, Width: 40, Locked: true},
	}, []ui.TableAction{{Icon: icons.Trash2, Tooltip: "Delete"}})
	t.ActionsOnHover = true
	t.Actions[0].OnClick = func(id string) { t.RemoveRow(id) }
	return t
}

func (e *EnvironmentsPage) selectEnv(env *environment) {
	e.current = env
	e.environments.SetActive(env.node)
}

// options lists the environments for the top bar's selector.
func (e *EnvironmentsPage) options() []ui.SelectOption {
	out := make([]ui.SelectOption, len(e.envs))
	for i, env := range e.envs {
		out[i] = ui.SelectOption{Label: env.name, Value: env.id}
	}
	return out
}

// hueOf is the hue of the environment with the given id.
func (e *EnvironmentsPage) hueOf(id string) (ui.Hue, bool) {
	for _, env := range e.envs {
		if env.id == id {
			return env.hue, true
		}
	}
	return 0, false
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
			ui.Button("new", ui.Text("New")).Primary().IconStart(icons.Plus).OnClick(func() {
				env := e.addEnv(fmt.Sprintf("Environment %d", e.nextEnv+1), nil)
				e.selectEnv(env)
				// Scroll the new row into view and highlight it.
				e.environments.Reveal(env.node)
			}),
		).Gap(th.Spacing.S).MarginTop(th.Spacing.S).MarginRight(th.Spacing.S),
		ui.TextField("env-search", e.query).
			Placeholder("Search...").
			IconStart(icons.Search).
			OnChange(func(s string) {
				e.query = s
				e.environments.SetFilter(s)
			}).
			Margin(th.Spacing.S),
		ui.ViewOf(e.environments).Grow(1),
	).Gap(th.Spacing.S).Background(ui.TokenChrome).Grow(1)
}

func (e *EnvironmentsPage) mainContent(c *ui.Ctx) ui.View {
	th := c.Theme()
	env := e.current
	if env == nil {
		return ui.Column(ui.Muted("Select an environment")).Padding(th.Spacing.L).Grow(1)
	}
	return ui.Column(
		ui.Row(
			ui.Title(env.name),
			ui.Tag(env.id).Hue(env.hue),
			ui.Spacer(),
			ui.Button("add-var", ui.Text("Add variable")).IconStart(icons.Plus).OnClick(func() {
				e.addVar(env, "newVariable", "", "")
			}),
		).Gap(th.Spacing.S).Align(ui.AlignCenter),
		ui.ViewOf(env.vars).Grow(1),
	).Gap(th.Spacing.M).Padding(th.Spacing.L).Grow(1)
}
