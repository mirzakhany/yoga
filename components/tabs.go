package components

import (
	"github.com/mirzakhany/yoga"
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
	Badge    string // optional count/label pill drawn after the title (e.g. "7")
}

// tabBadgeWidth returns the painted width of a tab's badge pill, or 0.
func tabBadgeWidth(badge string) float32 {
	if badge == "" {
		return 0
	}
	bw, _ := yoga.Text().MeasureAt(badge, theme.Current().Typography.Caption.Size)
	return bw + 10 // 5px padding each side
}

// TabBar is a horizontal strip of document tabs with a close box on each. Like
// Menu, it paints its tabs manually (rather than nesting child elements) so the
// per-tab geometry stays self-contained and is trivial to hit-test.
//
// The owner supplies the models and the two callbacks; the bar holds no document
// state of its own beyond the active index and transient hover indices.
type TabBar struct {
	El *layout.Element

	Tabs   []TabModel
	Active int

	OnActivate func(int)
	OnClose    func(int)

	hoverTab   int
	hoverClose int
	focused    bool
}

const tabMaxText = 22 // truncate titles beyond this many runes

// NewTabBar creates an empty tab bar of fixed height.
func NewTabBar() *TabBar {
	th := theme.Current()
	t := &TabBar{
		Active:     0,
		hoverTab:   -1,
		hoverClose: -1,
	}
	t.El = layout.New(layout.Box().H(th.Metrics.ControlHeight))
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
	th := theme.Current()
	f := t.El.Frame
	out := make([]tabExtent, len(t.Tabs))
	x := f.X + t.El.Style.Padding.Left
	padX := th.Spacing.M
	closeW := th.Metrics.IconSizeMD
	style := th.Typography.Body
	for i, tab := range t.Tabs {
		title := truncate(tab.Title, tabMaxText)
		tw, _ := yoga.Text().MeasureAt(title, style.Size)
		badgeW := tabBadgeWidth(tab.Badge)
		if badgeW > 0 {
			badgeW += th.Spacing.SNudge // gap before the badge
		}
		w := tw + badgeW + 2*padX + closeW
		closeX := x + w - closeW
		cy := f.Y + (f.H-closeW)/2
		out[i] = tabExtent{
			x: x,
			w: w,
			close: render.Rect{
				X: closeX,
				Y: cy,
				W: closeW - th.Spacing.XS,
				H: closeW - th.Spacing.XS,
			},
		}
		x += w
	}
	return out
}

func (t *TabBar) paint(dl *render.DrawList, text *shape.Engine) {
	th := theme.Current()
	f := t.El.Frame
	padX := th.Spacing.M
	style := th.Typography.Body
	dl.AddRect(f, th.Chrome)

	dl.PushClip(f)

	ext := t.layoutTabs()
	for i, tab := range t.Tabs {
		e := ext[i]
		rect := render.Rect{X: e.x, Y: f.Y, W: e.w, H: f.H}

		switch {
		case i == t.Active:
			dl.AddRect(rect, th.ListActive)
		case i == t.hoverTab:
			dl.AddRect(rect, th.ListHover)
		}
		if i == t.Active {
			dl.AddRect(render.Rect{X: e.x, Y: f.Y + f.H - th.Stroke.Thick, W: e.w, H: th.Stroke.Thick}, th.Accent)
		}
		if t.focused && i == t.Active {
			drawFocusRing(dl, rect, th.ListActive, th)
		}

		title := truncate(tab.Title, tabMaxText)
		tw, lh := text.MeasureAt(title, style.Size)
		ty := f.Y + (f.H-lh)/2
		text.DrawStringTopAt(dl, title, e.x+padX, ty, th.Foreground, style.Size)

		// Count badge pill after the title.
		if tab.Badge != "" {
			bsz := th.Typography.Caption.Size
			bw, bh := text.MeasureAt(tab.Badge, bsz)
			pillW := bw + 10
			pillH := bh + 2
			px := e.x + padX + tw + th.Spacing.SNudge
			py := f.Y + (f.H-pillH)/2
			dl.AddRoundedRect(render.Rect{X: px, Y: py, W: pillW, H: pillH}, th.Radius.Circular, th.ChromeMuted)
			text.DrawStringTopAt(dl, tab.Badge, px+5, py+(pillH-bh)/2, th.ForegroundMuted, bsz)
		}

		c := e.close
		if i == t.hoverClose {
			dl.AddRect(c, th.ListHover)
			yoga.Icons().Draw(dl, "close", c, th.Foreground)
		} else if tab.Modified {
			yoga.Icons().Draw(dl, "circle", shrinkRect(c, 0.5), th.ForegroundMuted)
		} else if i == t.hoverTab || i == t.Active {
			yoga.Icons().Draw(dl, "close", c, th.ForegroundMuted)
		}
	}

	dl.PopClip()
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
