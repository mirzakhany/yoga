//go:build nogpu

package main

import (
	"fmt"

	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
)

// Headless entry point for `go build -tags nogpu`. It exercises the entire
// CPU-side pipeline (atlas baking, layout, input dispatch, geometry generation)
// without linking wgpu-native or opening a window, then reports the geometry it
// would have drawn. Useful for CI and for machines where the GPU stack is
// unavailable.
func main() {
	const w, h = 1100, 720

	atlas := render.NewMonoAtlas()
	clip := &input.MemClipboard{}
	ws := BuildWorkspace(atlas, clip)
	defer ws.Close()

	mouse := &input.Mouse{}
	keyboard := &input.Keyboard{}
	drawList := &render.DrawList{}

	ws.Layout(w, h)

	// Exercise the editing path headlessly: type some text into the active doc.
	keyboard.TypeRune('h')
	keyboard.TypeRune('i')
	keyboard.PressKey(input.KeyEnter, 0)

	ws.Update(mouse, keyboard)
	mouse.EndFrame()
	keyboard.EndFrame()

	drawList.Reset()
	layout.Paint(ws.Root(), drawList, atlas)

	ed := ws.active2()
	fmt.Printf("headless frame: %d vertices, %d indices (atlas %dx%d)\n",
		len(drawList.Vertices), len(drawList.Indices), atlas.W, atlas.H)
	fmt.Printf("editor: content height %.0fpx, modified=%v, %d bytes\n",
		ed.ContentHeight, ed.Modified(), len(ed.Bytes()))
}
