package components

import (
	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
	"github.com/mirzakhany/yoga/theme"
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
	theme *theme.Theme
	text *shape.Engine

	Tabs   []TabModel
	Active int

	OnActivate func(int)
	OnClose    func(int)

	hoverTab   int
	hoverClose int
	sheet      *render.SpriteSheet
}

const (
	tabBarH    = 32
	tabPadX    = 12
	tabCloseW  = 18
	tabMaxText = 22 // truncate titles beyond this many runes
)

// NewTabBar creates an empty tab bar of fixed height.
func NewTabBar(text *shape.Engine, theme *theme.Theme) *TabBar {
	t := &TabBar{
		theme:      theme,
		text:       text,
		sheet:      render.NewSpriteSheet(text.Atlas),
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
		tw, _ := t.text.Measure(title)
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

func (t *TabBar) paint(dl *render.DrawList, text *shape.Engine) {
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
		_, th := text.Measure(title)
		ty := f.Y + (f.H-th)/2
		text.DrawStringTop(dl, title, e.x+tabPadX, ty, t.theme.Text)

		// Close box: a close icon on hover, a modified dot otherwise (if dirty),
		// and nothing for a clean, unhovered tab.
		c := e.close
		if i == t.hoverClose {
			dl.AddRect(c, t.theme.Hover)
			t.sheet.Draw(dl, "close", c, t.theme.Text)
		} else if tab.Modified {
			t.sheet.Draw(dl, "circle", shrinkRect(c, 0.5), t.theme.TextDim)
		} else if i == t.hoverTab || i == t.Active {
			t.sheet.Draw(dl, "close", c, t.theme.TextDim)
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

// shrinkRect returns r scaled about its center by factor (0..1).
func shrinkRect(r render.Rect, factor float32) render.Rect {
	w, h := r.W*factor, r.H*factor
	return render.Rect{X: r.X + (r.W-w)/2, Y: r.Y + (r.H-h)/2, W: w, H: h}
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
