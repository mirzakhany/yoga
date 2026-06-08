package components

import (
	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
	"github.com/mirzakhany/yoga/theme"
)

// NavOrientation is the axis along which navigation items are arranged.
type NavOrientation int

const (
	NavVertical NavOrientation = iota
	NavHorizontal
)

// NavItemLayout controls how icon and label are arranged inside each item.
type NavItemLayout int

const (
	NavIconLeft NavItemLayout = iota
	NavIconRight
	NavIconTop
	NavIconBottom
)

// NavItem is one navigation entry.
type NavItem struct {
	ID    string
	Label string
	Icon  string // sprite name; empty = text only
}

// Navigation is a selectable list of icon+text items in a vertical or horizontal strip.
type Navigation struct {
	El          *layout.Element
	Orientation NavOrientation
	ItemLayout  NavItemLayout
	Items       []NavItem
	Selected    int
	OnSelect    func(index int, id string)

	theme   *theme.Theme
	text    *shape.Engine
	sheet   *render.SpriteSheet
	itemEls []*layout.Element
	hover   int
}

// NewNavigation creates an empty navigation strip.
func NewNavigation(eng *shape.Engine, th *theme.Theme, sheet *render.SpriteSheet, orient NavOrientation, itemLayout NavItemLayout) *Navigation {
	n := &Navigation{
		theme: th, text: eng, sheet: sheet,
		Orientation: orient, ItemLayout: itemLayout,
		Selected: -1, hover: -1,
	}
	pad := th.Spacing.S
	gap := th.Spacing.XS
	box := layout.Box().Gap(gap).PaddingAll(pad)
	if orient == NavVertical {
		box = box.Direction(layout.Column).AlignItems(layout.AlignStretch)
	} else {
		box = box.Direction(layout.Row).AlignItems(layout.AlignCenter)
	}
	n.El = layout.New(box)
	n.El.Paint = n.paintContainer
	n.El.OnMouse = n.onContainerMouse
	return n
}

// Add appends an item and rebuilds layout children. Returns the item index.
func (n *Navigation) Add(item NavItem) int {
	n.Items = append(n.Items, item)
	idx := len(n.Items) - 1
	n.syncChildren()
	return idx
}

// Clear removes all items.
func (n *Navigation) Clear() {
	n.Items = nil
	n.Selected = -1
	n.hover = -1
	n.syncChildren()
}

// Select sets the active item by index and fires OnSelect when it changes.
func (n *Navigation) Select(index int) {
	if index < 0 || index >= len(n.Items) {
		return
	}
	if n.Selected == index {
		return
	}
	n.Selected = index
	if n.OnSelect != nil {
		n.OnSelect(index, n.Items[index].ID)
	}
}

// SelectID selects the item with the given id. Returns whether it was found.
func (n *Navigation) SelectID(id string) bool {
	i := n.IndexOfID(id)
	if i < 0 {
		return false
	}
	n.Select(i)
	return true
}

// SelectedID returns the id of the selected item, or "" if none.
func (n *Navigation) SelectedID() string {
	if n.Selected < 0 || n.Selected >= len(n.Items) {
		return ""
	}
	return n.Items[n.Selected].ID
}

// IndexOfID returns the index of the item with the given id, or -1.
func (n *Navigation) IndexOfID(id string) int {
	for i, item := range n.Items {
		if item.ID == id {
			return i
		}
	}
	return -1
}

func (n *Navigation) labelStyle() theme.TypographyStyle {
	switch n.ItemLayout {
	case NavIconTop, NavIconBottom:
		return n.theme.Typography.Caption
	default:
		return n.theme.Typography.Body
	}
}

func (n *Navigation) itemHeight() float32 {
	pad := n.theme.Spacing.M
	iconSz := n.theme.Metrics.IconSizeSM
	_, lh := n.text.MeasureAt("Ag", n.labelStyle().Size)
	switch n.ItemLayout {
	case NavIconTop, NavIconBottom:
		return pad*2 + iconSz + n.theme.Spacing.S + lh
	default:
		return n.theme.Metrics.ControlHeight
	}
}

func (n *Navigation) itemWidth(item NavItem) float32 {
	padX := n.theme.Spacing.M
	iconSz := n.theme.Metrics.IconSizeSM
	style := n.labelStyle()
	tw, _ := n.text.MeasureAt(item.Label, style.Size)
	gap := n.theme.Spacing.S

	switch n.ItemLayout {
	case NavIconTop, NavIconBottom:
		contentW := tw
		if item.Icon != "" && iconSz > contentW {
			contentW = iconSz
		}
		return contentW + 2*padX
	default:
		w := tw + 2*padX
		if item.Icon != "" {
			w += gap + iconSz
		}
		return w
	}
}

type navItemGeom struct {
	icon   render.Rect
	hasIcon bool
	textX  float32
	textY  float32
}

func (n *Navigation) itemGeom(f render.Rect, item NavItem, selected bool) navItemGeom {
	padX := n.theme.Spacing.M
	iconSz := n.theme.Metrics.IconSizeSM
	style := n.labelStyle()
	tw, lh := n.text.MeasureAt(item.Label, style.Size)
	gap := n.theme.Spacing.S
	if n.ItemLayout == NavIconTop || n.ItemLayout == NavIconBottom {
		gap = n.theme.Spacing.XS
	}
	hasIcon := item.Icon != ""

	var g navItemGeom
	g.hasIcon = hasIcon

	switch n.ItemLayout {
	case NavIconRight:
		textX := f.X + padX
		textY := f.Y + (f.H-lh)/2
		g.textX, g.textY = textX, textY
		if hasIcon {
			ix := textX + tw + gap
			iy := f.Y + (f.H-iconSz)/2
			g.icon = render.Rect{X: ix, Y: iy, W: iconSz, H: iconSz}
		}
	case NavIconTop:
		contentH := float32(0)
		if hasIcon {
			contentH = iconSz + gap + lh
		} else {
			contentH = lh
		}
		startY := f.Y + (f.H-contentH)/2
		if hasIcon {
			ix := f.X + (f.W-iconSz)/2
			g.icon = render.Rect{X: ix, Y: startY, W: iconSz, H: iconSz}
			g.textX = f.X + (f.W-tw)/2
			g.textY = startY + iconSz + gap
		} else {
			g.textX = f.X + (f.W-tw)/2
			g.textY = startY
		}
	case NavIconBottom:
		contentH := float32(0)
		if hasIcon {
			contentH = lh + gap + iconSz
		} else {
			contentH = lh
		}
		startY := f.Y + (f.H-contentH)/2
		g.textX = f.X + (f.W-tw)/2
		g.textY = startY
		if hasIcon {
			ix := f.X + (f.W-iconSz)/2
			g.icon = render.Rect{X: ix, Y: startY + lh + gap, W: iconSz, H: iconSz}
		}
	default: // NavIconLeft
		x := f.X + padX
		if hasIcon {
			iy := f.Y + (f.H-iconSz)/2
			g.icon = render.Rect{X: x, Y: iy, W: iconSz, H: iconSz}
			x += iconSz + gap
		}
		g.textX = x
		g.textY = f.Y + (f.H-lh)/2
	}
	_ = selected
	return g
}

func (n *Navigation) syncChildren() {
	h := n.itemHeight()
	children := make([]*layout.Element, 0, len(n.Items))
	n.itemEls = make([]*layout.Element, 0, len(n.Items))

	for i, item := range n.Items {
		idx := i
		var style layout.Style
		if n.Orientation == NavVertical {
			style = layout.Box().H(h).FlexShrink(0)
		} else {
			style = layout.Box().W(n.itemWidth(item)).H(h).FlexShrink(0)
		}
		el := layout.New(style)
		el.Paint = func(dl *render.DrawList, text *shape.Engine) {
			n.paintItem(dl, text, idx, el)
		}
		el.OnMouse = func(e *layout.Element, m *input.Mouse) {
			n.onItemMouse(idx, e, m)
		}
		children = append(children, el)
		n.itemEls = append(n.itemEls, el)
	}

	n.El.Children = children
	n.El.MarkDirty()
}

func (n *Navigation) paintContainer(dl *render.DrawList, _ *shape.Engine) {
	dl.AddRect(n.El.Frame, n.theme.Chrome)
}

func (n *Navigation) paintItem(dl *render.DrawList, text *shape.Engine, idx int, el *layout.Element) {
	f := el.Frame
	item := n.Items[idx]
	selected := idx == n.Selected
	hovered := idx == n.hover
	r := n.theme.Radius.Large

	switch {
	case selected:
		dl.AddRoundedRect(f, r, n.theme.ListActive)
	case hovered:
		dl.AddRoundedRect(f, r, n.theme.ListHover)
	}

	geom := n.itemGeom(f, item, selected)
	style := n.labelStyle()
	textCol := n.theme.Foreground
	iconCol := n.theme.ForegroundMuted
	if selected || hovered {
		iconCol = n.theme.Foreground
	}

	if geom.hasIcon {
		n.sheet.Draw(dl, item.Icon, geom.icon, iconCol)
	}
	if item.Label != "" {
		text.DrawStringTopAt(dl, item.Label, geom.textX, geom.textY, textCol, style.Size)
	}
}

func (n *Navigation) onItemMouse(idx int, e *layout.Element, m *input.Mouse) {
	if !e.Frame.Contains(m.X, m.Y) {
		return
	}
	n.hover = idx
	if m.Released {
		n.Select(idx)
		m.Consumed = true
	}
}

func (n *Navigation) onContainerMouse(e *layout.Element, m *input.Mouse) {
	if !e.Frame.Contains(m.X, m.Y) {
		n.hover = -1
		return
	}
	for _, child := range n.itemEls {
		if child.Frame.Contains(m.X, m.Y) {
			return
		}
	}
	n.hover = -1
}
