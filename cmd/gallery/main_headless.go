//go:build nogpu

package main

import (
	"fmt"

	"github.com/mirzakhany/yoga"
	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
	"github.com/mirzakhany/yoga/ui"
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

	// Drive one frame through the same ui build path the GPU runtime uses.
	c := ui.New(text, ui.NewFocusScope(), nil)
	c.SetIcons(sheet)
	c.SetClipboard(clip)

	keyboard.TypeRune('h')
	keyboard.TypeRune('i')
	keyboard.PressKey(input.KeyEnter, 0)

	root := ui.BuildFrame(c, ws.Layout, w, h, mouse, keyboard)
	layout.Dispatch(root, mouse)
	c.Focus().Route(keyboard)
	mouse.EndFrame()
	keyboard.EndFrame()

	drawList.Reset()
	layout.Paint(root, drawList, text)

	ed := ws.activeDoc()
	mw, mh := text.Atlas.MonoSize()
	fmt.Printf("headless frame: %d vertices, %d indices (atlas %dx%d)\n",
		len(drawList.Vertices), len(drawList.Indices), mw, mh)
	fmt.Printf("editor: content height %.0fpx, modified=%v, %d bytes\n",
		ed.ContentHeight, ed.Modified(), len(ed.Bytes()))
}
