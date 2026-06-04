//go:build !nogpu

package yoga

import (
	"fmt"
	"runtime"
	"strings"
	"time"

	"github.com/cogentcore/webgpu/wgpuglfw"
	"github.com/go-gl/glfw/v3.3/glfw"

	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
)

func init() {
	// GLFW + WebGPU must run on the OS thread that created the window.
	runtime.LockOSThread()
}

// App owns the window, GPU renderer, font atlas, and input state, and runs the
// per-frame loop that drives a Scene. Create it with New, hand it a Scene with
// SetScene, then call Run.
type App struct {
	window   *glfw.Window
	atlas    *render.FontAtlas
	renderer *render.Renderer
	mouse    *input.Mouse
	keyboard *input.Keyboard
	clip     input.Clipboard
	scene    Scene
	cursors  map[input.Cursor]*glfw.Cursor
	closed   bool
}

// New creates the window, bakes the font atlas at the display's pixel density,
// initializes the WebGPU renderer, and wires GLFW input callbacks. The returned
// App has no Scene yet; call SetScene before Run.
func New(cfg Config) (*App, error) {
	cfg = cfg.applyDefaults()

	if err := glfw.Init(); err != nil {
		return nil, fmt.Errorf("yoga: glfw init: %w", err)
	}

	// Tell GLFW not to create an OpenGL context; WebGPU owns the surface.
	glfw.WindowHint(glfw.ClientAPI, glfw.NoAPI)
	window, err := glfw.CreateWindow(cfg.Width, cfg.Height, cfg.Title, nil, nil)
	if err != nil {
		glfw.Terminate()
		return nil, fmt.Errorf("yoga: create window: %w", err)
	}

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

	renderer, err := render.NewRenderer(wgpuglfw.GetSurfaceDescriptor(window), fbW, fbH, w, h, atlas)
	if err != nil {
		window.Destroy()
		glfw.Terminate()
		return nil, fmt.Errorf("yoga: create renderer: %w", err)
	}
	renderer.ClearColor = cfg.ClearColor

	a := &App{
		window:   window,
		atlas:    atlas,
		renderer: renderer,
		mouse:    &input.Mouse{},
		keyboard: &input.Keyboard{},
		clip:     &glfwClipboard{window: window},
	}
	a.initCursors()
	a.wireCallbacks()
	return a, nil
}

func (a *App) initCursors() {
	shapes := map[input.Cursor]glfw.StandardCursor{
		input.CursorDefault:  glfw.ArrowCursor,
		input.CursorResizeEW: glfw.HResizeCursor,
		input.CursorResizeNS: glfw.VResizeCursor,
	}
	a.cursors = make(map[input.Cursor]*glfw.Cursor, len(shapes))
	for kind, shape := range shapes {
		a.cursors[kind] = glfw.CreateStandardCursor(shape)
	}
}

func (a *App) applyCursor() {
	c := a.cursors[a.mouse.Cursor]
	if c == nil {
		c = a.cursors[input.CursorDefault]
	}
	a.window.SetCursor(c)
}

// wireCallbacks translates raw GLFW events into the platform-agnostic input
// types owned by the App.
func (a *App) wireCallbacks() {
	a.window.SetCursorPosCallback(func(_ *glfw.Window, x, y float64) {
		a.mouse.SetPos(float32(x), float32(y))
	})
	a.window.SetMouseButtonCallback(func(_ *glfw.Window, button glfw.MouseButton, action glfw.Action, _ glfw.ModifierKey) {
		switch button {
		case glfw.MouseButtonLeft:
			a.mouse.SetButton(action == glfw.Press)
		case glfw.MouseButtonRight:
			a.mouse.SetRightButton(action == glfw.Press)
		}
	})
	a.window.SetScrollCallback(func(_ *glfw.Window, xoff, yoff float64) {
		a.mouse.AddScrollX(float32(xoff))
		a.mouse.AddScroll(float32(yoff))
	})
	// Text input: the char callback yields already-composed runes (layout/shift
	// applied), which is exactly what an editor wants to insert.
	a.window.SetCharCallback(func(_ *glfw.Window, char rune) {
		a.keyboard.TypeRune(char)
	})
	// Non-text keys (navigation/editing/shortcuts). Press and Repeat both count
	// so holding a key (e.g. arrow, backspace) auto-repeats.
	a.window.SetKeyCallback(func(_ *glfw.Window, key glfw.Key, _ int, action glfw.Action, mods glfw.ModifierKey) {
		if action == glfw.Release {
			return
		}
		if k, ok := mapKey(key); ok {
			a.keyboard.PressKey(k, mapMods(mods))
		}
	})
	a.window.SetSizeCallback(func(win *glfw.Window, width, height int) {
		nfbW, nfbH := win.GetFramebufferSize()
		a.renderer.Resize(nfbW, nfbH, width, height)
	})
}

// Atlas returns the baked font atlas, needed to construct widgets/scenes.
func (a *App) Atlas() *render.FontAtlas { return a.atlas }

// Clipboard returns the system clipboard adapter for the scene to use.
func (a *App) Clipboard() input.Clipboard { return a.clip }

// Window returns the underlying GLFW window as an escape hatch for advanced use.
func (a *App) Window() *glfw.Window { return a.window }

// SetScene installs the UI to drive. It may be called before Run or to swap the
// active scene between frames.
func (a *App) SetScene(s Scene) { a.scene = s }

// Run executes the blocking frame loop until the window is closed. It must be
// called on the main goroutine (the same one that called New).
//
// The loop is event-driven: it renders one frame, then SLEEPS in glfw until the
// next input/resize event or, for animating scenes, until the next animation
// tick. This keeps idle CPU near zero instead of busy-rendering at the vsync
// rate. Input callbacks run during the wait, so each event wakes the loop and
// produces exactly one repaint.
func (a *App) Run() {
	drawList := &render.DrawList{}

	for !a.window.ShouldClose() {
		fw, fh := a.window.GetSize()
		if fw > 0 && fh > 0 && a.scene != nil { // skip drawing while minimized
			// Pass 1 + 2: solve the layout and flatten to absolute frames.
			a.scene.Layout(float32(fw), float32(fh))

			// State + input now that geometry exists for this frame.
			a.mouse.Cursor = input.CursorDefault
			a.scene.Update(a.mouse, a.keyboard)
			a.applyCursor()
			a.mouse.EndFrame()
			a.keyboard.EndFrame()

			// Let the scene drive the clear color each frame (so a runtime theme
			// switch updates the window background immediately). Optional capability.
			if cc, ok := a.scene.(interface{ ClearColor() render.Color }); ok {
				a.renderer.ClearColor = cc.ClearColor()
			}

			// Geometry generation: walk the tree, append into one DrawList.
			drawList.Reset()
			layout.Paint(a.scene.Root(), drawList, a.atlas)

			// Single batched upload + draw.
			if err := a.renderer.Render(drawList); err != nil && !transientSurfaceError(err) {
				fmt.Println("render error:", err)
			}
		}

		// Sleep until the next event (or animation deadline) instead of spinning.
		if d, ok := a.sceneWait(); ok {
			if d < 0 {
				d = 0
			}
			glfw.WaitEventsTimeout(d.Seconds())
		} else {
			glfw.WaitEvents()
		}
	}
}

// sceneWait reports how long the loop may sleep before the next time-based
// repaint, honoring the scene's optional Animator capability. ok=false means the
// scene has no pending animation and the loop should block until an event.
func (a *App) sceneWait() (time.Duration, bool) {
	if a.scene == nil {
		return 0, false
	}
	if an, ok := a.scene.(interface {
		AnimationWait() (time.Duration, bool)
	}); ok {
		return an.AnimationWait()
	}
	return 0, false
}

// Close releases the scene, the GPU resources, and the window. It is safe to
// call more than once.
func (a *App) Close() {
	if a.closed {
		return
	}
	a.closed = true
	if a.scene != nil {
		a.scene.Close()
	}
	for _, c := range a.cursors {
		if c != nil {
			c.Destroy()
		}
	}
	if a.renderer != nil {
		a.renderer.Destroy()
	}
	if a.window != nil {
		a.window.Destroy()
	}
	glfw.Terminate()
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

// glfwClipboard adapts the GLFW window clipboard to input.Clipboard so widgets
// (in the components package) stay free of any windowing dependency.
type glfwClipboard struct{ window *glfw.Window }

func (c *glfwClipboard) Get() string  { return c.window.GetClipboardString() }
func (c *glfwClipboard) Set(s string) { c.window.SetClipboardString(s) }
