package ui

import (
	"testing"
	"time"

	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/theme"
)

// overlayScrollFrame builds a tall Scroll ("s") with the given mode and returns
// the root and its ScrollView.
func overlayScrollFrame(c *Ctx, mode theme.ScrollbarMode, m *input.Mouse) (*layout.Element, *ScrollView) {
	root := BuildFrame(c, func(*Ctx) View {
		blocks := make([]View, 0, 20)
		for range 20 {
			blocks = append(blocks, Raw(layout.New(layout.Box().H(40).FlexShrink(0))))
		}
		return Scroll("s", Column(blocks...)).Scrollbars(mode).Grow(1)
	}, 400, 200, m, &input.Keyboard{})
	c.EndFrame()
	return root, c.Widget("s", nil).(*ScrollView)
}

func TestScrollOverlayReservesNoGutter(t *testing.T) {
	m := &input.Mouse{X: 500, Y: 500}
	c := New(nil, NewFocusScope(), nil)
	_, sv := overlayScrollFrame(c, ScrollOverlay, m)
	_, sv = overlayScrollFrame(c, ScrollOverlay, m)
	if w := sv.viewEl.Frame.W; w != 400 {
		t.Fatalf("overlay content width %v, want the full 400", w)
	}

	c = New(nil, NewFocusScope(), nil)
	_, sv = overlayScrollFrame(c, ScrollAlways, m)
	_, sv = overlayScrollFrame(c, ScrollAlways, m)
	want := 400 - theme.Current().Metrics.ScrollbarSize
	if w := sv.viewEl.Frame.W; w != want {
		t.Fatalf("classic content width %v, want %v", w, want)
	}
	if sv.vbar.host.Frame.W != theme.Current().Metrics.ScrollbarSize {
		t.Fatalf("classic bar width %v changed", sv.vbar.host.Frame.W)
	}
}

func TestScrollOverlayFades(t *testing.T) {
	m := &input.Mouse{X: 500, Y: 500} // outside the scroll area
	c := New(nil, NewFocusScope(), nil)
	overlayScrollFrame(c, ScrollOverlay, m)
	_, sv := overlayScrollFrame(c, ScrollOverlay, m)
	if sv.vbar.alpha != 0 {
		t.Fatalf("thumb visible at rest: alpha %v", sv.vbar.alpha)
	}
	if sv.vbar.hit(395, 50) {
		t.Fatal("a hidden overlay bar must let clicks through")
	}

	// Wheel over the content shows it.
	m.X, m.Y = 100, 100
	root, _ := overlayScrollFrame(c, ScrollOverlay, m)
	m.ScrollY = -1
	layout.Dispatch(root, m)
	m.ScrollY = 0
	_, sv = overlayScrollFrame(c, ScrollOverlay, m)
	if sv.vbar.alpha != 1 {
		t.Fatalf("thumb hidden after scrolling: alpha %v", sv.vbar.alpha)
	}
	if _, ok := c.AnimationWait(); !ok {
		t.Fatal("a shown thumb must schedule its fade")
	}

	// Mid-fade, then gone once idle.
	sv.vbar.activeAt = time.Now().Add(-overlayHold - overlayFade/2)
	overlayScrollFrame(c, ScrollOverlay, m)
	if a := sv.vbar.alpha; a <= 0 || a >= 1 {
		t.Fatalf("alpha %v mid-fade, want between 0 and 1", a)
	}
	sv.vbar.activeAt = time.Now().Add(-2 * time.Second)
	overlayScrollFrame(c, ScrollOverlay, m)
	if sv.vbar.alpha != 0 {
		t.Fatalf("thumb still visible after idle: alpha %v", sv.vbar.alpha)
	}
	if _, ok := c.AnimationWait(); ok {
		t.Fatal("a hidden thumb must not keep the loop awake")
	}

	// Moving the pointer over the area shows it again.
	m.X = 120
	overlayScrollFrame(c, ScrollOverlay, m)
	if sv.vbar.alpha != 1 {
		t.Fatalf("pointer move did not show the thumb: alpha %v", sv.vbar.alpha)
	}
}

func TestScrollOverlayDrag(t *testing.T) {
	m := &input.Mouse{X: 100, Y: 100}
	c := New(nil, NewFocusScope(), nil)
	overlayScrollFrame(c, ScrollOverlay, m)
	m.X = 110 // move over the area to show the thumb
	root, sv := overlayScrollFrame(c, ScrollOverlay, m)
	if sv.vbar.alpha != 1 {
		t.Fatalf("setup: thumb hidden, alpha %v", sv.vbar.alpha)
	}
	thumb := sv.vbar.thumb()
	m.X, m.Y = thumb.X+thumb.W-4, thumb.Y+thumb.H/2
	m.Down, m.Pressed = true, true
	layout.Dispatch(root, m)
	if !sv.vbar.dragging {
		t.Fatal("press on the overlay thumb did not start a drag")
	}
	if !sv.vbar.hovered {
		t.Fatal("thumb should widen while grabbed")
	}
	m.Pressed = false
	m.Y += 50
	root, sv = overlayScrollFrame(c, ScrollOverlay, m)
	layout.Dispatch(root, m)
	if sv.scrollY <= 0 {
		t.Fatalf("drag did not scroll: offset %v", sv.scrollY)
	}
}

func TestScrollOverlayPaintsThinRoundedThumb(t *testing.T) {
	m := &input.Mouse{X: 100, Y: 100}
	c := New(nil, NewFocusScope(), nil)
	overlayScrollFrame(c, ScrollOverlay, m)
	m.X = 110
	_, sv := overlayScrollFrame(c, ScrollOverlay, m)
	var dl render.DrawList
	sv.vbar.paint(&dl, nil)
	if len(dl.Vertices) == 0 {
		t.Fatal("visible overlay thumb painted nothing")
	}
	minX := float32(1e9)
	for _, v := range dl.Vertices {
		minX = min(minX, v.Pos[0])
	}
	f := sv.host.Frame
	if got, want := f.X+f.W-minX, overlaySize()+overlayPad; got != want {
		t.Fatalf("thumb spans %v px from the edge, want %v", got, want)
	}
}

func TestTreeOverlayScrollbarsTakeNoWidth(t *testing.T) {
	setupTreeTest(t)
	root := &TreeNode{Label: "root"}
	for range 100 {
		root.Children = append(root.Children, &TreeNode{Label: "leaf", Leaf: true})
	}
	tr := NewTree(root)
	tr.host.Frame = render.Rect{W: 300, H: 200}
	cw, _, vShow, _ := tr.scrollMetrics()
	if !vShow || cw != 300-theme.Current().Metrics.ScrollbarSize {
		t.Fatalf("classic: client width %v (vShow %v)", cw, vShow)
	}
	tr.SetScrollbars(ScrollOverlay)
	cw, _, vShow, _ = tr.scrollMetrics()
	if !vShow || cw != 300 {
		t.Fatalf("overlay: client width %v (vShow %v), want 300", cw, vShow)
	}
}
