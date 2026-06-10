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
	const w, h = 1100, 720

	text, err := shape.NewEngine(1, false)
	if err != nil {
		panic(err)
	}
	clip := &input.MemClipboard{}
	sheet := render.NewSpriteSheet(text.Atlas)
	yoga.SetResources(text, sheet, clip)

	ws := BuildWorkspace()
	defer ws.Close()

	mouse := &input.Mouse{}
	keyboard := &input.Keyboard{}
	drawList := &render.DrawList{}

	ws.Layout(w, h)

	keyboard.TypeRune('h')
	keyboard.TypeRune('i')
	keyboard.PressKey(input.KeyEnter, 0)

	ws.Update(mouse, keyboard)
	mouse.EndFrame()
	keyboard.EndFrame()

	drawList.Reset()
	layout.Paint(ws.Root(), drawList, text)

	ed := ws.active2()
	mw, mh := text.Atlas.MonoSize()
	fmt.Printf("headless frame: %d vertices, %d indices (atlas %dx%d)\n",
		len(drawList.Vertices), len(drawList.Indices), mw, mh)
	fmt.Printf("editor: content height %.0fpx, modified=%v, %d bytes\n",
		ed.ContentHeight, ed.Modified(), len(ed.Bytes()))
}
