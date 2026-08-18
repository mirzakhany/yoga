package ui

import "github.com/mirzakhany/yoga/layout"

// ----------------------------------------------------------------------------
// List: a thin helper that arranges children as a row or column stack using
// Yoga's flex flow. It is intentionally just a styled container — the power is
// in the layout engine.
// ----------------------------------------------------------------------------

// NewList returns a container that stacks its children along dir.
func NewList(dir layout.FlexDirection, items ...*layout.Element) *layout.Element {
	return layout.New(layout.Box().Direction(dir), items...)
}
