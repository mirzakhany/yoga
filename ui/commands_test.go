package ui

import (
	"testing"

	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/theme"
)

func TestParseChordSymbols(t *testing.T) {
	c, err := ParseChord("⌘S")
	if err != nil {
		t.Fatal(err)
	}
	if c.Key != input.KeyS || !c.primary {
		t.Fatalf("got key=%v primary=%v", c.Key, c.primary)
	}
	if got := c.Label(); got != "⌘S" {
		t.Fatalf("Label = %q, want ⌘S", got)
	}
}

func TestParseChordModPlus(t *testing.T) {
	c, err := ParseChord("Mod+K")
	if err != nil {
		t.Fatal(err)
	}
	if c.Key != input.KeyK || !c.primary {
		t.Fatalf("got key=%v primary=%v", c.Key, c.primary)
	}
	c2, err := ParseChord("Ctrl+Shift+P")
	if err != nil {
		t.Fatal(err)
	}
	if c2.Key != input.KeyP || !c2.primary || !c2.Mods.Has(input.ModShift) {
		t.Fatalf("got %+v", c2)
	}
}

func TestChordMatchesPrimary(t *testing.T) {
	c := MustParseChord("⌘S")
	if !c.Matches(input.KeyEvent{Key: input.KeyS, Mods: input.ModSuper}) {
		t.Fatal("should match Super+S")
	}
	if !c.Matches(input.KeyEvent{Key: input.KeyS, Mods: input.ModCtrl}) {
		t.Fatal("should match Ctrl+S")
	}
	if c.Matches(input.KeyEvent{Key: input.KeyS, Mods: 0}) {
		t.Fatal("should not match bare S")
	}
	if c.Matches(input.KeyEvent{Key: input.KeyS, Mods: input.ModSuper | input.ModShift}) {
		t.Fatal("should not match Super+Shift+S")
	}
}

func TestFilterCommandsSubsequence(t *testing.T) {
	cmds := []*Command{
		Cmd("file.save").Title("Save File"),
		Cmd("file.open").Title("Open File"),
		Cmd("view.theme").Title("Toggle Theme").Group("View"),
	}
	all := filterCommands(cmds, "")
	if len(all) != 3 {
		t.Fatalf("empty query: got %d", len(all))
	}
	got := filterCommands(cmds, "sf")
	if len(got) != 1 || got[0].id != "file.save" {
		t.Fatalf("sf match = %v", idsOf(got))
	}
	got = filterCommands(cmds, "theme")
	if len(got) != 1 || got[0].id != "view.theme" {
		t.Fatalf("theme match = %v", idsOf(got))
	}
	// Prefix ranks above scattered.
	cmds2 := []*Command{
		Cmd("a").Title("save elsewhere"),
		Cmd("b").Title("Save"),
	}
	got = filterCommands(cmds2, "save")
	if len(got) < 2 || got[0].id != "b" {
		t.Fatalf("prefix should win, got %v", idsOf(got))
	}
}

func idsOf(cmds []*Command) []string {
	out := make([]string, len(cmds))
	for i, c := range cmds {
		out[i] = c.id
	}
	return out
}

func TestCommandsDispatchRunsAndConsumes(t *testing.T) {
	c := New(nil, NewFocusScope(), nil)
	c.BeginFrame(800, 600, nil, nil)
	ran := false
	c.Commands().Register(Cmd("file.save").Title("Save").Shortcut("⌘S").Run(func() { ran = true }))
	c.Commands().Layout(c) // bind ctx for modal checks

	kb := &input.Keyboard{Keys: []input.KeyEvent{{Key: input.KeyS, Mods: input.ModSuper}}}
	c.Commands().Dispatch(kb)
	if !ran {
		t.Fatal("expected Run")
	}
	if len(kb.Keys) != 0 {
		t.Fatalf("expected consumed, leftover %v", kb.Keys)
	}
}

func TestCommandsDispatchSkippedWhenDialogOpen(t *testing.T) {
	c := New(nil, NewFocusScope(), nil)
	c.BeginFrame(800, 600, nil, nil)
	ran := false
	c.Commands().Register(Cmd("file.save").Title("Save").Shortcut("⌘S").Run(func() { ran = true }))
	c.Dialogs().ShowInfo("x", "y", nil)
	c.layoutWindowOverlays()

	kb := &input.Keyboard{Keys: []input.KeyEvent{{Key: input.KeyS, Mods: input.ModSuper}}}
	c.Commands().Dispatch(kb)
	if ran {
		t.Fatal("should not run while dialog open")
	}
	if len(kb.Keys) != 1 {
		t.Fatalf("should leave keys, got %d", len(kb.Keys))
	}
}

func TestCommandsDispatchSkipsWhilePaletteOpen(t *testing.T) {
	c := New(nil, NewFocusScope(), nil)
	c.BeginFrame(800, 600, nil, nil)
	ran := false
	c.Commands().Register(Cmd("file.save").Title("Save").Shortcut("⌘S").Run(func() { ran = true }))
	c.Commands().Show()
	c.Commands().Layout(c)

	kb := &input.Keyboard{Keys: []input.KeyEvent{{Key: input.KeyS, Mods: input.ModSuper}}}
	c.Commands().Dispatch(kb)
	if ran {
		t.Fatal("save should not fire while palette open")
	}
	if len(kb.Keys) != 1 {
		t.Fatal("save chord should remain for focus routing")
	}
}

func TestCommandsToggleChord(t *testing.T) {
	c := New(nil, NewFocusScope(), nil)
	c.BeginFrame(800, 600, nil, nil)
	h := c.Commands()
	h.Layout(c)
	kb := &input.Keyboard{Keys: []input.KeyEvent{{Key: input.KeyK, Mods: input.ModSuper}}}
	h.Dispatch(kb)
	if !h.Open {
		t.Fatal("Mod+K should open palette")
	}
	if len(kb.Keys) != 0 {
		t.Fatal("toggle should consume")
	}
	kb.Keys = []input.KeyEvent{{Key: input.KeyK, Mods: input.ModCtrl}}
	h.Dispatch(kb)
	if h.Open {
		t.Fatal("second toggle should close")
	}
}

func TestCommandsPaletteFilterKeys(t *testing.T) {
	c := New(nil, NewFocusScope(), nil)
	c.BeginFrame(800, 600, nil, nil)
	ran := ""
	h := c.Commands()
	h.Register(
		Cmd("a").Title("Alpha").Run(func() { ran = "a" }),
		Cmd("b").Title("Beta").Run(func() { ran = "b" }),
	)
	h.Show()
	h.Layout(c)

	pass := h.FilterKeys([]input.KeyEvent{{Key: input.KeyDown}})
	if len(pass) != 0 {
		t.Fatal("Down should be consumed")
	}
	if h.cursor != 1 {
		t.Fatalf("cursor=%d want 1", h.cursor)
	}
	h.FilterKeys([]input.KeyEvent{{Key: input.KeyEnter}})
	if ran != "b" {
		t.Fatalf("ran=%q want b", ran)
	}
	if h.Open {
		t.Fatal("Enter should close palette")
	}
}

func TestCommandsEnsureCursorVisible(t *testing.T) {
	h := NewCommandsHost()
	sv := NewScrollView(nil)
	h.listSV = sv
	h.listView = 100
	rowH := commandRowHeight(theme.Current())
	// Enough items that cursor 5 is below the first viewport.
	n := int(h.listView/rowH) + 5
	for i := 0; i < n; i++ {
		h.filtered = append(h.filtered, Cmd("x").Title("Item"))
	}
	h.cursor = n - 1
	h.ensureCursorVisible()
	wantTop := float32(h.cursor)*rowH - h.listView
	if wantTop < 0 {
		wantTop = 0
	}
	if sv.scrollY < wantTop-0.5 {
		t.Fatalf("scrollY=%v want >= %v so last row is visible", sv.scrollY, wantTop)
	}
	bottom := float32(h.cursor)*rowH + rowH
	if bottom > sv.scrollY+h.listView+0.5 {
		t.Fatalf("selected bottom %v still below viewport end %v", bottom, sv.scrollY+h.listView)
	}
}

func TestCommandsPaletteEscape(t *testing.T) {
	h := NewCommandsHost()
	h.Show()
	h.HandleKeys([]input.KeyEvent{{Key: input.KeyEscape}})
	if h.Open {
		t.Fatal("Escape should close")
	}
}

func TestCommandsPaletteOverlays(t *testing.T) {
	c := New(nil, NewFocusScope(), nil)
	c.BeginFrame(800, 600, nil, nil)
	h := c.Commands()
	h.Register(Cmd("x").Title("X"))
	h.Show()
	h.Layout(c)
	if got := len(c.Overlays()); got != 2 {
		t.Fatalf("open palette overlays=%d want 2 (scrim+panel)", got)
	}
}

func TestCommandsHiddenAndDisabled(t *testing.T) {
	cmds := []*Command{
		Cmd("vis").Title("Visible"),
		Cmd("hid").Title("Hidden").Hidden(true),
		Cmd("off").Title("Off").Enabled(false),
	}
	got := filterCommands(cmds, "")
	if len(got) != 2 {
		t.Fatalf("hidden omitted: got %d", len(got))
	}
	h := NewCommandsHost()
	ran := false
	h.Register(Cmd("off").Title("Off").Shortcut("⌘O").Enabled(false).Run(func() { ran = true }))
	kb := &input.Keyboard{Keys: []input.KeyEvent{{Key: input.KeyO, Mods: input.ModSuper}}}
	h.Dispatch(kb)
	if ran {
		t.Fatal("disabled shortcut should not run")
	}
}
