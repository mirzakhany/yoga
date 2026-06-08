// Package layout is layer 1 of the framework: a declarative element tree driven
// by a pure-Go layout engine (flex, grid, stack) and a two-pass pipeline.
//
//	Pass 1 (Calculate): the engine solves constraints, assigning each node a
//	        position/size *relative to its parent's content box*.
//	Pass 2 (Flatten):   we walk the tree once, accumulating parent origins into
//	        absolute screen-space rectangles (Element.Frame) that the renderer
//	        and hit-testing consume.
package layout

import (
	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
)

// PaintFunc draws an element's visuals into the shared per-frame draw list.
type PaintFunc func(dl *render.DrawList, text *shape.Engine)

// MouseFunc receives pointer events for an element. Handlers should check
// e.Frame.Contains and may set m.Consumed to stop propagation to elements
// behind them.
type MouseFunc func(e *Element, m *input.Mouse)

// Element is a node in the UI tree. It couples layout style, an optional paint
// hook, and an optional input hook. Components (layer 3) build Elements and
// attach those hooks.
type Element struct {
	Style    Style
	Children []*Element

	// Frame is the absolute screen-space rectangle filled in by Flatten (pass 2).
	Frame render.Rect

	Paint   PaintFunc
	OnMouse MouseFunc

	// Overlay elements (dropdowns, context menus) are painted and hit-tested
	// after — i.e. on top of — the normal tree, regardless of their position in
	// it. This implements simple Z-axis ordering.
	Overlay bool

	// ScrollOffset shifts this element's children upward during Flatten,
	// implementing content scrolling without re-running the layout solver.
	ScrollOffset float32

	// Clip marks a viewport whose children should be clipped to its Frame. The
	// flag is advisory metadata the renderer/components can honor (the editor
	// uses it to drive virtualization).
	Clip bool

	// Computed layout (relative to parent content box); filled by the engine.
	cx, cy, cw, ch float32
}

// New creates an element with the given style and children. This is the
// declarative constructor: trees are written as nested calls, SwiftUI/Flutter
// style.
//
//	root := layout.New(layout.Box().Direction(layout.Row),
//	    sidebar,
//	    layout.New(layout.Box().FlexGrow(1), editor),
//	)
func New(style Style, children ...*Element) *Element {
	return &Element{Style: style, Children: children}
}

// WithPaint attaches a paint hook and returns the element for chaining.
func (e *Element) WithPaint(fn PaintFunc) *Element { e.Paint = fn; return e }

// WithMouse attaches an input hook and returns the element for chaining.
func (e *Element) WithMouse(fn MouseFunc) *Element { e.OnMouse = fn; return e }

// WithBackground attaches a paint hook that fills the element's frame with c.
func (e *Element) WithBackground(c render.Color) *Element {
	e.Paint = func(dl *render.DrawList, _ *shape.Engine) {
		dl.AddRect(e.Frame, c)
	}
	return e
}

// WithBackgroundPtr fills the element's frame with the color pointed to by c,
// read fresh every frame. This lets a background track a live theme color (the
// pointer stays valid while the theme's contents change on a runtime switch).
func (e *Element) WithBackgroundPtr(c *render.Color) *Element {
	e.Paint = func(dl *render.DrawList, _ *shape.Engine) {
		dl.AddRect(e.Frame, *c)
	}
	return e
}

// Add appends children and returns the element for chaining.
func (e *Element) Add(children ...*Element) *Element {
	e.Children = append(e.Children, children...)
	return e
}

// Engine abstracts the layout solver so the backend is swappable.
type Engine interface {
	// Compute runs both layout passes for the tree rooted at root inside the
	// (width, height) box, leaving absolute rectangles in each Element.Frame.
	Compute(root *Element, width, height float32)
}

// DefaultEngine is the active layout engine.
var DefaultEngine Engine = customEngine{}

// Calculate runs the two-pass pipeline using the default engine.
func (e *Element) Calculate(width, height float32) {
	DefaultEngine.Compute(e, width, height)
}

// MarkDirty clears computed geometry so the next Calculate relayouts the subtree.
// Call after structural changes such as opening a menu or adding children.
func (e *Element) MarkDirty() {
	e.cx, e.cy, e.cw, e.ch = 0, 0, 0, 0
	for _, c := range e.Children {
		c.MarkDirty()
	}
}

// ReapplyStyle is a no-op for the custom engine (styles are read fresh each
// Calculate). Kept for API compatibility with components that call it.
func (e *Element) ReapplyStyle() {}

func flatten(e *Element, originX, originY float32) {
	x := originX + e.cx
	y := originY + e.cy
	e.Frame = render.Rect{
		X: x, Y: y,
		W: e.cw,
		H: e.ch,
	}
	contentX := x + e.Style.Padding.Left
	contentY := y + e.Style.Padding.Top - e.ScrollOffset
	for _, c := range e.Children {
		flatten(c, contentX, contentY)
	}
}

// Paint walks the tree and emits geometry in painter's-algorithm order: the
// normal tree first, then overlay subtrees on top.
func Paint(root *Element, dl *render.DrawList, text *shape.Engine) {
	paintBase(root, dl, text)
	forEachOverlayRoot(root, func(o *Element) {
		paintAll(o, dl, text)
	})
}

func paintBase(e *Element, dl *render.DrawList, text *shape.Engine) {
	if e.Overlay {
		return
	}
	if e.Paint != nil {
		e.Paint(dl, text)
	}
	for _, c := range e.Children {
		paintBase(c, dl, text)
	}
}

func paintAll(e *Element, dl *render.DrawList, text *shape.Engine) {
	if e.Paint != nil {
		e.Paint(dl, text)
	}
	for _, c := range e.Children {
		paintAll(c, dl, text)
	}
}

func forEachOverlayRoot(e *Element, fn func(*Element)) {
	if e.Overlay {
		fn(e)
		return // treat the whole subtree as part of this overlay
	}
	for _, c := range e.Children {
		forEachOverlayRoot(c, fn)
	}
}

// Dispatch delivers a mouse event to the tree, front-to-back: overlay subtrees
// receive it before the base tree so a click on an open menu is not also seen
// by the widgets beneath it. Handlers may set m.Consumed to halt propagation.
func Dispatch(root *Element, m *input.Mouse) {
	if m == nil || m.Consumed {
		return
	}
	stop := false
	forEachOverlayRoot(root, func(o *Element) {
		if stop {
			return
		}
		dispatchTree(o, m, &stop)
	})
	if !stop {
		dispatchBase(root, m, &stop)
	}
}

func dispatchTree(e *Element, m *input.Mouse, stop *bool) {
	// Deepest/last children are visually on top, so visit them first.
	for i := len(e.Children) - 1; i >= 0; i-- {
		dispatchTree(e.Children[i], m, stop)
		if *stop {
			return
		}
	}
	if e.OnMouse != nil {
		e.OnMouse(e, m)
		if m.Consumed {
			*stop = true
		}
	}
}

func dispatchBase(e *Element, m *input.Mouse, stop *bool) {
	if e.Overlay {
		return
	}
	for i := len(e.Children) - 1; i >= 0; i-- {
		dispatchBase(e.Children[i], m, stop)
		if *stop {
			return
		}
	}
	if e.OnMouse != nil {
		e.OnMouse(e, m)
		if m.Consumed {
			*stop = true
		}
	}
}
