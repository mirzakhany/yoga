// Command todo is a minimal todo-list demo built on the Yoga UI framework.
package main

import (
	"strings"

	"github.com/mirzakhany/yoga"
	"github.com/mirzakhany/yoga/components"
	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/theme"
	"github.com/mirzakhany/yoga/ui"
)

// TodoApp is a simple todo list demo implementing yoga.App. Persistent state
// lives in these fields; Body derives the element tree from them each frame.
type TodoApp struct {
	addBtn *components.Button
	input  *components.TextField
	list   *components.ListView
	items  []*components.Checkbox
}

var _ yoga.App = (*TodoApp)(nil)

func todoLabelStyle(done bool) components.CheckboxLabelStyle {
	if !done {
		return components.CheckboxLabelStyle{}
	}
	return components.CheckboxLabelStyle{Muted: true, Strikethrough: true}
}

func newTodoItem(title string) *components.Checkbox {
	cb := components.NewCheckbox(title)
	cb.Changed(func(checked bool) {
		cb.SetLabelStyle(todoLabelStyle(checked))
	})
	return cb
}

// BuildTodoApp assembles the todo list (retained state, built once).
func BuildTodoApp() *TodoApp {
	app := &TodoApp{}

	app.input = components.NewTextField(components.TextFieldConfig{
		Placeholder: "Add a todo and press Enter...",
	}).WithIconStart("add")
	app.input.El.Grow(1)
	app.input.OnSubmit = func(v string) { app.addTodo(v) }

	app.addBtn = components.NewButton("Add").Primary().Action(func() {
		app.addTodo(app.input.Value)
	})

	app.list = components.NewListView(components.ListViewConfig{})
	app.list.El.Grow(1)

	return app
}

// Body derives the element tree from current state, registering focus and
// caret animation through the context via each component's Layout(c).
func (app *TodoApp) Body(c *ui.Ctx) *ui.Element {
	th := theme.Current()
	title := components.NewLabel("Todos", components.LabelTitle)

	inputRow := ui.HStack(app.input.Layout(c).Grow(1), app.addBtn.Layout(c)).Gap(th.Spacing.S)

	// Default focus to the input so typing/Enter works immediately. Enter-to-add
	// is wired via app.input.OnSubmit.
	c.Focus().EnsureFocus(app.input)

	return ui.VStack(title.Layout(c), inputRow, app.list.Layout(c)).
		Gap(th.Spacing.M).
		Grow(1).
		Padding(th.Spacing.L).
		BgPtr(&th.Background)
}

func (app *TodoApp) addTodo(text string) {
	text = strings.TrimSpace(text)
	if text == "" {
		return
	}
	cb := newTodoItem(text)
	app.items = append(app.items, cb)
	app.list.Add(cb.El)
	app.clearInput()
}

func (app *TodoApp) clearInput() {
	if app.input.Value != "" {
		app.input.HandleKeys([]input.KeyEvent{{Key: input.KeyX, Mods: input.ModCtrl}})
	}
}

func (app *TodoApp) ClearColor() render.Color { return theme.Current().Background }
