package ui

import (
	"strings"
	"testing"

	"github.com/mirzakhany/yoga/highlight"
	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
)

// minimapFrame lays out ed at 1000x600 with mouse m and paints it.
func minimapFrame(c *Ctx, ed *Editor, m *input.Mouse) *render.DrawList {
	body := func(cc *Ctx) View { return Column(ViewOf(ed).Grow(1)).Grow(1) }
	root := BuildFrame(c, body, 1000, 600, m, &input.Keyboard{})
	layout.Dispatch(root, m)
	dl := &render.DrawList{}
	layout.Paint(root, dl, c.Text())
	c.EndFrame()
	return dl
}

func TestEditorMinimapTakesTextWidth(t *testing.T) {
	c := newTestCtx(t)
	ed := NewEditor([]byte(strings.Repeat("func main() {}\n", 2000)), highlight.Noop{})
	t.Cleanup(ed.Close)
	minimapFrame(c, ed, &input.Mouse{})
	without, _, _, _ := ed.scrollMetrics()
	ed.SetMinimap(true)
	minimapFrame(c, ed, &input.Mouse{})
	with, _, _, _ := ed.scrollMetrics()
	if without-with != minimapW {
		t.Fatalf("text width %v with the minimap, %v without", with, without)
	}

	quiet := NewEditor([]byte("x"), highlight.Noop{}, WithoutGutter())
	t.Cleanup(quiet.Close)
	quiet.SetMinimap(true)
	if quiet.Minimap() {
		t.Fatal("an editor without a gutter shows no minimap")
	}
}

// A press near the bottom of a long file's minimap scrolls to its end, and
// a drag back up follows the mouse.
func TestEditorMinimapScrolls(t *testing.T) {
	c := newTestCtx(t)
	ed := NewEditor([]byte(strings.Repeat("x := 1\n", 5000)), highlight.Noop{})
	t.Cleanup(ed.Close)
	ed.SetMinimap(true)
	minimapFrame(c, ed, &input.Mouse{})
	r, _ := ed.minimapGeom()
	x := r.X + r.W/2
	minimapFrame(c, ed, &input.Mouse{X: x, Y: r.Y + r.H - 1, Down: true, Pressed: true})
	_, ch, _, _ := ed.scrollMetrics()
	if end := ed.ContentHeight - ch; ed.ScrollPx < end*0.95 {
		t.Fatalf("scrolled to %v of %v", ed.ScrollPx, end)
	}
	minimapFrame(c, ed, &input.Mouse{X: x, Y: r.Y, Down: true})
	if ed.ScrollPx != 0 {
		t.Fatalf("dragging to the top scrolled to %v", ed.ScrollPx)
	}
	minimapFrame(c, ed, &input.Mouse{X: x, Y: r.Y, Released: true})
	if ed.minimapDrag || ed.Caret() != 0 {
		t.Fatal("the drag should end without moving the caret")
	}
}

// A short file's minimap shows lines where they are: a click centers that
// line, and painting draws a bar per non-blank line.
func TestEditorMinimapShortFile(t *testing.T) {
	c := newTestCtx(t)
	ed := NewEditor([]byte(strings.Repeat("abc\n\n", 80)), highlight.Noop{})
	t.Cleanup(ed.Close)
	before := len(minimapFrame(c, ed, &input.Mouse{}).Vertices)
	ed.SetMinimap(true)
	after := len(minimapFrame(c, ed, &input.Mouse{}).Vertices)
	if bars := (after - before) / 4; bars < 80 {
		t.Fatalf("%d more rects with the minimap, want a bar per line", bars)
	}
	r, _ := ed.minimapGeom()
	minimapFrame(c, ed, &input.Mouse{X: r.X + 10, Y: r.Y + 100*minimapLineH + 1, Down: true, Pressed: true})
	_, ch, _, _ := ed.scrollMetrics()
	if got := ed.rowLine(int((ed.ScrollPx + ch/2) / ed.lineH)); got != 100 {
		t.Fatalf("line %d at the middle, want 100", got)
	}
}
