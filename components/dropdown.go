package components

import (
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/ui"
)

// Dropdown combines a trigger Button with a Menu that opens beneath it.
type Dropdown struct {
	Button *Button
	Menu   *Menu
}

// Layout is the new ui.View entry point: it registers the trigger button with
// the frame's focus scope and, while open, self-registers the menu as an
// overlay — no manual Menu.El mounting on the root.
func (d *Dropdown) Layout(c *ui.Ctx) *layout.Element {
	c.Focus().Add(d.Button)
	if d.Menu.Open {
		c.Overlay(d.Menu.El)
	}
	return d.Button.El
}

// NewDropdown builds a labelled trigger button plus its overlay menu. Add
// Dropdown.Button.El into the layout where the trigger should appear, and add
// Dropdown.Menu.El to the tree root.
func NewDropdown(label string, width float32, items []MenuItem) *Dropdown {
	d := &Dropdown{}
	d.Menu = NewMenu(width, items)
	d.Button = NewButton(label).Action(func() {
		if d.Menu.Open {
			d.Menu.Close()
			return
		}
		f := d.Button.El.Frame
		d.Menu.OpenAt(f.X, f.Y+f.H)
	})
	return d
}
