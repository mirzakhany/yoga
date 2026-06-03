package components

import (
	"os"
	"path/filepath"
	"sort"

	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
)

// fileNode is one entry in the explorer tree. Directory children are loaded
// lazily (on first expand) so opening the app in a large repo does not stat the
// whole tree up front.
type fileNode struct {
	name     string
	path     string
	isDir    bool
	expanded bool
	loaded   bool
	depth    int
	children []*fileNode
}

// FileTree is a recursive directory explorer rooted at a path. It paints a
// flattened list of the currently-visible rows itself (like Menu/TabBar),
// honoring per-row depth indentation. Clicking a folder toggles it; clicking a
// file invokes OnOpenFile. Any structural change (expand/collapse) calls
// OnChange so the owner can relayout if the panel's content height matters.
type FileTree struct {
	El    *layout.Element
	theme Theme
	atlas *render.FontAtlas
	sheet *render.SpriteSheet

	root    *fileNode
	visible []*fileNode // flattened, rebuilt on every structural change

	OnOpenFile func(path string)
	OnChange   func()

	hover int
	rowH  float32
}

const (
	ftIndent = 14 // px per depth level
	ftPadX   = 8
	ftIconW  = 14
)

// NewFileTree builds an explorer for rootPath, eagerly reading the top level so
// something shows immediately; deeper levels load on demand.
func NewFileTree(atlas *render.FontAtlas, theme Theme, sheet *render.SpriteSheet, rootPath string) *FileTree {
	t := &FileTree{
		theme: theme,
		atlas: atlas,
		sheet: sheet,
		hover: -1,
		rowH:  atlas.CellH + 8,
	}
	t.root = &fileNode{
		name:     filepath.Base(rootPath),
		path:     rootPath,
		isDir:    true,
		expanded: true,
		depth:    -1, // its children render at depth 0
	}
	t.load(t.root)
	t.rebuild()

	t.El = layout.New(layout.Box().Grow_(1))
	t.El.Clip = true
	t.El.Paint = t.paint
	t.El.OnMouse = t.onMouse
	return t
}

// ContentHeight is the total pixel height of all visible rows.
func (t *FileTree) ContentHeight() float32 { return float32(len(t.visible)) * t.rowH }

// load reads a directory's immediate children once, sorting folders first then
// files, both alphabetically.
func (t *FileTree) load(n *fileNode) {
	if n.loaded || !n.isDir {
		return
	}
	n.loaded = true
	entries, err := os.ReadDir(n.path)
	if err != nil {
		return
	}
	for _, de := range entries {
		name := de.Name()
		if len(name) > 0 && name[0] == '.' {
			continue // hide dotfiles to keep the demo tidy
		}
		n.children = append(n.children, &fileNode{
			name:  name,
			path:  filepath.Join(n.path, name),
			isDir: de.IsDir(),
			depth: n.depth + 1,
		})
	}
	sort.SliceStable(n.children, func(i, j int) bool {
		a, b := n.children[i], n.children[j]
		if a.isDir != b.isDir {
			return a.isDir // directories before files
		}
		return a.name < b.name
	})
}

// rebuild flattens the expanded tree into the visible slice (pre-order).
func (t *FileTree) rebuild() {
	t.visible = t.visible[:0]
	var walk func(n *fileNode)
	walk = func(n *fileNode) {
		for _, c := range n.children {
			t.visible = append(t.visible, c)
			if c.isDir && c.expanded {
				walk(c)
			}
		}
	}
	walk(t.root)
}

func (t *FileTree) toggle(n *fileNode) {
	if !n.isDir {
		return
	}
	n.expanded = !n.expanded
	if n.expanded {
		t.load(n)
	}
	t.rebuild()
	if t.OnChange != nil {
		t.OnChange()
	}
}

func (t *FileTree) paint(dl *render.DrawList, atlas *render.FontAtlas) {
	f := t.El.Frame
	dl.AddRect(f, t.theme.Panel)

	maxRows := int(f.H/t.rowH) + 1
	for i, n := range t.visible {
		if i >= maxRows {
			break // simple clip: panel does not scroll in this demo
		}
		y := f.Y + float32(i)*t.rowH
		row := render.Rect{X: f.X, Y: y, W: f.W, H: t.rowH}
		if i == t.hover {
			dl.AddRect(row, t.theme.Hover)
		}

		indent := float32(n.depth) * ftIndent
		iconX := f.X + ftPadX + indent
		iconRect := render.Rect{X: iconX, Y: y + (t.rowH-ftIconW)/2, W: ftIconW, H: ftIconW}

		if n.isDir {
			name := "square"
			if n.expanded {
				name = "chevron-down"
			}
			t.sheet.Draw(dl, name, iconRect, t.theme.Accent)
		}

		_, th := atlas.Measure(n.name)
		ty := y + (t.rowH-th)/2
		color := t.theme.Text
		if n.isDir {
			color = t.theme.Text
		} else {
			color = t.theme.TextDim
		}
		atlas.DrawText(dl, n.name, iconX+ftIconW+6, ty, color)
	}
}

func (t *FileTree) onMouse(el *layout.Element, m *input.Mouse) {
	t.hover = -1
	if !el.Frame.Contains(m.X, m.Y) {
		return
	}
	idx := int((m.Y - el.Frame.Y) / t.rowH)
	if idx < 0 || idx >= len(t.visible) {
		return
	}
	t.hover = idx
	if m.Pressed {
		m.Consumed = true
	}
	if m.Released {
		n := t.visible[idx]
		if n.isDir {
			t.toggle(n)
		} else if t.OnOpenFile != nil {
			t.OnOpenFile(n.path)
		}
	}
}
