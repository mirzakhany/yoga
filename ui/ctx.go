// Package ui is the ergonomic, SwiftUI-inspired surface of the framework. It
// sits above layout/components/theme and threads a single frame context (Ctx)
// through every build pass, absorbing the host wiring, invalidation, animation
// scheduling, and overlay mounting that apps previously did by hand.
//
// Phase 0: this package only defines Ctx and View. Nothing is wired into the
// runtime yet, so behavior is unchanged; the runtime takes over construction
// and per-frame plumbing in Phase 1.
package ui

import (
	"time"

	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/shape"
	"github.com/mirzakhany/yoga/theme"
)

// Ctx is the per-frame build context (the "gtx"). A single value is threaded
// through Body and every component's Layout(c) call. It is the one place apps
// and components reach for invalidation, animation scheduling, overlay
// registration, and frame-wide state (viewport, theme, text engine).
//
// A Ctx is owned by the runtime and reset at the start of each build pass; do
// not retain it across frames. Component *state* lives in the retained
// component struct, not here.
type Ctx struct {
	now      time.Time
	vw, vh   float32
	text     *shape.Engine
	theme    *theme.Theme
	overlays []*layout.Element
	wakeIn   time.Duration // smallest Animate() request this frame; <0 = none
	focus    *FocusScope   // runtime-owned, persists across frames
	mouse    *input.Mouse
	keyboard *input.Keyboard
	post     func() // thread-safe wake (glfw.PostEmptyEvent); may be nil
}

// New builds a Ctx bound to a text engine and an optional thread-safe wake.
// post breaks the event loop from a worker goroutine (glfw.PostEmptyEvent);
// pass nil for the headless/no-op case. The runtime constructs the Ctx once,
// owns the FocusScope, and calls BeginFrame before every build pass.
//
// The ui app path rebuilds the body every frame (the event-driven loop only
// wakes to draw on input or an Animate request), so there is no host caching
// here: Overlay/Animate registration is collected fresh each frame.
func New(text *shape.Engine, focus *FocusScope, post func()) *Ctx {
	return &Ctx{text: text, focus: focus, post: post, wakeIn: -1}
}

// BeginFrame resets per-frame state and records the frame clock, viewport, and
// this frame's input. The runtime calls this immediately before building the
// body.
func (c *Ctx) BeginFrame(vw, vh float32, m *input.Mouse, kb *input.Keyboard) {
	c.now = time.Now()
	c.vw, c.vh = vw, vh
	c.mouse, c.keyboard = m, kb
	c.theme = theme.Current()
	c.overlays = c.overlays[:0]
	c.wakeIn = -1
	if c.focus != nil {
		c.focus.beginFrame()
	}
}

// Mouse returns this frame's pointer state, so components can do per-frame work
// (drag continuation, caret advance) inside Layout(c). May be nil in tests.
func (c *Ctx) Mouse() *input.Mouse { return c.mouse }

// Keyboard returns this frame's keyboard state. May be nil in tests.
func (c *Ctx) Keyboard() *input.Keyboard { return c.keyboard }

// Invalidate requests a repaint. Safe to call from any goroutine: the body is
// rebuilt every frame, so on the main thread the next loop iteration already
// repaints; from a worker this posts an empty event to break the idle wait.
func (c *Ctx) Invalidate() {
	if c.post != nil {
		c.post()
	}
}

// Focus returns the runtime-owned focus scope. Components register themselves
// with c.Focus().Add(widget) during Layout(c); the runtime routes keyboard
// input and click-to-focus after the build.
func (c *Ctx) Focus() *FocusScope { return c.focus }

// Animate requests a repaint within d ("this component animates"). The runtime
// takes the minimum across all Animate calls this frame as its wait budget,
// replacing the old per-Scene AnimationWait. Non-positive d means "as soon as
// possible".
func (c *Ctx) Animate(d time.Duration) {
	if d < 0 {
		d = 0
	}
	if c.wakeIn < 0 || d < c.wakeIn {
		c.wakeIn = d
	}
}

// AnimationWait reports the aggregated animation budget for this frame:
// (duration, true) if any component called Animate, else (0, false) meaning the
// loop may block on input with zero idle CPU.
func (c *Ctx) AnimationWait() (time.Duration, bool) {
	if c.wakeIn < 0 {
		return 0, false
	}
	return c.wakeIn, true
}

// Overlay registers portal content (dropdown/menu/tooltip/modal) for this
// frame. The runtime composes registered overlays into a synthetic layer
// painted and hit-tested after the body. Components call this from Layout(c)
// only while their overlay is open; open state lives in the component struct.
func (c *Ctx) Overlay(e *layout.Element) {
	if e != nil {
		c.overlays = append(c.overlays, e)
	}
}

// Overlays returns the overlay elements registered this frame, for the runtime
// to compose the portal layer.
func (c *Ctx) Overlays() []*layout.Element { return c.overlays }

// Now is the single frame clock. All time-based animation (caret blink, toast
// expiry, spinner) should read this so a frame is internally consistent.
func (c *Ctx) Now() time.Time {
	if c.now.IsZero() {
		return time.Now()
	}
	return c.now
}

// Viewport returns the current drawable size in logical pixels.
func (c *Ctx) Viewport() (w, h float32) { return c.vw, c.vh }

// Theme returns the active theme for this frame.
func (c *Ctx) Theme() *theme.Theme {
	if c.theme == nil {
		return theme.Current()
	}
	return c.theme
}

// Text returns the shared shaping/measuring engine.
func (c *Ctx) Text() *shape.Engine { return c.text }
