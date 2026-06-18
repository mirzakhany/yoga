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

// TodoApp is a simple todo list demo implementing yoga.Scene. It uses the Body
// pattern (layout.Host): persistent state lives in these fields, and body()
// derives the element tree from them. State changes just call host.Invalidate()
// — no manual MarkDirty/Calculate or .Children surgery.
type TodoApp struct {
	host   *layout.Host
	addBtn *components.Button
	input  *components.TextField
	list   *components.ListView
	items  []*components.Checkbox
	focus  *components.FocusManager
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
	app := &TodoApp{}

	app.input = components.NewTextField(components.TextFieldConfig{
		Placeholder: "Add a todo and press Enter...",
	}).WithIconStart("add")
	app.input.El.Grow(1)

	app.addBtn = components.NewButton("Add").Primary().Action(func() {
		app.addTodo(app.input.Value)
	})

	app.list = components.NewListView(components.ListViewConfig{})
	app.list.El.Grow(1)

	app.host = layout.NewHost(app.body)

	app.focus = components.NewFocusManager()
	app.focus.Add(app.input, app.addBtn)
	app.focus.Focus(app.input)

	return app
}

// body derives the element tree from current state. The runtime calls it (via
// host) on the first frame and after every Invalidate — never directly.
func (app *TodoApp) body() *layout.Element {
	th := theme.Current()
	title := components.NewLabel("Todos", components.LabelTitle)
	inputRow := layout.HStack(th.Spacing.S, app.input.El, app.addBtn.El)

	return layout.VStack(th.Spacing.M, title.El, inputRow, app.list.El).
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
	app.host.Invalidate()
}

func (app *TodoApp) clearInput() {
	if app.input.Value != "" {
		app.input.HandleKeys([]input.KeyEvent{{Key: input.KeyX, Mods: input.ModCtrl}})
	}
}

func (app *TodoApp) Root() *layout.Element { return app.host.Root() }

func (app *TodoApp) ClearColor() render.Color { return theme.Current().Background }

func (app *TodoApp) Layout(w, h float32) {
	components.SetViewport(w, h)
	app.host.Layout(w, h)
}

func (app *TodoApp) Update(m *input.Mouse, kb *input.Keyboard) {
	layout.Dispatch(app.host.Root(), m)
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
