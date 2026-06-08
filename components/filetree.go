package components

import (
	"os"
	"path/filepath"
	"sort"

	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
	"github.com/mirzakhany/yoga/theme"
)

// FileTree is a recursive directory explorer. It is a thin wrapper over the
// generic Tree: it supplies a lazy directory Loader, file/folder icons, and maps
// activation/right-click onto file-oriented callbacks. Directory children are
// loaded on first expand so opening the app in a large repo does not stat the
// whole tree up front.
type FileTree struct {
	tree *Tree

	OnOpenFile func(path string)
	OnChange   func()
}

// NewFileTree builds an explorer for rootPath, eagerly reading the top level so
// something shows immediately; deeper levels load on demand.
func NewFileTree(text *shape.Engine, th *theme.Theme, sheet *render.SpriteSheet, rootPath string) *FileTree {
	ft := &FileTree{}

	root := &TreeNode{Label: filepath.Base(rootPath), Data: rootPath}
	t := NewTree(text, th, sheet, root)

	// Lazy directory listing: folders first, then files, both alphabetical;
	// dotfiles hidden to keep the demo tidy.
	t.Loader = func(n *TreeNode) []*TreeNode {
		dir, _ := n.Data.(string)
		entries, err := os.ReadDir(dir)
		if err != nil {
			return nil
		}
		var kids []*TreeNode
		for _, de := range entries {
			name := de.Name()
			if len(name) > 0 && name[0] == '.' {
				continue
			}
			kids = append(kids, &TreeNode{
				Label: name,
				Data:  filepath.Join(dir, name),
				Leaf:  !de.IsDir(),
			})
		}
		sort.SliceStable(kids, func(i, j int) bool {
			a, b := kids[i], kids[j]
			if a.Leaf != b.Leaf {
				return !a.Leaf // directories (non-leaf) first
			}
			return a.Label < b.Label
		})
		return kids
	}

	t.OnActivate = func(n *TreeNode) {
		if p, ok := n.Data.(string); ok && ft.OnOpenFile != nil {
			ft.OnOpenFile(p)
		}
	}
	t.OnToggle = func(_ *TreeNode) {
		if ft.OnChange != nil {
			ft.OnChange()
		}
	}

	// Re-seed now that the Loader is set (NewTree loaded the empty root before
	// the Loader existed).
	t.SetRoot(root)

	ft.tree = t
	return ft
}

// El is the file tree's layout element (the scrollable panel).
func (ft *FileTree) El() *layout.Element { return ft.tree.El }

// MenuEl is the right-click context-menu overlay; mount it at the UI root.
func (ft *FileTree) MenuEl() *layout.Element { return ft.tree.MenuEl() }

// Tree exposes the underlying generic tree for advanced customization (icons,
// context menu, etc.).
func (ft *FileTree) Tree() *Tree { return ft.tree }

// ContentHeight is the total pixel height of all visible rows.
func (ft *FileTree) ContentHeight() float32 { return ft.tree.ContentHeight() }

// Update drives the tree's scrollbars; call once per frame after layout.
func (ft *FileTree) Update(m *input.Mouse) { ft.tree.Update(m) }

// SetFilter restricts visible rows to files/folders whose names match query.
// See Tree.SetFilter for lazy-loading limitations.
func (ft *FileTree) SetFilter(query string) { ft.tree.SetFilter(query) }

// Focus grants keyboard focus to the file tree.
func (ft *FileTree) Focus() { ft.tree.Focus() }

// Blur removes keyboard focus from the file tree.
func (ft *FileTree) Blur() { ft.tree.Blur() }

// Focused reports whether the file tree has keyboard focus.
func (ft *FileTree) Focused() bool { return ft.tree.Focused() }

// HandleText is a no-op; the file tree does not accept text input.
func (ft *FileTree) HandleText(runes []rune) { ft.tree.HandleText(runes) }

// HandleKeys processes keyboard navigation for the file tree.
func (ft *FileTree) HandleKeys(keys []input.KeyEvent) { ft.tree.HandleKeys(keys) }

// CapturesTab reports that plain Tab should move focus rather than act on the tree.
func (ft *FileTree) CapturesTab() bool { return false }

// FocusOnClick reports that clicking the tree should grant focus.
func (ft *FileTree) FocusOnClick() bool { return true }

// FocusEl returns the element used for click-to-focus hit testing.
func (ft *FileTree) FocusEl() *layout.Element { return ft.tree.El }

// SetContextMenu installs a builder for the per-file right-click menu.
func (ft *FileTree) SetContextMenu(fn func(path string) []MenuItem) {
	ft.tree.ContextMenu = func(n *TreeNode) []MenuItem {
		p, _ := n.Data.(string)
		return fn(p)
	}
}
