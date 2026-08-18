package ui

import (
	"github.com/mirzakhany/yoga/layout"
)

// Dropdown combines a trigger button with a Menu that opens beneath it.
type Dropdown struct {
	id    string
	label string
	Menu  *Menu
}

// NewDropdown builds a labelled trigger plus its overlay menu.
func NewDropdown(id, label string, width float32, items []MenuItem) *Dropdown {
	d := &Dropdown{id: id, label: label}
	d.Menu = NewMenu(width, items)
	return d
}

// Layout registers the trigger and, while open, the menu overlay.
func (d *Dropdown) Layout(c *Ctx) *layout.Element {
	el := Button(d.id, Text(d.label)).OnClick(func() {
		st := c.Widget(d.id, func() any { return &buttonState{} }).(*buttonState)
		if d.Menu.Open {
			d.Menu.Close()
			return
		}
		if st.el != nil {
			f := st.el.Frame
			d.Menu.OpenAt(f.X, f.Y+f.H)
		}
	}).Layout(c)
	if d.Menu.Open {
		c.Overlay(d.Menu.El)
	}
	return el
}
