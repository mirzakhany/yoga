package ui

// Command is a registered action with optional shortcut and palette metadata.
type Command struct {
	id       string
	title    string
	group    string
	icon     string
	shortcut Chord
	enabled  bool
	hidden   bool
	run      func()
}

// Cmd starts a fluent command builder. Commands are enabled by default.
func Cmd(id string) *Command {
	return &Command{id: id, enabled: true}
}

// Title sets the palette display name.
func (c *Command) Title(s string) *Command { c.title = s; return c }

// Group sets an optional grouping label shown muted under the title.
func (c *Command) Group(s string) *Command { c.group = s; return c }

// Icon sets an optional leading icon name.
func (c *Command) Icon(name string) *Command { c.icon = name; return c }

// Shortcut parses and attaches a chord (e.g. "⌘S", "Mod+K"). Invalid strings are ignored.
func (c *Command) Shortcut(s string) *Command {
	if ch, err := ParseChord(s); err == nil {
		c.shortcut = ch
	}
	return c
}

// ShortcutChord attaches a pre-parsed chord.
func (c *Command) ShortcutChord(ch Chord) *Command {
	c.shortcut = ch
	return c
}

// Enable sets whether the command can run via shortcut or Enter.
func (c *Command) Enable(v bool) *Command { c.enabled = v; return c }

// Enabled is an alias for Enable.
func (c *Command) Enabled(v bool) *Command { return c.Enable(v) }

// Hide omits the command from the palette while keeping its shortcut active.
func (c *Command) Hide(v bool) *Command { c.hidden = v; return c }

// Hidden is an alias for Hide.
func (c *Command) Hidden(v bool) *Command { return c.Hide(v) }

// Run sets the action invoked when the command is selected or its shortcut fires.
func (c *Command) Run(fn func()) *Command { c.run = fn; return c }

// ID returns the command id.
func (c *Command) ID() string { return c.id }

// DisplayTitle returns the palette title (falls back to id).
func (c *Command) DisplayTitle() string {
	if c.title != "" {
		return c.title
	}
	return c.id
}
