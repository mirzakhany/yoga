// Package theme defines the framework's color palette and a registry of
// prebuilt themes that can be switched at runtime.
//
// Runtime switching model: there is exactly one live *Theme instance
// (the "active" theme) returned by Current(). Every widget holds that pointer.
// Use(name) overwrites the active instance's fields in place, so a switch is
// reflected by all widgets on the very next paint with no rebuild.
//
// Layering: theme imports only render (for Color) and highlight (for the syntax
// ColorClass keys). The components package imports theme, never the reverse.
package theme

import (
	"sort"

	"github.com/mirzakhany/yoga/highlight"
	"github.com/mirzakhany/yoga/render"
)

// Theme is the standard semantic color palette shared by all widgets. Every
// builtin theme fills all tokens; custom themes should do the same.
type Theme struct {
	Name string // unique registry key, e.g. "dark"
	Dark bool   // true for dark palettes (lets widgets adapt if needed)

	Background render.Color // window / workspace background
	Panel      render.Color // sidebars, menus, bars
	PanelAlt   render.Color // secondary panel shade (tracks, gutters)
	Text       render.Color // primary foreground
	TextDim    render.Color // secondary / muted foreground
	Accent     render.Color // primary accent (selection, active marks)
	AccentText render.Color // foreground drawn on top of Accent
	Hover      render.Color // hovered row/control background
	Active     render.Color // active/pressed background
	Border     render.Color // separators, dividers
	Selection  render.Color // text selection highlight

	ScrollTrack      render.Color // scrollbar track background
	ScrollThumb      render.Color // scrollbar thumb (handle)
	ScrollThumbHover render.Color // thumb while hovered or dragging

	Error   render.Color
	Warning render.Color
	Success render.Color

	// Syntax maps highlight classes to colors for the code editor.
	Syntax map[highlight.ColorClass]render.Color
}

// SyntaxColor resolves a token class to a color, defaulting to plain text.
func (t *Theme) SyntaxColor(c highlight.ColorClass) render.Color {
	if col, ok := t.Syntax[c]; ok {
		return col
	}
	return t.Text
}

// registry holds all registered themes by name.
var registry = map[string]Theme{}

// active is the single live theme instance every widget reads from. Use()
// mutates it in place so switches are instant and pointer-stable.
var active = &Theme{}

// Current returns the live active theme. The pointer is stable for the lifetime
// of the process; its contents change when Use is called.
func Current() *Theme { return active }

// Register adds or replaces a theme in the registry (keyed by t.Name).
func Register(t Theme) { registry[t.Name] = t }

// Get returns a copy of the named theme.
func Get(name string) (Theme, bool) {
	t, ok := registry[name]
	return t, ok
}

// Names returns the sorted list of registered theme names.
func Names() []string {
	out := make([]string, 0, len(registry))
	for name := range registry {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

// Use switches the active theme to the named one, updating the shared instance
// in place. Returns false if the name is unknown (active is left unchanged).
func Use(name string) bool {
	t, ok := registry[name]
	if !ok {
		return false
	}
	*active = t
	return true
}

func init() {
	for _, t := range builtins() {
		Register(t)
	}
	Use("dark")
}
