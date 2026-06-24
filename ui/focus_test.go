package ui

import (
	"testing"

	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/layout"
)

// fakeFocusable is a minimal Focusable for traversal/routing tests.
type fakeFocusable struct {
	el          *layout.Element
	focused     bool
	capturesTab bool
	keys        []input.KeyEvent
	chars       []rune
}

func newFake() *fakeFocusable { return &fakeFocusable{el: layout.New(layout.Box())} }

func (f *fakeFocusable) Focus()                        { f.focused = true }
func (f *fakeFocusable) Blur()                         { f.focused = false }
func (f *fakeFocusable) Focused() bool                 { return f.focused }
func (f *fakeFocusable) HandleText(r []rune)           { f.chars = append(f.chars, r...) }
func (f *fakeFocusable) HandleKeys(k []input.KeyEvent) { f.keys = append(f.keys, k...) }
func (f *fakeFocusable) CapturesTab() bool             { return f.capturesTab }
func (f *fakeFocusable) FocusOnClick() bool            { return true }
func (f *fakeFocusable) FocusEl() *layout.Element      { return f.el }

func TestFocusTraversalPreservedAcrossFrames(t *testing.T) {
	fs := NewFocusScope()
	a, b, c := newFake(), newFake(), newFake()

	// Frame 1: register and step forward twice.
	fs.beginFrame()
	fs.Add(a, b, c)
	fs.Next() // -> a
	fs.Next() // -> b
	if fs.Current() != b || !b.focused {
		t.Fatalf("want b focused after two Next")
	}

	// Frame 2: re-register (rebuild); focus identity must survive.
	fs.beginFrame()
	fs.Add(a, b, c)
	if fs.Current() != b {
		t.Fatalf("focus lost across rebuild: got %v", fs.Current())
	}
	fs.Next() // -> c
	if fs.Current() != c {
		t.Fatalf("want c after Next, got %v", fs.Current())
	}
}

func TestRouteTabMovesFocusUnlessCaptured(t *testing.T) {
	fs := NewFocusScope()
	a, b := newFake(), newFake()
	fs.beginFrame()
	fs.Add(a, b)
	fs.Next() // focus a

	// a does not capture Tab: Tab advances to b.
	kb := &input.Keyboard{Keys: []input.KeyEvent{{Key: input.KeyTab}}}
	fs.Route(kb)
	if fs.Current() != b {
		t.Fatalf("plain Tab should advance focus to b, got %v", fs.Current())
	}

	// b captures Tab: Tab is delivered, not consumed for traversal.
	b.capturesTab = true
	kb = &input.Keyboard{Keys: []input.KeyEvent{{Key: input.KeyTab}}}
	fs.Route(kb)
	if fs.Current() != b {
		t.Fatalf("captured Tab must not move focus, got %v", fs.Current())
	}
	if len(b.keys) != 1 || b.keys[0].Key != input.KeyTab {
		t.Fatalf("captured Tab should reach the widget, got %v", b.keys)
	}
}
