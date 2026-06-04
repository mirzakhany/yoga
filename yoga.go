// Package yoga is the application entry layer of the framework. It hides the
// platform boilerplate (GLFW window creation, HiDPI scaling, the WebGPU
// renderer, font-atlas baking, input wiring, and the per-frame loop) behind a
// tiny API:
//
//	app, err := yoga.New(yoga.Config{Title: "My App", Width: 1100, Height: 720})
//	if err != nil { panic(err) }
//	defer app.Close()
//	app.SetScene(myScene)   // anything implementing yoga.Scene
//	app.Run()               // blocks until the window closes
//
// Layering: yoga sits above render/input/layout and deliberately does NOT import
// the components package, so application widgets stay decoupled from the runtime.
// An application supplies its own theme and only hands yoga a ClearColor.
//
// Threading: the OS event/render loop must run on the thread that created the
// window (a hard requirement of GLFW on macOS). yoga.New locks the OS thread and
// app.Run blocks the calling (main) goroutine for the lifetime of the window.
// Application/background work may still run on other goroutines, but all window,
// input, and rendering calls happen on the main goroutine inside Run.
package yoga

import (
	"time"

	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
)

// Config describes the window to create. Zero fields fall back to the defaults
// applied by New (see applyDefaults).
type Config struct {
	Title         string
	Width, Height int
	// ClearColor is the framebuffer clear color (typically the app theme's
	// background). When left as the zero value (alpha 0) a dark default is used.
	ClearColor render.Color
}

// applyDefaults fills any unset Config fields with sensible defaults.
func (c Config) applyDefaults() Config {
	if c.Title == "" {
		c.Title = "Yoga"
	}
	if c.Width <= 0 {
		c.Width = 1100
	}
	if c.Height <= 0 {
		c.Height = 720
	}
	if c.ClearColor.A == 0 {
		c.ClearColor = render.RGBA8(24, 24, 29, 255)
	}
	return c
}

// Scene is the contract between the runtime and an application's UI. The example
// Workspace implements it. Per frame, the app calls Layout (solve the tree for
// the current viewport), then Update (input + state), then paints Root.
type Scene interface {
	// Root returns the root element of the UI tree to paint and hit-test.
	Root() *layout.Element
	// Layout solves the layout for a viewport of the given logical size.
	Layout(w, h float32)
	// Update advances one frame of state using this frame's input.
	Update(m *input.Mouse, kb *input.Keyboard)
	// Close releases any resources the scene owns (worker goroutines, etc.).
	Close()
}

// Animator is an OPTIONAL capability a Scene may implement to control idle CPU.
// The runtime's event loop sleeps between frames; if a Scene implements this,
// AnimationWait tells the runtime how long it may sleep before the next
// time-based repaint is due (e.g. a caret blink toggle or an in-flight syntax
// highlight result), and whether such an animation is currently active.
//
//   - ok == false: no time-based work pending; the runtime blocks until the next
//     input/resize event (zero idle CPU).
//   - ok == true:  the runtime waits at most d for an event, then repaints; d is
//     the time until the next animation tick (small while animating fast, larger
//     while merely blinking a caret).
//
// It is discovered via a runtime type assertion, so the Scene interface itself
// stays minimal and yoga remains decoupled from the components/theme packages.
type Animator interface {
	AnimationWait() (d time.Duration, ok bool)
}
