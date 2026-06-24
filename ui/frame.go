package ui

import (
	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/layout"
)

// BuildFrame runs one build pass: it resets the context with the frame's
// viewport and input, builds the body, composes any registered overlays into a
// synthetic root, and solves layout. Both the GPU runtime and the headless
// driver use this so the two paths stay identical.
func BuildFrame(c *Ctx, body func(*Ctx) *layout.Element, w, h float32, m *input.Mouse, kb *input.Keyboard) *layout.Element {
	c.BeginFrame(w, h, m, kb)
	root := compose(body(c), c.overlays)
	root.Calculate(w, h)
	return root
}

// compose wraps body with an overlay layer when overlays are registered. With
// none (the common case) body is returned unchanged. Overlays are absolute and
// flagged Overlay by their components, so layout.Paint/Dispatch handle them
// after the body regardless of tree position.
func compose(body *layout.Element, overlays []*layout.Element) *layout.Element {
	if len(overlays) == 0 {
		return body
	}
	children := make([]*layout.Element, 0, len(overlays)+1)
	children = append(children, body)
	children = append(children, overlays...)
	return layout.New(layout.Box().FlexGrow(1), children...)
}
