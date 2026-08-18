package ui

import (
	"testing"

	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
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

func TestEnsureFocusWaitsUntilBuildFinishes(t *testing.T) {
	fs := NewFocusScope()
	c := New(nil, fs, nil)
	url, ed := newFake(), newFake()
	url.el.Frame = render.Rect{X: 0, Y: 0, W: 200, H: 24}
	ed.el.Frame = render.Rect{X: 0, Y: 40, W: 200, H: 120}

	body := func(c *Ctx) View {
		c.Focus().Add(url)
		c.Focus().EnsureFocus(url)
		c.Focus().Add(ed)
		return Raw(layout.New(layout.Box()))
	}

	BuildFrame(c, body, 200, 200, nil, nil)
	if fs.Current() != url {
		t.Fatal("empty scope should take DefaultFocus after build")
	}

	mouse := &input.Mouse{X: 10, Y: 50, Pressed: true}
	fs.HandleMouse(mouse)
	if fs.Current() != ed {
		t.Fatal("click should focus the editor")
	}

	BuildFrame(c, body, 200, 200, mouse, nil)
	if fs.Current() != ed {
		t.Fatalf("DefaultFocus stole the editor; current=%v", fs.Current())
	}
}

func TestEnsureFocusAppliesWhenPreviousWidgetGone(t *testing.T) {
	fs := NewFocusScope()
	c := New(nil, fs, nil)
	url, ed := newFake(), newFake()
	ed.el.Frame = render.Rect{X: 0, Y: 0, W: 100, H: 100}

	BuildFrame(c, func(c *Ctx) View {
		c.Focus().Add(url)
		c.Focus().EnsureFocus(url)
		c.Focus().Add(ed)
		return Raw(layout.New(layout.Box()))
	}, 200, 200, nil, nil)
	fs.HandleMouse(&input.Mouse{X: 10, Y: 10, Pressed: true})
	if fs.Current() != ed {
		t.Fatal("setup: editor should be focused")
	}

	BuildFrame(c, func(c *Ctx) View {
		c.Focus().Add(url)
		c.Focus().EnsureFocus(url)
		return Raw(layout.New(layout.Box()))
	}, 200, 200, nil, nil)
	if fs.Current() != url {
		t.Fatalf("want url after editor left the tree, got %v", fs.Current())
	}
}
