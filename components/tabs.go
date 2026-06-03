package components

import (
	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
)

// TabModel is the display state of a single tab: its title and whether the
// underlying document has unsaved changes (shown as a dot in place of the close
// box until hovered).
type TabModel struct {
	Title    string
	Modified bool
}

// TabBar is a horizontal strip of document tabs with a close box on each. Like
// Menu, it paints its tabs manually (rather than nesting child elements) so the
// per-tab geometry stays self-contained and is trivial to hit-test.
//
// The owner supplies the models and the two callbacks; the bar holds no document
// state of its own beyond the active index and transient hover indices.
type TabBar struct {
	El    *layout.Element
	theme Theme
	atlas *render.FontAtlas

	Tabs   []TabModel
	Active int

	OnActivate func(int)
	OnClose    func(int)

	hoverTab   int
	hoverClose int
}

const (
	tabBarH    = 32
	tabPadX    = 12
	tabCloseW  = 18
	tabMaxText = 22 // truncate titles beyond this many runes
)

// NewTabBar creates an empty tab bar of fixed height.
func NewTabBar(atlas *render.FontAtlas, theme Theme) *TabBar {
	t := &TabBar{
		theme:      theme,
		atlas:      atlas,
		Active:     0,
		hoverTab:   -1,
		hoverClose: -1,
	}
	t.El = layout.New(layout.Box().H(tabBarH))
	t.El.Paint = t.paint
	t.El.OnMouse = t.onMouse
	return t
}

// tabExtent is the computed horizontal placement of one tab and its close box.
type tabExtent struct {
	x, w  float32
	close render.Rect
}

// layoutTabs computes each tab's x/width and close-box rect from the current
// frame. paint and onMouse share it so hit-testing always matches what's drawn.
func (t *TabBar) layoutTabs() []tabExtent {
	f := t.El.Frame
	out := make([]tabExtent, len(t.Tabs))
	x := f.X
	for i, tab := range t.Tabs {
		title := truncate(tab.Title, tabMaxText)
		tw, _ := t.atlas.Measure(title)
		w := tw + 2*tabPadX + tabCloseW
		closeX := x + w - tabCloseW
		cy := f.Y + (f.H-tabCloseW)/2
		out[i] = tabExtent{
			x: x,
			w: w,
			close: render.Rect{
				X: closeX,
				Y: cy,
				W: tabCloseW - 4,
				H: tabCloseW - 4,
			},
		}
		x += w
	}
	return out
}

func (t *TabBar) paint(dl *render.DrawList, atlas *render.FontAtlas) {
	f := t.El.Frame
	dl.AddRect(f, t.theme.Panel)

	ext := t.layoutTabs()
	for i, tab := range t.Tabs {
		e := ext[i]
		rect := render.Rect{X: e.x, Y: f.Y, W: e.w, H: f.H}

		switch {
		case i == t.Active:
			dl.AddRect(rect, t.theme.Active)
		case i == t.hoverTab:
			dl.AddRect(rect, t.theme.Hover)
		}
		// A subtle accent underline marks the active tab.
		if i == t.Active {
			dl.AddRect(render.Rect{X: e.x, Y: f.Y + f.H - 2, W: e.w, H: 2}, t.theme.Accent)
		}

		title := truncate(tab.Title, tabMaxText)
		_, th := atlas.Measure(title)
		ty := f.Y + (f.H-th)/2
		atlas.DrawText(dl, title, e.x+tabPadX, ty, t.theme.Text)

		// Close box: an "x" on hover, a modified dot otherwise (if dirty), and
		// nothing for a clean, unhovered tab.
		c := e.close
		if i == t.hoverClose {
			dl.AddRect(c, t.theme.Hover)
			drawX(dl, c, t.theme.Text)
		} else if tab.Modified {
			drawDot(dl, c, t.theme.TextDim)
		} else if i == t.hoverTab || i == t.Active {
			drawX(dl, c, t.theme.TextDim)
		}
	}
}

func (t *TabBar) onMouse(el *layout.Element, m *input.Mouse) {
	t.hoverTab = -1
	t.hoverClose = -1
	if !el.Frame.Contains(m.X, m.Y) {
		return
	}
	ext := t.layoutTabs()
	for i, e := range ext {
		if m.X < e.x || m.X > e.x+e.w {
			continue
		}
		t.hoverTab = i
		if e.close.Contains(m.X, m.Y) {
			t.hoverClose = i
			if m.Pressed {
				m.Consumed = true
				if t.OnClose != nil {
					t.OnClose(i)
				}
			}
			return
		}
		if m.Pressed {
			m.Consumed = true
			if t.OnActivate != nil {
				t.OnActivate(i)
			}
		}
		return
	}
}

// drawX renders a small "x" close glyph centered in r.
func drawX(dl *render.DrawList, r render.Rect, c render.Color) {
	const t = 1.5
	n := int(r.W)
	for i := 0; i < n; i++ {
		p := float32(i)
		dl.AddRect(render.Rect{X: r.X + p, Y: r.Y + p, W: t, H: t}, c)
		dl.AddRect(render.Rect{X: r.X + p, Y: r.Y + r.H - p - t, W: t, H: t}, c)
	}
}

// drawDot renders a small filled dot centered in r (unsaved indicator).
func drawDot(dl *render.DrawList, r render.Rect, c render.Color) {
	d := r.W * 0.5
	dl.AddRect(render.Rect{X: r.X + (r.W-d)/2, Y: r.Y + (r.H-d)/2, W: d, H: d}, c)
}

func truncate(s string, max int) string {
	rs := []rune(s)
	if len(rs) <= max {
		return s
	}
	if max <= 2 {
		return string(rs[:max])
	}
	return string(rs[:max-2]) + ".."
}
