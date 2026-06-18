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
	"github.com/mirzakhany/yoga/shape"
)

func init() {
	runtime.LockOSThread()
}

// App owns the window, GPU renderer, text engine, and input state.
type App struct {
	window   *glfw.Window
	text     *shape.Engine
	renderer *render.Renderer
	mouse    *input.Mouse
	keyboard *input.Keyboard
	clip     input.Clipboard
	scene    Scene
	host     *layout.Host // non-nil when scene is a View; owns tree rebuild/caching
	cursors  map[input.Cursor]*glfw.Cursor
	closed   bool
	drawList render.DrawList
}

// legacyScene is the pre-View contract: a scene that owns and solves its own
// element tree. Detected via assertion so the Scene interface stays minimal.
type legacyScene interface {
	Root() *layout.Element
	Layout(w, h float32)
}

// New creates the window, text engine, WebGPU renderer, and input wiring.
func New(cfg Config) (*App, error) {
	cfg = cfg.applyDefaults()

	if err := glfw.Init(); err != nil {
		return nil, fmt.Errorf("yoga: glfw init: %w", err)
	}

	glfw.WindowHint(glfw.ClientAPI, glfw.NoAPI)
	window, err := glfw.CreateWindow(cfg.Width, cfg.Height, cfg.Title, nil, nil)
	if err != nil {
		glfw.Terminate()
		return nil, fmt.Errorf("yoga: create window: %w", err)
	}

	w, h := window.GetSize()
	fbW, fbH := window.GetFramebufferSize()
	scale := float32(1)
	if w > 0 {
		scale = float32(fbW) / float32(w)
	}

	text, err := shape.NewEngine(scale, true)
	if err != nil {
		window.Destroy()
		glfw.Terminate()
		return nil, fmt.Errorf("yoga: text engine: %w", err)
	}

	renderer, err := render.NewRenderer(wgpuglfw.GetSurfaceDescriptor(window), fbW, fbH, w, h, text.Atlas)
	if err != nil {
		window.Destroy()
		glfw.Terminate()
		return nil, fmt.Errorf("yoga: create renderer: %w", err)
	}
	renderer.ClearColor = cfg.ClearColor

	clip := input.Clipboard(&glfwClipboard{window: window})
	icons := render.NewSpriteSheet(text.Atlas)
	a := &App{
		window:   window,
		text:     text,
		renderer: renderer,
		mouse:    &input.Mouse{},
		keyboard: &input.Keyboard{},
		clip:     clip,
	}
	a.initCursors()
	a.wireCallbacks()
	SetResources(text, icons, clip)
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
	a.window.SetCharCallback(func(_ *glfw.Window, char rune) {
		a.keyboard.TypeRune(char)
	})
	a.window.SetKeyCallback(func(_ *glfw.Window, key glfw.Key, _ int, action glfw.Action, mods glfw.ModifierKey) {
		if action == glfw.Release {
			return
		}
		if k, ok := mapKey(key); ok {
			a.keyboard.PressKey(k, mapMods(mods))
		}
	})
	a.window.SetFramebufferSizeCallback(func(win *glfw.Window, fbW, fbH int) {
		logicalW, logicalH := win.GetSize()
		if fbW <= 0 || fbH <= 0 {
			return
		}
		a.renderer.Resize(fbW, fbH, logicalW, logicalH)
		a.paintFrame(float32(logicalW), float32(logicalH))
	})
}

// Text returns the shaped text engine for constructing widgets.
func (a *App) Text() *shape.Engine { return a.text }

// Atlas returns the glyph atlas (legacy accessor).
func (a *App) Atlas() *render.FontAtlas { return a.text.Atlas }

func (a *App) Clipboard() input.Clipboard { return a.clip }
func (a *App) Window() *glfw.Window       { return a.window }
// SetScene installs the scene. For a View, it creates the layout.Host that
// caches and rebuilds the tree, and hands it to the scene via Attach.
func (a *App) SetScene(s Scene) {
	a.scene = s
	a.host = nil
	if v, ok := s.(View); ok {
		a.host = layout.NewHost(v.Body)
		if at, ok := s.(Attacher); ok {
			at.Attach(a.host)
		}
	}
}

// sceneRoot returns the scene's current tree solved for the given size: a View's
// host rebuilds only if invalidated, while a legacy scene solves its own tree.
func (a *App) sceneRoot(w, h float32) *layout.Element {
	if a.host != nil {
		a.host.Layout(w, h)
		return a.host.Root()
	}
	if ls, ok := a.scene.(legacyScene); ok {
		ls.Layout(w, h)
		return ls.Root()
	}
	return nil
}

// paintFrame re-solves layout and submits a GPU frame for the given logical size.
// Safe to call from inside a GLFW callback (same OS thread as Run).
func (a *App) paintFrame(logicalW, logicalH float32) {
	if a.scene == nil || logicalW <= 0 || logicalH <= 0 {
		return
	}
	root := a.sceneRoot(logicalW, logicalH)
	if root == nil {
		return
	}
	a.drawList.Reset()
	layout.Paint(root, &a.drawList, a.text)
	_ = a.text.FlushAtlas(a.renderer)
	if err := a.renderer.Render(&a.drawList); err != nil && !transientSurfaceError(err) {
		fmt.Println("render error:", err)
	}
}

func (a *App) Run() {
	for !a.window.ShouldClose() {
		fw, fh := a.window.GetSize()
		if fw > 0 && fh > 0 && a.scene != nil {
			a.mouse.Cursor = input.CursorDefault
			a.scene.Update(a.mouse, a.keyboard)
			a.applyCursor()
			a.mouse.EndFrame()
			a.keyboard.EndFrame()

			if cc, ok := a.scene.(interface{ ClearColor() render.Color }); ok {
				a.renderer.ClearColor = cc.ClearColor()
			}

			a.paintFrame(float32(fw), float32(fh))
		}

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

func transientSurfaceError(err error) bool {
	s := err.Error()
	return strings.Contains(s, "Surface timed out") ||
		strings.Contains(s, "Surface is outdated") ||
		strings.Contains(s, "Surface was lost")
}

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
	case glfw.KeyF:
		return input.KeyF, true
	case glfw.KeyH:
		return input.KeyH, true
	case glfw.KeyEscape:
		return input.KeyEscape, true
	case glfw.KeySpace:
		return input.KeySpace, true
	}
	return input.KeyNone, false
}

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

type glfwClipboard struct{ window *glfw.Window }

func (c *glfwClipboard) Get() string  { return c.window.GetClipboardString() }
func (c *glfwClipboard) Set(s string) { c.window.SetClipboardString(s) }
