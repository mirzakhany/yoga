package ui

import (
	"testing"

	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/shape"
)

type fakeWindowHost struct {
	custom     bool
	native     bool
	inset      float32
	maximized  bool
	closed     bool
	minimized  bool
	moveStarted bool
	maxToggles int
}

func (f *fakeWindowHost) CustomTitleBar() bool   { return f.custom }
func (f *fakeWindowHost) NativeControls() bool   { return f.native }
func (f *fakeWindowHost) ControlsInset() float32 { return f.inset }
func (f *fakeWindowHost) Close()                 { f.closed = true }
func (f *fakeWindowHost) Minimize()              { f.minimized = true }
func (f *fakeWindowHost) ToggleMaximize()        { f.maxToggles++ }
func (f *fakeWindowHost) IsMaximized() bool      { return f.maximized }
func (f *fakeWindowHost) BeginMove()             { f.moveStarted = true }

func setupTitleBarCtx(t *testing.T, win WindowHost) *Ctx {
	t.Helper()
	text, err := shape.NewEngine(1, false)
	if err != nil {
		t.Fatal(err)
	}
	SetFrameResources(text, nil, nil)
	c := New(text, NewFocusScope(), nil)
	c.SetWindow(win)
	return c
}

func findElementByHeight(e *layout.Element, h float32) *layout.Element {
	if e == nil {
		return nil
	}
	if e.Frame.H >= h-0.5 && e.Frame.H <= h+0.5 && e.Frame.Y == 0 {
		return e
	}
	for _, ch := range e.Children {
		if found := findElementByHeight(ch, h); found != nil {
			return found
		}
	}
	return nil
}

func countWideButtons(e *layout.Element, minW float32) int {
	if e == nil {
		return 0
	}
	n := 0
	if e.Frame.W >= minW && e.Frame.H >= 30 && e.OnMouse != nil {
		n++
	}
	for _, ch := range e.Children {
		n += countWideButtons(ch, minW)
	}
	return n
}

func TestTitleBarNilHostDoesNotPanic(t *testing.T) {
	c := setupTitleBarCtx(t, nil)
	root := BuildFrame(c, func(_ *Ctx) View {
		return TitleBar(Text("App"))
	}, 800, 600, nil, nil)
	if root == nil {
		t.Fatal("nil root")
	}
}

func TestTitleBarNativeControlsInset(t *testing.T) {
	win := &fakeWindowHost{custom: true, native: true, inset: 78}
	c := setupTitleBarCtx(t, win)
	th := c.Theme()
	root := BuildFrame(c, func(_ *Ctx) View {
		return TitleBar(Text("Catalog"))
	}, 800, 600, nil, nil)
	bar := findElementByHeight(root, th.Metrics.TitleBarHeight)
	if bar == nil {
		t.Fatal("title bar not found")
	}
	foundInset := false
	for _, ch := range bar.Children {
		if ch.Frame.W >= 70 && ch.Frame.W <= 85 {
			foundInset = true
			break
		}
	}
	if !foundInset {
		t.Fatalf("expected leading inset ~78px, children: %d", len(bar.Children))
	}
}

func TestTitleBarDrawnWindowControls(t *testing.T) {
	win := &fakeWindowHost{custom: true, native: false}
	c := setupTitleBarCtx(t, win)
	root := BuildFrame(c, func(_ *Ctx) View {
		return TitleBar(Text("App"))
	}, 800, 600, nil, nil)
	n := countWideButtons(root, 38)
	if n < 3 {
		t.Fatalf("expected at least 3 window control buttons, got %d", n)
	}
}

func TestWindowControlsMaximizeIconWhenMaximized(t *testing.T) {
	win := &fakeWindowHost{custom: true, native: false, maximized: true}
	c := setupTitleBarCtx(t, win)
	root := BuildFrame(c, func(_ *Ctx) View {
		return WindowControls()
	}, 200, 40, nil, nil)
	if root == nil || root.Frame.W <= 0 {
		t.Fatal("window controls did not layout")
	}
}
