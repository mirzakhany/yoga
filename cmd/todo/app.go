// Command todo is a minimal todo-list demo built on the Yoga UI framework.
package main

import (
	"strings"
	"time"

	"github.com/mirzakhany/yoga"
	"github.com/mirzakhany/yoga/components"
	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/theme"
)

const textFieldBlink = 500 * time.Millisecond

// TodoApp is a simple todo list demo implementing yoga.Scene.
type TodoApp struct {
	root  *layout.Element
	input *components.TextField
	list  *components.ListView
	items []*components.Checkbox
	focus *components.FocusManager
	lastW float32
	lastH float32
}

var _ yoga.Scene = (*TodoApp)(nil)

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

// BuildTodoApp assembles the todo list scene.
func BuildTodoApp() *TodoApp {
	th := theme.Current()
	app := &TodoApp{}

	app.input = components.NewTextField(components.TextFieldConfig{
		Placeholder: "Add a todo and press Enter...",
	}).WithIconStart("add")
	app.input.El.Style = app.input.El.Style.FlexGrow(1)

	addBtn := components.NewButton("Add").Primary().Action(func() {
		app.addTodo(app.input.Value)
	})

	inputRow := layout.HStack(th.Spacing.S, app.input.El, addBtn.El)

	app.list = components.NewListView(components.ListViewConfig{})
	app.list.El.Style = app.list.El.Style.FlexGrow(1)

	title := components.NewLabel("Todos", components.LabelTitle)

	app.root = layout.New(layout.Box().Direction(layout.Column).FlexGrow(1).Gap(th.Spacing.M).PaddingAll(th.Spacing.L),
		title.El,
		inputRow,
		app.list.El,
	).WithBackgroundPtr(&th.Background)

	app.focus = components.NewFocusManager()
	app.focus.Add(app.input, addBtn)
	app.focus.Focus(app.input)

	return app
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
	app.relayout()
}

func (app *TodoApp) clearInput() {
	if app.input.Value != "" {
		app.input.HandleKeys([]input.KeyEvent{{Key: input.KeyX, Mods: input.ModCtrl}})
	}
}

func (app *TodoApp) relayout() {
	app.root.MarkDirty()
	if app.lastW > 0 && app.lastH > 0 {
		app.root.Calculate(app.lastW, app.lastH)
	}
}

func (app *TodoApp) Root() *layout.Element { return app.root }

func (app *TodoApp) ClearColor() render.Color { return theme.Current().Background }

func (app *TodoApp) Layout(w, h float32) {
	app.lastW, app.lastH = w, h
	components.SetViewport(w, h)
	app.root.Calculate(w, h)
}

func (app *TodoApp) Update(m *input.Mouse, kb *input.Keyboard) {
	layout.Dispatch(app.root, m)
	app.input.Update(m)
	app.list.Update(m)

	if app.focus != nil {
		app.focus.HandleMouse(m)
		if kb != nil {
			app.focus.Route(kb)
			if app.input.Focused() {
				for _, ev := range kb.Keys {
					if ev.Key == input.KeyEnter {
						app.addTodo(app.input.Value)
						break
					}
				}
			}
		}
	}
}

func (app *TodoApp) AnimationWait() (time.Duration, bool) {
	if app.input.Focused() {
		return textFieldBlink, true
	}
	return 0, false
}

func (app *TodoApp) Close() {}
