package ui

import (
	"runtime"
	"testing"

	"github.com/mirzakhany/yoga/input"
)

func TestParseKeySeq(t *testing.T) {
	s, err := ParseKeySeq("Mod+K Mod+S")
	if err != nil || s.First.Key != input.KeyK || s.Then.Key != input.KeyS {
		t.Fatalf("got %+v %v", s, err)
	}
	if s, err := ParseKeySeq("Ctrl + Shift + P"); err != nil || s.Then.Valid() || s.First.Key != input.KeyP {
		t.Fatalf("spaces inside a chord: %+v %v", s, err)
	}
	if s, err := ParseKeySeq("alt+mouse4"); err != nil || s.First.Key != input.KeyMouseBack || !s.First.Mods.Has(input.ModAlt) {
		t.Fatalf("mouse chord: %+v %v", s, err)
	}
	if _, err := ParseKeySeq("a b c"); err == nil {
		t.Fatal("three chords should fail")
	}
	if got := MustParseChord("mouse5").Label(); got != "Mouse Forward" {
		t.Fatalf("label %q", got)
	}
}

func dispatchKeys(c *Ctx, evs ...input.KeyEvent) {
	for _, ev := range evs {
		c.Commands().Dispatch(&input.Keyboard{Keys: []input.KeyEvent{ev}})
	}
}

func TestTwoChordShortcutWaitsForSecondKey(t *testing.T) {
	c := New(nil, NewFocusScope(), nil)
	c.BeginFrame(800, 600, nil, nil)
	var ran []string
	c.Commands().Register(
		Cmd("save.all").Shortcut("Mod+K S").Run(func() { ran = append(ran, "save.all") }),
		Cmd("k").Shortcut("Mod+K").Run(func() { ran = append(ran, "k") }),
		Cmd("back").Shortcut("Mouse4").Run(func() { ran = append(ran, "back") }),
	)
	c.Commands().Layout(c)
	modK := input.KeyEvent{Key: input.KeyK, Mods: input.ModSuper}
	dispatchKeys(c, modK)
	if len(ran) != 0 || c.Commands().Pending() != "⌘K" {
		t.Fatalf("after ⌘K: ran %v, pending %q", ran, c.Commands().Pending())
	}
	dispatchKeys(c, input.KeyEvent{Key: input.KeyS})
	if len(ran) != 1 || ran[0] != "save.all" || c.Commands().Pending() != "" {
		t.Fatalf("after ⌘K S: ran %v", ran)
	}
	// A second key that completes nothing is swallowed.
	kb := &input.Keyboard{Keys: []input.KeyEvent{modK, {Key: input.KeyX}}}
	c.Commands().Dispatch(kb)
	if len(kb.Keys) != 0 || len(ran) != 1 {
		t.Fatalf("keys left %v, ran %v", kb.Keys, ran)
	}
	dispatchKeys(c, input.KeyEvent{Key: input.KeyMouseBack})
	if ran[len(ran)-1] != "back" {
		t.Fatalf("mouse button shortcut did not run: %v", ran)
	}
}

func TestLiteralCtrlOnMac(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("only macOS tells Ctrl and Cmd apart")
	}
	LiteralCtrl = true
	defer func() { LiteralCtrl = false }()
	ctrl := MustParseChord("Ctrl+`")
	if !ctrl.Matches(input.KeyEvent{Key: input.KeyBacktick, Mods: input.ModCtrl}) ||
		ctrl.Matches(input.KeyEvent{Key: input.KeyBacktick, Mods: input.ModSuper}) {
		t.Fatal("Ctrl+` should match Control only")
	}
	cmd := MustParseChord("Cmd+S")
	if !cmd.Matches(input.KeyEvent{Key: input.KeyS, Mods: input.ModSuper}) ||
		cmd.Matches(input.KeyEvent{Key: input.KeyS, Mods: input.ModCtrl}) {
		t.Fatal("Cmd+S should match Command only")
	}
	if !MustParseChord("Mod+S").Matches(input.KeyEvent{Key: input.KeyS, Mods: input.ModCtrl}) {
		t.Fatal("Mod stays primary")
	}
	if got := ctrl.Label(); got != "⌃`" {
		t.Fatalf("label %q", got)
	}
}

func TestCommandsRunByID(t *testing.T) {
	c := New(nil, NewFocusScope(), nil)
	c.BeginFrame(800, 600, nil, nil)
	ran := 0
	c.Commands().Register(Cmd("a").Run(func() { ran++ }), Cmd("b").Enabled(false).Run(func() { ran += 10 }))
	if !c.Commands().Run("a") || c.Commands().Run("b") || c.Commands().Run("nope") || ran != 1 {
		t.Fatalf("ran %d", ran)
	}
}
