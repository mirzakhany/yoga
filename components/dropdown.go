package components

// Dropdown combines a trigger Button with a Menu that opens beneath it.
type Dropdown struct {
	Button *Button
	Menu   *Menu
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
