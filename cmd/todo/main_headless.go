//go:build nogpu

package main

import (
	"fmt"

	"github.com/mirzakhany/yoga"
	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
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

	// Drive the View like the runtime does: a layout.Host owns rebuild/caching.
	host := layout.NewHost(todo.Body)
	todo.Attach(host)

	mouse := &input.Mouse{}
	keyboard := &input.Keyboard{}

	host.Layout(w, h)

	for _, ch := range "Buy milk" {
		keyboard.TypeRune(ch)
	}
	keyboard.PressKey(input.KeyEnter, 0)

	todo.Update(mouse, keyboard)
	mouse.EndFrame()
	keyboard.EndFrame()

	host.Layout(w, h) // reflect state changed during Update
	drawList := &render.DrawList{}
	layout.Paint(host.Root(), drawList, text)

	fmt.Printf("headless frame: %d vertices, %d indices\n", len(drawList.Vertices), len(drawList.Indices))
	fmt.Printf("todos: %d items\n", len(todo.items))
	if len(todo.items) > 0 {
		fmt.Printf("first todo: %q done=%v\n", todo.items[0].Label, todo.items[0].Checked)
	}
}
