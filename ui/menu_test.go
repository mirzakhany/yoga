package ui

import (
	"fmt"
	"testing"

	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
	"github.com/mirzakhany/yoga/theme"
)

func longMenu(n, checked int) *Menu {
	items := make([]MenuItem, n)
	for i := range items {
		items[i] = MenuItem{Label: fmt.Sprintf("item %d", i), Checked: i == checked}
	}
	return NewMenu(200, items)
}

func withViewport(t *testing.T, w, h float32) {
	t.Helper()
	pw, ph := viewportW, viewportH
	SetViewport(w, h)
	t.Cleanup(func() { SetViewport(pw, ph) })
}

func TestMenuTallerThanViewportScrolls(t *testing.T) {
	withViewport(t, 800, 400)
	mu := longMenu(60, -1)
	mu.OpenAt(100, 40)
	f := mu.host.Frame
	if f.Y != 40 {
		t.Fatalf("menu should open below the click with room for rows there, got y=%v", f.Y)
	}
	if f.Y+f.H > 400 {
		t.Fatalf("menu runs off the viewport: %v", f)
	}
	if mu.height() <= f.H {
		t.Fatal("expected a scrolling menu")
	}
	first := mu.itemAt(f.Y + 1)
	m := &input.Mouse{X: f.X + 10, Y: f.Y + 1, ScrollY: -2}
	mu.onMouse(mu.host, m)
	if mu.scrollY <= 0 || m.ScrollY != 0 {
		t.Fatalf("wheel should scroll the menu, scrollY=%v", mu.scrollY)
	}
	if got := mu.itemAt(f.Y + 1); got <= first {
		t.Fatalf("top row after scrolling: got %d, was %d", got, first)
	}
	mu.onMouse(mu.host, &input.Mouse{X: f.X + 10, Y: f.Y + 1, ScrollY: -1000})
	if want := mu.height() - f.H; mu.scrollY != want {
		t.Fatalf("scroll not clamped: got %v want %v", mu.scrollY, want)
	}
}

func TestMenuOpensScrolledToCheckedItem(t *testing.T) {
	withViewport(t, 800, 400)
	mu := longMenu(60, 50)
	mu.OpenAt(100, 40)
	f := mu.host.Frame
	if i := mu.itemAt(f.Y + f.H/2); i < 45 || i > 55 {
		t.Fatalf("checked item 50 should be near the middle, got %d", i)
	}
}

func TestMenuShortFitsWithoutScrolling(t *testing.T) {
	withViewport(t, 800, 400)
	mu := longMenu(3, 1)
	mu.OpenAt(100, 40)
	if mu.host.Frame.H != mu.height() || mu.scrollY != 0 {
		t.Fatalf("short menu should show every row unscrolled: h=%v scroll=%v", mu.host.Frame.H, mu.scrollY)
	}
}

// The hover fill used to paint over the menu's own left and right border.
func TestMenuHoverStaysInsideBorder(t *testing.T) {
	text, err := shape.NewEngine(1, false)
	if err != nil {
		t.Fatal(err)
	}
	SetFrameResources(text, nil, nil)
	withViewport(t, 800, 600)

	mu := longMenu(4, -1)
	mu.OpenAt(100, 40)
	mu.hover = 1

	dl := &render.DrawList{}
	mu.paint(dl, text)

	f := mu.host.Frame
	bw := float32(theme.Current().Stroke.Thin)
	want := render.Rect{X: f.X + bw, Y: f.Y + bw, W: f.W - 2*bw, H: f.H - 2*bw}
	for _, cmd := range dl.Commands {
		if cmd.Clip.W < 0 {
			continue // border and shadow draw unclipped
		}
		if cmd.Clip != want {
			t.Fatalf("row clip: got %+v want %+v", cmd.Clip, want)
		}
	}
}
