//go:build nogpu

package main

import (
	"fmt"

	"github.com/yogaui/yoga"
	"github.com/yogaui/yoga/input"
	"github.com/yogaui/yoga/layout"
	"github.com/yogaui/yoga/render"
	"github.com/yogaui/yoga/shape"
	"github.com/yogaui/yoga/ui"
)

func main() {
	const w, h = 520, 640

	text, err := shape.NewEngine(1, false)
	if err != nil {
		panic(err)
	}
	clip := &input.MemClipboard{}
	sheet := render.NewSpriteSheet(text.Atlas)
	yoga.SetResources(text, sheet, clip)

	todo := BuildTodoApp()
	c := ui.New(text, ui.NewFocusScope(), nil)
	c.SetIcons(sheet)
	c.SetClipboard(clip)

	mouse := &input.Mouse{}
	keyboard := &input.Keyboard{}

	for _, ch := range "Buy milk" {
		keyboard.TypeRune(ch)
	}
	keyboard.PressKey(input.KeyEnter, 0)

	root := ui.BuildFrame(c, todo.Body, w, h, mouse, keyboard)
	layout.Dispatch(root, mouse)
	c.Focus().Route(keyboard)
	mouse.EndFrame()
	keyboard.EndFrame()

	// Second frame reflects state changed during the first.
	root = ui.BuildFrame(c, todo.Body, w, h, mouse, keyboard)
	drawList := &render.DrawList{}
	layout.Paint(root, drawList, text)

	fmt.Printf("headless frame: %d vertices, %d indices\n", len(drawList.Vertices), len(drawList.Indices))
	fmt.Printf("todos: %d items\n", len(todo.items))
	if len(todo.items) > 0 {
		fmt.Printf("first todo: %q done=%v\n", todo.items[0].title, todo.items[0].done)
	}
}
