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
	const w, h = 900, 700

	text, err := shape.NewEngine(1, false)
	if err != nil {
		panic(err)
	}
	clip := &input.MemClipboard{}
	sheet := render.NewSpriteSheet(text.Atlas)
	yoga.SetResources(text, sheet, clip)

	api := BuildAPITestApp()

	mouse := &input.Mouse{}
	keyboard := &input.Keyboard{}

	api.Layout(w, h)
	api.Update(mouse, keyboard)
	mouse.EndFrame()
	keyboard.EndFrame()

	drawList := &render.DrawList{}
	layout.Paint(api.Root(), drawList, text)

	fmt.Printf("headless frame: %d vertices, %d indices\n", len(drawList.Vertices), len(drawList.Indices))
	fmt.Printf("status: %s\n", api.StatusText())
}
