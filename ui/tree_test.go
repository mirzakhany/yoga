package ui

import (
	"testing"

	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
	"github.com/mirzakhany/yoga/theme"
)

func setupTreeTest(t *testing.T) {
	t.Helper()
	text, err := shape.NewEngine(1, false)
	if err != nil {
		t.Fatal(err)
	}
	SetFrameResources(text, render.NewSpriteSheet(text.Atlas), nil)
}

func treeVisibleRows(t *testing.T, tr *Tree) int {
	t.Helper()
	th := theme.Current()
	rowH := th.Typography.Body.LineHeight + th.Spacing.S
	if rowH <= 0 {
		return 0
	}
	return int(tr.ContentHeight() / rowH)
}

func TestTreeStaticChildren(t *testing.T) {
	setupTreeTest(t)
	root := &TreeNode{
		Label: "root",
		Children: []*TreeNode{
			{Label: "main.go", Leaf: true},
			{Label: "app.go", Leaf: true},
		},
	}
	tr := NewTree(root)
	if got := treeVisibleRows(t, tr); got != 2 {
		t.Fatalf("visible rows: got %d want 2", got)
	}
}

func TestTreeAddChild(t *testing.T) {
	setupTreeTest(t)
	root := &TreeNode{Label: "root"}
	tr := NewTree(root)
	tr.AddChild(nil, &TreeNode{Label: "a", Leaf: true})
	tr.AddChild(nil, &TreeNode{Label: "b", Leaf: true})
	if len(tr.Root().Children) != 2 {
		t.Fatalf("children: got %d want 2", len(tr.Root().Children))
	}
	if got := treeVisibleRows(t, tr); got != 2 {
		t.Fatalf("visible rows: got %d want 2", got)
	}
}

func TestTreeRemove(t *testing.T) {
	setupTreeTest(t)
	a := &TreeNode{Label: "a", Leaf: true}
	b := &TreeNode{Label: "b", Leaf: true}
	root := &TreeNode{Label: "root", Children: []*TreeNode{a, b}}
	tr := NewTree(root)
	if !tr.Remove(a) {
		t.Fatal("Remove failed")
	}
	if len(tr.Root().Children) != 1 || tr.Root().Children[0] != b {
		t.Fatalf("children after remove: %+v", tr.Root().Children)
	}
	if got := treeVisibleRows(t, tr); got != 1 {
		t.Fatalf("visible rows: got %d want 1", got)
	}
}

func TestTreeInsertChild(t *testing.T) {
	root := &TreeNode{Label: "root"}
	root.AddChild(&TreeNode{Label: "b", Leaf: true})
	root.InsertChild(0, &TreeNode{Label: "a", Leaf: true})
	if len(root.Children) != 2 {
		t.Fatalf("children: got %d want 2", len(root.Children))
	}
	if root.Children[0].Label != "a" || root.Children[1].Label != "b" {
		t.Fatalf("order: %q, %q", root.Children[0].Label, root.Children[1].Label)
	}
}

func TestTreeLoaderSkipsExistingChildren(t *testing.T) {
	setupTreeTest(t)
	root := &TreeNode{
		Label: "root",
		Children: []*TreeNode{
			{
				Label: "folder",
				Children: []*TreeNode{
					{Label: "existing", Leaf: true},
				},
			},
		},
	}
	tr := NewTree(root)
	loaderCalled := false
	tr.Loader = func(n *TreeNode) []*TreeNode {
		loaderCalled = true
		return []*TreeNode{{Label: "from-loader", Leaf: true}}
	}
	tr.SetFilter("existing")
	if loaderCalled {
		t.Fatal("Loader should not run when Children are already set")
	}
	if got := treeVisibleRows(t, tr); got != 2 {
		t.Fatalf("visible rows: got %d want 2 (folder + existing)", got)
	}
}

func TestTreeLoaderReload(t *testing.T) {
	setupTreeTest(t)
	folder := &TreeNode{Label: "folder", Data: "folder"}
	root := &TreeNode{Label: "root", Children: []*TreeNode{folder}}
	tr := NewTree(root)
	loadCount := 0
	tr.Loader = func(n *TreeNode) []*TreeNode {
		loadCount++
		return []*TreeNode{{Label: "lazy-child", Leaf: true}}
	}
	tr.SetFilter("lazy")
	if loadCount != 1 {
		t.Fatalf("Loader calls: got %d want 1", loadCount)
	}
	if got := treeVisibleRows(t, tr); got != 2 {
		t.Fatalf("visible rows: got %d want 2", got)
	}

	folder.loaded = false
	folder.Children = nil
	tr.ensureLoaded(folder)
	if loadCount != 2 {
		t.Fatalf("Loader reload calls: got %d want 2", loadCount)
	}
	if len(folder.Children) != 1 || folder.Children[0].Label != "lazy-child" {
		t.Fatalf("reloaded children: %+v", folder.Children)
	}
}
