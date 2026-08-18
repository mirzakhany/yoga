package main

import (
	"fmt"
	"strings"

	"github.com/mirzakhany/yoga"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/theme"
	"github.com/mirzakhany/yoga/ui"
)

type todoItem struct {
	id    string
	title string
	done  bool
}

// TodoApp is a simple todo list demo implementing yoga.App.
type TodoApp struct {
	draft string
	items []todoItem
	next  int
}

var _ yoga.App = (*TodoApp)(nil)

func BuildTodoApp() *TodoApp { return &TodoApp{} }

func (app *TodoApp) Body(c *ui.Ctx) ui.View {
	th := c.Theme()
	rows := make([]ui.View, 0, len(app.items))
	for _, it := range app.items {
		item := it
		cb := ui.Checkbox(item.id, item.title).
			Check(item.done).
			OnToggle(func(v bool) { app.setDone(item.id, v) })
		if item.done {
			cb.LabelMuted(true).LabelStrike(true)
		}
		rows = append(rows, cb)
	}

	return ui.Column(
		ui.Title("Todos"),
		ui.Row(
			ui.TextField("draft", app.draft).
				Placeholder("Add a todo and press Enter...").
				IconStart("add").
				OnChange(func(s string) { app.draft = s }).
				OnSubmit(func(s string) { app.addTodo(s) }).
				DefaultFocus().
				Grow(1),
			ui.Button("add", ui.Text("Add")).Primary().OnClick(func() { app.addTodo(app.draft) }),
		).Gap(th.Spacing.S),
		ui.Column(rows...).Gap(th.Spacing.S).Grow(1),
	).Gap(th.Spacing.M).Grow(1).Padding(th.Spacing.L).Background(ui.TokenSurface)
}

func (app *TodoApp) addTodo(text string) {
	text = strings.TrimSpace(text)
	if text == "" {
		return
	}
	app.next++
	app.items = append(app.items, todoItem{
		id:    fmt.Sprintf("todo-%d", app.next),
		title: text,
	})
	app.draft = ""
}

func (app *TodoApp) setDone(id string, done bool) {
	for i := range app.items {
		if app.items[i].id == id {
			app.items[i].done = done
			return
		}
	}
}

func (app *TodoApp) ClearColor() render.Color { return theme.Current().Background }
