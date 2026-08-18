package ui

import (
	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
	"github.com/mirzakhany/yoga/theme"
)

// ----------------------------------------------------------------------------
// Menu: an absolutely-positioned overlay that is painted and
// hit-tested on top of the normal tree (Z-axis ordering). The menu paints its
// own item rows rather than nesting child elements, which keeps overlay
// geometry self-contained.
// ----------------------------------------------------------------------------

// MenuItem is a single selectable menu entry.
type MenuItem struct {
	Label    string
	OnSelect func()
}

type Menu struct {
	El    *layout.Element
	items []MenuItem
	width float32

	Open  bool
	hover int
}

// NewMenu builds a closed overlay menu. Add its El to the root of the tree so
// that its absolute Left/Top are interpreted as screen coordinates.
func NewMenu(width float32, items []MenuItem) *Menu {
	mu := &Menu{items: items, width: width, hover: -1}
	mu.El = layout.New(layout.Box())
	mu.El.Overlay = true // render above and hit-test before the base tree
	mu.El.Paint = mu.paint
	mu.El.OnMouse = mu.onMouse
	return mu
}

func (mu *Menu) itemHeight() float32 {
	th := theme.Current()
	if th.Metrics.MenuItemHeight > 0 {
		return th.Metrics.MenuItemHeight
	}
	return th.Metrics.ControlHeight
}

// OpenAt positions and shows the menu at the given screen coordinates, shifted
// to stay inside the viewport recorded via SetViewport (if any).
func (mu *Menu) OpenAt(x, y float32) {
	mu.Open = true
	h := float32(len(mu.items)) * mu.itemHeight()
	x, y = clampToViewport(x, y, mu.width, h)
	mu.El.Style = layout.Box().Absolute(x, y).Size(mu.width, h)
	mu.El.ReapplyStyle()
	// The menu is a screen-space overlay mounted on the root, so its absolute
	// frame is exactly {x, y, width, height}. Seed it now so the menu paints in
	// the right place on the same frame it opens: OpenAt runs during input
	// dispatch, which is after the layout pass, so without this the next paint
	// would use the stale zeroed frame and flash at the window origin until the
	// following Calculate. The next layout pass recomputes the identical frame.
	mu.El.Frame = render.Rect{X: x, Y: y, W: mu.width, H: h}
}

// Close hides the menu.
func (mu *Menu) Close() { mu.Open = false; mu.hover = -1 }

// SetItems replaces the menu's entries. Call before OpenAt when the items depend
// on context (e.g. which tree row was right-clicked). If the menu is already
// open, its height is refreshed in place to match the new item count.
func (mu *Menu) SetItems(items []MenuItem) {
	mu.items = items
	if mu.Open {
		h := float32(len(items)) * mu.itemHeight()
		mu.El.Style.Height = h
		mu.El.Frame.H = h
	}
}

func (mu *Menu) paint(dl *render.DrawList, text *shape.Engine) {
	if !mu.Open {
		return
	}
	th := theme.Current()
	f := mu.El.Frame
	itemH := mu.itemHeight()
	padX := th.Spacing.MNudge
	r := th.Radius.Medium
	drawElevationShadow(dl, f, r, th.Elevation.ShadowMd)
	dl.AddRoundedRectBorder(f, r, th.Stroke.Thin, th.Chrome, th.Border)
	// Labels wider than the configured menu width are clipped to the frame.
	dl.PushClip(f)
	for i, it := range mu.items {
		row := render.Rect{X: f.X, Y: f.Y + float32(i)*itemH, W: f.W, H: itemH}
		if i == mu.hover {
			dl.AddRect(row, th.ListHover)
		}
		style := th.Typography.Body
		_, lh := text.MeasureAt(it.Label, style.Size)
		text.DrawStringTopAt(dl, it.Label, row.X+padX, row.Y+(itemH-lh)/2, th.Foreground, style.Size)
	}
	dl.PopClip()
}

func (mu *Menu) onMouse(e *layout.Element, m *input.Mouse) {
	if !mu.Open {
		return
	}
	if e.Frame.Contains(m.X, m.Y) {
		idx := int((m.Y - e.Frame.Y) / mu.itemHeight())
		mu.hover = idx
		m.Consumed = true // block all events (including hover) from reaching layers below
		if m.Released && idx >= 0 && idx < len(mu.items) {
			if fn := mu.items[idx].OnSelect; fn != nil {
				fn()
			}
			mu.Close()
		}
	} else {
		mu.hover = -1
		if m.Pressed { // click outside closes the menu
			mu.Close()
			m.Consumed = true
		}
	}
}
