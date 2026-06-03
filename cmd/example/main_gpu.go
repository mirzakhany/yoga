//go:build !nogpu

package main

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/cogentcore/webgpu/wgpuglfw"
	"github.com/go-gl/glfw/v3.3/glfw"

	"github.com/mirzakhany/yoga/components"
	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
)

func init() {
	// GLFW + WebGPU must run on the OS thread that created the window.
	runtime.LockOSThread()
}

func main() {
	if err := glfw.Init(); err != nil {
		panic(err)
	}
	defer glfw.Terminate()

	// Tell GLFW not to create an OpenGL context; WebGPU owns the surface.
	glfw.WindowHint(glfw.ClientAPI, glfw.NoAPI)
	window, err := glfw.CreateWindow(1100, 720, "Yoga UI — Dark Coding Workspace", nil, nil)
	if err != nil {
		panic(err)
	}
	defer window.Destroy()

	// Determine the device pixel scale (Retina/HiDPI): the framebuffer is
	// `scale` times larger than the logical window size.
	w, h := window.GetSize()
	fbW, fbH := window.GetFramebufferSize()
	scale := float32(1)
	if w > 0 {
		scale = float32(fbW) / float32(w)
	}

	// Build the font atlas once (CPU) at the device pixel scale so glyphs are
	// rasterized crisply, then hand it to the renderer (GPU upload) and to the
	// UI (text measurement / glyph quads, in logical units).
	atlas := render.NewMonoAtlasScale(scale)
	theme := components.DarkTheme()
	clip := &glfwClipboard{window: window}
	ws := BuildWorkspace(atlas, theme, clip)
	defer ws.Close()

	renderer, err := render.NewRenderer(wgpuglfw.GetSurfaceDescriptor(window), fbW, fbH, w, h, atlas)
	if err != nil {
		panic(err)
	}
	defer renderer.Destroy()
	renderer.ClearColor = theme.Background

	// --- Translate raw GLFW events into the platform-agnostic input types. ---
	mouse := &input.Mouse{}
	keyboard := &input.Keyboard{}
	window.SetCursorPosCallback(func(_ *glfw.Window, x, y float64) {
		mouse.SetPos(float32(x), float32(y))
	})
	window.SetMouseButtonCallback(func(_ *glfw.Window, button glfw.MouseButton, action glfw.Action, _ glfw.ModifierKey) {
		if button == glfw.MouseButtonLeft {
			mouse.SetButton(action == glfw.Press)
		}
	})
	window.SetScrollCallback(func(_ *glfw.Window, _, yoff float64) {
		mouse.AddScroll(float32(yoff))
	})
	// Text input: the char callback yields already-composed runes (layout/shift
	// applied), which is exactly what the editor wants to insert.
	window.SetCharCallback(func(_ *glfw.Window, char rune) {
		keyboard.TypeRune(char)
	})
	// Non-text keys (navigation/editing/shortcuts). Press and Repeat both count
	// so holding a key (e.g. arrow, backspace) auto-repeats.
	window.SetKeyCallback(func(_ *glfw.Window, key glfw.Key, _ int, action glfw.Action, mods glfw.ModifierKey) {
		if action == glfw.Release {
			return
		}
		if k, ok := mapKey(key); ok {
			keyboard.PressKey(k, mapMods(mods))
		}
	})
	window.SetSizeCallback(func(win *glfw.Window, width, height int) {
		nfbW, nfbH := win.GetFramebufferSize()
		renderer.Resize(nfbW, nfbH, width, height)
	})

	drawList := &render.DrawList{}

	for !window.ShouldClose() {
		glfw.PollEvents()

		fw, fh := window.GetSize()
		if fw == 0 || fh == 0 { // minimized
			continue
		}

		// Pass 1 + 2: solve the layout and flatten to absolute frames.
		ws.Layout(float32(fw), float32(fh))

		// State + input now that geometry exists for this frame.
		ws.Update(mouse, keyboard)
		mouse.EndFrame()
		keyboard.EndFrame()

		// Geometry generation: walk the tree, append into one DrawList.
		drawList.Reset()
		layout.Paint(ws.Root, drawList, atlas)

		// Single batched upload + draw.
		if err := renderer.Render(drawList); err != nil {
			if transientSurfaceError(err) {
				continue
			}
			fmt.Println("render error:", err)
		}
	}
}

// transientSurfaceError reports recoverable per-frame surface conditions that
// should simply skip the frame (window resize/minimize races, etc.).
func transientSurfaceError(err error) bool {
	s := err.Error()
	return strings.Contains(s, "Surface timed out") ||
		strings.Contains(s, "Surface is outdated") ||
		strings.Contains(s, "Surface was lost")
}

// mapKey translates the editing subset of GLFW keys into input.Key. Keys we do
// not care about return ok=false and are ignored.
func mapKey(k glfw.Key) (input.Key, bool) {
	switch k {
	case glfw.KeyLeft:
		return input.KeyLeft, true
	case glfw.KeyRight:
		return input.KeyRight, true
	case glfw.KeyUp:
		return input.KeyUp, true
	case glfw.KeyDown:
		return input.KeyDown, true
	case glfw.KeyHome:
		return input.KeyHome, true
	case glfw.KeyEnd:
		return input.KeyEnd, true
	case glfw.KeyBackspace:
		return input.KeyBackspace, true
	case glfw.KeyDelete:
		return input.KeyDelete, true
	case glfw.KeyEnter, glfw.KeyKPEnter:
		return input.KeyEnter, true
	case glfw.KeyTab:
		return input.KeyTab, true
	case glfw.KeyA:
		return input.KeyA, true
	case glfw.KeyC:
		return input.KeyC, true
	case glfw.KeyV:
		return input.KeyV, true
	case glfw.KeyX:
		return input.KeyX, true
	case glfw.KeyZ:
		return input.KeyZ, true
	case glfw.KeyS:
		return input.KeyS, true
	}
	return input.KeyNone, false
}

// mapMods translates GLFW modifier flags into input.Mod flags.
func mapMods(m glfw.ModifierKey) input.Mod {
	var mods input.Mod
	if m&glfw.ModShift != 0 {
		mods |= input.ModShift
	}
	if m&glfw.ModControl != 0 {
		mods |= input.ModCtrl
	}
	if m&glfw.ModAlt != 0 {
		mods |= input.ModAlt
	}
	if m&glfw.ModSuper != 0 {
		mods |= input.ModSuper
	}
	return mods
}

// glfwClipboard adapts the GLFW window clipboard to input.Clipboard so the
// editor (in the components package) stays free of any windowing dependency.
type glfwClipboard struct{ window *glfw.Window }

func (c *glfwClipboard) Get() string  { return c.window.GetClipboardString() }
func (c *glfwClipboard) Set(s string) { c.window.SetClipboardString(s) }
