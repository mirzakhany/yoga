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
	focused    bool
	sheet      *render.SpriteSheet
}

const tabMaxText = 22 // truncate titles beyond this many runes

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
	t.El = layout.New(layout.Box().H(theme.Metrics.ControlHeight))
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
	padX := t.theme.Spacing.M
	closeW := t.theme.Metrics.IconSizeMD
	style := t.theme.Typography.Body
	for i, tab := range t.Tabs {
		title := truncate(tab.Title, tabMaxText)
		tw, _ := t.text.MeasureAt(title, style.Size)
		w := tw + 2*padX + closeW
		closeX := x + w - closeW
		cy := f.Y + (f.H-closeW)/2
		out[i] = tabExtent{
			x: x,
			w: w,
			close: render.Rect{
				X: closeX,
				Y: cy,
				W: closeW - t.theme.Spacing.XS,
				H: closeW - t.theme.Spacing.XS,
			},
		}
		x += w
	}
	return out
}

func (t *TabBar) paint(dl *render.DrawList, text *shape.Engine) {
	f := t.El.Frame
	padX := t.theme.Spacing.M
	style := t.theme.Typography.Body
	dl.AddRect(f, t.theme.Chrome)

	ext := t.layoutTabs()
	for i, tab := range t.Tabs {
		e := ext[i]
		rect := render.Rect{X: e.x, Y: f.Y, W: e.w, H: f.H}

		switch {
		case i == t.Active:
			dl.AddRect(rect, t.theme.ListActive)
		case i == t.hoverTab:
			dl.AddRect(rect, t.theme.ListHover)
		}
		if i == t.Active {
			dl.AddRect(render.Rect{X: e.x, Y: f.Y + f.H - t.theme.Stroke.Thick, W: e.w, H: t.theme.Stroke.Thick}, t.theme.Accent)
		}
		if t.focused && i == t.Active {
			drawFocusRing(dl, rect, t.theme.ListActive, t.theme)
		}

		title := truncate(tab.Title, tabMaxText)
		_, lh := text.MeasureAt(title, style.Size)
		ty := f.Y + (f.H-lh)/2
		text.DrawStringTopAt(dl, title, e.x+padX, ty, t.theme.Foreground, style.Size)

		c := e.close
		if i == t.hoverClose {
			dl.AddRect(c, t.theme.ListHover)
			t.sheet.Draw(dl, "close", c, t.theme.Foreground)
		} else if tab.Modified {
			t.sheet.Draw(dl, "circle", shrinkRect(c, 0.5), t.theme.ForegroundMuted)
		} else if i == t.hoverTab || i == t.Active {
			t.sheet.Draw(dl, "close", c, t.theme.ForegroundMuted)
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

// Focus grants keyboard focus to the tab bar.
func (t *TabBar) Focus() { t.focused = true }

// Blur removes keyboard focus from the tab bar.
func (t *TabBar) Blur() { t.focused = false }

// Focused reports whether the tab bar has keyboard focus.
func (t *TabBar) Focused() bool { return t.focused }

// HandleText is a no-op; the tab bar does not accept text input.
func (t *TabBar) HandleText(_ []rune) {}

// HandleKeys processes Left/Right/Enter for keyboard tab switching.
func (t *TabBar) HandleKeys(keys []input.KeyEvent) {
	if !t.focused || len(t.Tabs) == 0 {
		return
	}
	for _, ev := range keys {
		if ev.Mods != 0 {
			continue
		}
		switch ev.Key {
		case input.KeyLeft:
			if t.Active > 0 {
				t.Active--
				if t.OnActivate != nil {
					t.OnActivate(t.Active)
				}
			}
		case input.KeyRight:
			if t.Active < len(t.Tabs)-1 {
				t.Active++
				if t.OnActivate != nil {
					t.OnActivate(t.Active)
				}
			}
		case input.KeyEnter:
			if t.OnActivate != nil {
				t.OnActivate(t.Active)
			}
		}
	}
}

// CapturesTab reports that plain Tab should move focus rather than act on tabs.
func (t *TabBar) CapturesTab() bool { return false }

// FocusOnClick reports that clicking a tab activates it but does not take focus.
func (t *TabBar) FocusOnClick() bool { return false }

// FocusEl returns the element used for click-to-focus hit testing.
func (t *TabBar) FocusEl() *layout.Element { return t.El }

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
