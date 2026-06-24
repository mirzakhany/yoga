package ui

import "github.com/mirzakhany/yoga/layout"

// View is the universal widget interface. Every component implements
// Layout(c) and returns its *layout.Element directly, so call sites never touch
// a public .El field. The eager model means Layout produces a concrete element
// that fluent modifiers (.Gap, .Grow, ...) then chain on.
type View interface {
	Layout(c *Ctx) *layout.Element
}

// rawView adapts an already-built *layout.Element to View. A method cannot be
// added to layout.Element from this package (and layout must not import ui, to
// keep the dependency direction downward), so Raw wraps it instead.
type rawView struct{ e *layout.Element }

func (r rawView) Layout(*Ctx) *layout.Element { return r.e }

// Raw lets a bare *layout.Element be used wherever a View is expected (focus
// registries, overlay helpers, heterogeneous child lists).
func Raw(e *layout.Element) View { return rawView{e} }
