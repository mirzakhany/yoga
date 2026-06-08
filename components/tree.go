package components

import (
	"strings"

	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
	"github.com/mirzakhany/yoga/theme"
)

// TreeNode is one item in a Tree. Applications build a node hierarchy (or supply
// a Loader for lazy children) and attach arbitrary data via Data. Icons are
// customizable per node, or globally via Tree.IconFor.
type TreeNode struct {
	Label string

	// Icon is the leaf glyph (default "file"). OpenIcon/ClosedIcon are the branch
	// glyphs (defaults "folder_open"/"folder"). Any may be overridden globally by
	// Tree.IconFor.
	Icon       string
	OpenIcon   string
	ClosedIcon string

	// Leaf forces a node to be non-expandable even if it has (or could load)
	// children.
	Leaf bool

	// Data is an opaque payload the application can read back in callbacks.
	Data any

	// internal flattening/expansion state
	expanded bool
	loaded   bool
	depth    int
	children []*TreeNode
	parent   *TreeNode
}

// Tree is a scrollable, expandable tree view. It paints a flattened list of the
// currently-visible rows itself (like Menu/TabBar) and owns its own vertical and
// horizontal scrollbars. Content is clipped to the viewport via the renderer's
// scissor support, so horizontally-scrolled long labels do not bleed out of the
// panel.
//
// It is data-generic: nodes carry a Data payload, children can load lazily via
// Loader, and icons / context menu / activation are all driven by hooks.
type Tree struct {
	El    *layout.Element
	theme *theme.Theme
	text *shape.Engine
	sheet *render.SpriteSheet

	root    *TreeNode
	visible []*TreeNode // flattened, rebuilt on every structural change

	vbar, hbar *Scrollbar
	menu       *Menu

	scrollX, scrollY   float32
	contentW, contentH float32

	// ChevronOpen/ChevronClosed are the expand indicator glyphs for branches.
	ChevronOpen   string
	ChevronClosed string

	// Loader returns a node's children the first time it is expanded. If nil, a
	// node's pre-populated children slice is used as-is.
	Loader func(n *TreeNode) []*TreeNode

	// IconFor optionally overrides the icon (and its color) for a node, given its
	// expanded state. If nil, the node's Icon/OpenIcon/ClosedIcon fields (with
	// sensible defaults) and theme colors are used.
	IconFor func(n *TreeNode, expanded bool) (name string, color render.Color)

	// ContextMenu builds the right-click menu items for a node. If nil (or it
	// returns no items) no menu is shown.
	ContextMenu func(n *TreeNode) []MenuItem

	// OnActivate fires when a leaf is clicked (or a branch is double-purposed by
	// the app). OnToggle fires after a branch expands/collapses.
	OnActivate func(n *TreeNode)
	OnToggle   func(n *TreeNode)

	hover    int
	selected int
	focused  bool
	rowH     float32

	// filter restricts visible rows to subtrees whose labels match (case-insensitive
	// substring). Empty means no filter. With a lazy Loader, only loaded branches
	// are searched.
	filter string
}

const treeMenuW = 180

// NewTree builds a tree rooted at root (root itself is not drawn; its children
// are the top-level rows). sheet provides the icon glyphs.
func NewTree(text *shape.Engine, th *theme.Theme, sheet *render.SpriteSheet, root *TreeNode) *Tree {
	t := &Tree{
		theme:         th,
		text:          text,
		sheet:         sheet,
		root:          root,
		hover:         -1,
		selected:      -1,
		rowH:          th.Typography.Body.LineHeight + th.Spacing.S,
		ChevronOpen:   "expand_more",
		ChevronClosed: "chevron_right",
	}
	if t.root == nil {
		t.root = &TreeNode{}
	}
	t.root.expanded = true

	barSize := th.Metrics.ScrollbarSize
	t.vbar = NewScrollbarAxis(th, Vertical, &t.scrollY, &t.contentH, barSize)
	t.hbar = NewScrollbarAxis(th, Horizontal, &t.scrollX, &t.contentW, barSize)
	t.menu = NewMenu(text, th, treeMenuW, nil)

	t.El = layout.New(layout.Box().FlexGrow(1), t.vbar.El, t.hbar.El)
	t.El.Clip = true
	t.El.Paint = t.paint
	t.El.OnMouse = t.onMouse

	t.ensureLoaded(t.root)
	t.rebuild()
	return t
}

// SetRoot replaces the root node and rebuilds the visible list. The root is
// (re)loaded with the current Loader, so it is safe to call after assigning
// Tree.Loader.
func (t *Tree) SetRoot(root *TreeNode) {
	t.root = root
	if t.root == nil {
		t.root = &TreeNode{}
	}
	t.root.expanded = true
	t.root.loaded = false // force a fresh load via the (possibly new) Loader
	t.ensureLoaded(t.root)
	t.rebuild()
}

// MenuEl returns the context-menu overlay element. Mount it at the root of the
// UI tree so its absolute position is interpreted in screen space.
func (t *Tree) MenuEl() *layout.Element { return t.menu.El }

// ContentHeight is the total pixel height of all visible rows.
func (t *Tree) ContentHeight() float32 { return t.contentH }

// Root returns the (undrawn) root node so callers can mutate the hierarchy.
func (t *Tree) Root() *TreeNode { return t.root }

// ensureLoaded populates a branch's children once, via Loader if provided.
func (t *Tree) ensureLoaded(n *TreeNode) {
	if n.loaded || n.Leaf {
		return
	}
	n.loaded = true
	if t.Loader != nil {
		n.children = t.Loader(n)
	}
	for _, c := range n.children {
		c.parent = n
		c.depth = n.depth + 1
	}
}

// SetFilter restricts visible rows to nodes whose labels match query (case-insensitive
// substring), auto-expanding branches that contain matches. Pass "" to clear.
func (t *Tree) SetFilter(query string) {
	t.filter = query
	t.rebuild()
}

func (t *Tree) labelMatches(n *TreeNode) bool {
	if t.filter == "" {
		return true
	}
	return strings.Contains(strings.ToLower(n.Label), strings.ToLower(t.filter))
}

func (t *Tree) subtreeMatches(n *TreeNode) bool {
	if t.labelMatches(n) {
		return true
	}
	if n.Leaf {
		return false
	}
	t.ensureLoaded(n)
	for _, c := range n.children {
		if t.subtreeMatches(c) {
			return true
		}
	}
	return false
}

// rebuild flattens expanded branches into the visible slice (pre-order) and
// recomputes the content extents that drive the scrollbars.
func (t *Tree) rebuild() {
	t.visible = t.visible[:0]
	if t.filter == "" {
		var walk func(n *TreeNode)
		walk = func(n *TreeNode) {
			for _, c := range n.children {
				c.parent = n
				c.depth = n.depth + 1
				t.visible = append(t.visible, c)
				if !c.Leaf && c.expanded {
					walk(c)
				}
			}
		}
		walk(t.root)
	} else {
		var walk func(n *TreeNode)
		walk = func(n *TreeNode) {
			t.ensureLoaded(n)
			for _, c := range n.children {
				if !t.subtreeMatches(c) {
					continue
				}
				c.parent = n
				c.depth = n.depth + 1
				if !c.Leaf {
					c.expanded = true
				}
				t.visible = append(t.visible, c)
				if !c.Leaf {
					walk(c)
				}
			}
		}
		walk(t.root)
	}
	t.computeContentSize()
}

func (t *Tree) barSize() float32     { return t.theme.Metrics.ScrollbarSize }
func (t *Tree) indent() float32      { return t.theme.Metrics.TreeIndent }
func (t *Tree) padX() float32        { return t.theme.Spacing.S }
func (t *Tree) iconW() float32       { return t.theme.Metrics.TreeIconSize }
func (t *Tree) chevW() float32       { return t.theme.Metrics.TreeChevronSize }
func (t *Tree) labelGap() float32    { return t.theme.Spacing.SNudge }

// scrollMetrics computes the content viewport (area not covered by scrollbars)
// and whether each bar should be shown. When both axes overflow, the corner is
// reserved so the last row is not hidden under the horizontal bar.
func (t *Tree) scrollMetrics() (clientW, clientH float32, vShow, hShow bool) {
	f := t.El.Frame
	clientW, clientH = f.W, f.H
	vShow = t.contentH > clientH
	bar := t.barSize()
	if vShow {
		clientW = f.W - bar
	}
	hShow = t.contentW > clientW
	if hShow {
		clientH = f.H - bar
	}
	if t.contentH > clientH {
		vShow = true
		clientW = f.W - bar
		hShow = t.contentW > clientW
		if hShow {
			clientH = f.H - bar
		}
	}
	return clientW, clientH, vShow, hShow
}

func (t *Tree) contentViewport() render.Rect {
	f := t.El.Frame
	cw, ch, _, _ := t.scrollMetrics()
	return render.Rect{X: f.X, Y: f.Y, W: cw, H: ch}
}

// syncScrollbarLayout pins the vertical bar above the horizontal bar and the
// horizontal bar left of the vertical bar when both are visible.
func (t *Tree) syncScrollbarLayout(vShow, hShow bool) {
	if vShow {
		bottom := float32(0)
		if hShow {
			bottom = t.barSize()
		}
		t.vbar.El.Style = layout.Box().W(t.barSize()).AbsTop(0).AbsRight(0).AbsBottom(bottom)
		t.vbar.El.ReapplyStyle()
	}
	if hShow {
		right := float32(0)
		if vShow {
			right = t.barSize()
		}
		t.hbar.El.Style = layout.Box().H(t.barSize()).AbsLeft(0).AbsRight(right).AbsBottom(0)
		t.hbar.El.ReapplyStyle()
	}
}

func (t *Tree) clampScroll() {
	clientW, clientH, _, _ := t.scrollMetrics()
	t.scrollY = clampf(t.scrollY, 0, f32max(0, t.contentH-clientH))
	t.scrollX = clampf(t.scrollX, 0, f32max(0, t.contentW-clientW))
}

// computeContentSize measures the widest row and the total height so the
// horizontal/vertical scrollbars can size their thumbs.
func (t *Tree) computeContentSize() {
	var maxW float32
	for _, n := range t.visible {
		lw, _ := t.text.Measure(n.Label)
		rowW := t.padX() + float32(n.depth)*t.indent() + t.chevW() + t.iconW() + t.labelGap() + lw + t.padX()
		if rowW > maxW {
			maxW = rowW
		}
	}
	t.contentW = maxW
	t.contentH = float32(len(t.visible)) * t.rowH
}

// toggle expands/collapses a branch (loading children on first expand).
func (t *Tree) toggle(n *TreeNode) {
	if n.Leaf {
		return
	}
	n.expanded = !n.expanded
	if n.expanded {
		t.ensureLoaded(n)
	}
	t.rebuild()
	if t.OnToggle != nil {
		t.OnToggle(n)
	}
}

// branch reports whether a node should be treated as expandable.
func (n *TreeNode) branch() bool { return !n.Leaf }

func (t *Tree) iconFor(n *TreeNode) (string, render.Color) {
	if t.IconFor != nil {
		return t.IconFor(n, n.expanded)
	}
	if n.branch() {
		name := n.ClosedIcon
		if name == "" {
			name = "folder"
		}
		if n.expanded {
			if n.OpenIcon != "" {
				name = n.OpenIcon
			} else {
				name = "folder_open"
			}
		}
		return name, t.theme.Accent
	}
	name := n.Icon
	if name == "" {
		name = "file"
	}
	return name, t.theme.ForegroundMuted
}

func (t *Tree) paint(dl *render.DrawList, text *shape.Engine) {
	f := t.El.Frame
	vp := t.contentViewport()
	dl.AddRect(f, t.theme.Chrome)

	// Clip rows to the content viewport (above the horizontal bar when shown).
	dl.PushClip(vp)

	first := int(t.scrollY / t.rowH)
	if first < 0 {
		first = 0
	}
	for i := first; i < len(t.visible); i++ {
		y := f.Y + float32(i)*t.rowH - t.scrollY
		if y >= vp.Y+vp.H {
			break
		}
		n := t.visible[i]

		if t.focused && i == t.selected {
			dl.AddRect(render.Rect{X: f.X, Y: y, W: f.W, H: t.rowH}, t.theme.ListActive)
		} else if i == t.hover {
			dl.AddRect(render.Rect{X: f.X, Y: y, W: f.W, H: t.rowH}, t.theme.ListHover)
		}

		baseX := f.X - t.scrollX + t.padX() + float32(n.depth)*t.indent()
		chevW := t.chevW()
		iconW := t.iconW()
		style := t.theme.Typography.Body

		if n.branch() {
			chev := t.ChevronClosed
			if n.expanded {
				chev = t.ChevronOpen
			}
			r := render.Rect{X: baseX, Y: y + (t.rowH-chevW)/2, W: chevW, H: chevW}
			t.sheet.Draw(dl, chev, r, t.theme.ForegroundMuted)
		}

		iconX := baseX + chevW
		name, col := t.iconFor(n)
		t.sheet.Draw(dl, name, render.Rect{X: iconX, Y: y + (t.rowH-iconW)/2, W: iconW, H: iconW}, col)

		labelColor := t.theme.Foreground
		if !n.branch() {
			labelColor = t.theme.ForegroundMuted
		}
		_, lh := text.MeasureAt(n.Label, style.Size)
		text.DrawStringTopAt(dl, n.Label, iconX+iconW+t.labelGap(), y+(t.rowH-lh)/2, labelColor, style.Size)
	}

	dl.PopClip()
}

// overScrollbar reports whether the cursor is over a currently-visible bar so
// row hit-testing can defer to the scrollbar's own drag handling.
func (t *Tree) overScrollbar(m *input.Mouse) bool {
	_, _, vShow, hShow := t.scrollMetrics()
	if vShow && t.vbar.El.Frame.Contains(m.X, m.Y) {
		return true
	}
	if hShow && t.hbar.El.Frame.Contains(m.X, m.Y) {
		return true
	}
	return false
}

func (t *Tree) onMouse(el *layout.Element, m *input.Mouse) {
	t.hover = -1
	if !el.Frame.Contains(m.X, m.Y) || t.overScrollbar(m) {
		return
	}
	idx := int((m.Y - el.Frame.Y + t.scrollY) / t.rowH)
	if idx < 0 || idx >= len(t.visible) {
		return
	}
	t.hover = idx
	t.selected = idx
	n := t.visible[idx]

	// Right-click: open the context menu for this node.
	if m.RightPressed && t.ContextMenu != nil {
		if items := t.ContextMenu(n); len(items) > 0 {
			t.menu.SetItems(items)
			t.menu.OpenAt(m.X, m.Y)
			m.Consumed = true
			return
		}
	}

	if m.Pressed {
		m.Consumed = true
	}
	if m.Released {
		if n.branch() {
			t.toggle(n)
		} else if t.OnActivate != nil {
			t.OnActivate(n)
		}
	}
}

// Focus grants keyboard focus to the tree.
func (t *Tree) Focus() {
	t.focused = true
	if t.selected < 0 && len(t.visible) > 0 {
		t.selected = 0
	}
	t.ensureSelectedVisible()
}

// Blur removes keyboard focus from the tree.
func (t *Tree) Blur() { t.focused = false }

// Focused reports whether the tree has keyboard focus.
func (t *Tree) Focused() bool { return t.focused }

// CapturesTab reports that plain Tab should move focus rather than act on the tree.
func (t *Tree) CapturesTab() bool { return false }

// FocusOnClick reports that clicking the tree should grant focus.
func (t *Tree) FocusOnClick() bool { return true }

// FocusEl returns the element used for click-to-focus hit testing.
func (t *Tree) FocusEl() *layout.Element { return t.El }

// HandleText is a no-op; the tree does not accept text input.
func (t *Tree) HandleText(_ []rune) {}

// HandleKeys processes arrow-key navigation and Enter for the focused tree.
func (t *Tree) HandleKeys(keys []input.KeyEvent) {
	if !t.focused || len(t.visible) == 0 {
		return
	}
	if t.selected < 0 {
		t.selected = 0
	}
	for _, ev := range keys {
		if ev.Mods != 0 {
			continue
		}
		switch ev.Key {
		case input.KeyUp:
			if t.selected > 0 {
				t.selected--
				t.ensureSelectedVisible()
			}
		case input.KeyDown:
			if t.selected < len(t.visible)-1 {
				t.selected++
				t.ensureSelectedVisible()
			}
		case input.KeyRight:
			n := t.visible[t.selected]
			if n.branch() && !n.expanded {
				t.toggle(n)
			} else if t.selected < len(t.visible)-1 {
				t.selected++
				t.ensureSelectedVisible()
			}
		case input.KeyLeft:
			n := t.visible[t.selected]
			if n.branch() && n.expanded {
				t.toggle(n)
			} else if n.parent != nil && n.parent != t.root {
				for i, v := range t.visible {
					if v == n.parent {
						t.selected = i
						t.ensureSelectedVisible()
						break
					}
				}
			}
		case input.KeyEnter:
			n := t.visible[t.selected]
			if n.branch() {
				t.toggle(n)
			} else if t.OnActivate != nil {
				t.OnActivate(n)
			}
		}
	}
}

func (t *Tree) ensureSelectedVisible() {
	if t.selected < 0 || t.selected >= len(t.visible) {
		return
	}
	top := float32(t.selected) * t.rowH
	bottom := top + t.rowH
	_, clientH, _, _ := t.scrollMetrics()
	if top < t.scrollY {
		t.scrollY = top
	} else if bottom > t.scrollY+clientH {
		t.scrollY = bottom - clientH
	}
	t.clampScroll()
}

// Update drives the scrollbars (wheel + drag) and keeps offsets clamped. Call it
// once per frame after layout, before paint (mirrors the editor+scrollbar
// pattern). The owner must also dispatch mouse events to the tree's El.
func (t *Tree) Update(m *input.Mouse) {
	t.computeContentSize()
	_, _, vShow, hShow := t.scrollMetrics()
	t.syncScrollbarLayout(vShow, hShow)
	vp := t.contentViewport()
	t.vbar.Update(m, vp)
	t.hbar.Update(m, vp)
	t.clampScroll()
}
