package ui

import (
	"fmt"
	"testing"

	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
)

// deepTree has 30 top-level folders, each holding a folder with 5 leaves.
func deepTree() (*TreeNode, *TreeNode) {
	root := &TreeNode{Label: "root"}
	var target *TreeNode
	for i := range 30 {
		dir := &TreeNode{Label: fmt.Sprintf("dir%d", i)}
		sub := &TreeNode{Label: "sub"}
		for j := range 5 {
			leaf := &TreeNode{Label: fmt.Sprintf("leaf%d-%d", i, j), Leaf: true}
			sub.Children = append(sub.Children, leaf)
			if i == 25 && j == 3 {
				target = leaf
			}
		}
		dir.Children = []*TreeNode{sub}
		root.Children = append(root.Children, dir)
	}
	return root, target
}

func TestTreeRevealExpandsSelectsAndCenters(t *testing.T) {
	c := newTestCtx(t)
	root, target := deepTree()
	tr := NewTree(root)
	body := func(*Ctx) View { return Column(ViewOf(tr).Height(300)) }
	BuildFrame(c, body, 400, 300, &input.Mouse{X: -1, Y: -1}, &input.Keyboard{})
	c.EndFrame()
	if !tr.Reveal(target) {
		t.Fatal("Reveal failed")
	}
	if tr.Selected() != target || tr.Active() != target {
		t.Fatalf("selected %v active %v, want the target", tr.Selected(), tr.Active())
	}
	idx := tr.selected
	_, clientH, _, _ := tr.scrollMetrics()
	center := float32(idx)*tr.rowH + tr.rowH/2 - tr.scrollY
	if d := center - clientH/2; d < -1 || d > 1 {
		t.Fatalf("target row center at %v of %v, want centered", center, clientH)
	}
	// Already on screen: Reveal leaves the scroll alone.
	before := tr.scrollY
	tr.Reveal(tr.visible[idx+1])
	if tr.scrollY != before {
		t.Fatalf("revealing a visible node scrolled from %v to %v", before, tr.scrollY)
	}
	if tr.Reveal(&TreeNode{Label: "stranger"}) {
		t.Fatal("Reveal of a node not in the tree should fail")
	}
	tr.Reveal(target)
	tr.Remove(target)
	if tr.Active() != nil {
		t.Fatal("removing the active node should clear it")
	}
}

func TestTreeLeadingTrailingSlots(t *testing.T) {
	c := newTestCtx(t)
	root := &TreeNode{Label: "root", Children: []*TreeNode{
		{Label: "List todos with a rather long label that runs on", Leaf: true},
		{Label: "plain", Leaf: true},
	}}
	tr := NewTree(root)
	tr.Leading = func(n *TreeNode) View {
		if n.Label == "plain" {
			return nil
		}
		return Badge("GET")
	}
	tr.Trailing = func(n *TreeNode) View { return Badge("12") }
	body := func(*Ctx) View { return Column(ViewOf(tr).Width(260).Height(200)) }
	var root2 *layout.Element
	for range 2 {
		root2 = BuildFrame(c, body, 400, 300, &input.Mouse{X: -1, Y: -1}, &input.Keyboard{})
		c.EndFrame()
	}
	_ = root2
	lead := tr.slotAt(tr.leads, 0)
	trail := tr.slotAt(tr.trails, 0)
	if lead == nil || trail == nil || tr.slotAt(tr.leads, 1) != nil {
		t.Fatal("slots missing, or a nil Leading made a slot")
	}
	f := tr.host.Frame
	clientW, _, _, _ := tr.scrollMetrics()
	if right := trail.Frame.X + trail.Frame.W; right != f.X+clientW-tr.padX() {
		t.Fatalf("trailing ends at %v, want right-aligned at %v", right, f.X+clientW-tr.padX())
	}
	if lead.Frame.X != f.X+tr.padX()+float32(1)*tr.indent()+tr.chevW() {
		t.Fatalf("leading at %v, want the icon's place", lead.Frame.X)
	}
	if lead.Frame.Y < f.Y || lead.Frame.Y+lead.Frame.H > f.Y+tr.rowH || trail.Frame.Y != lead.Frame.Y {
		t.Fatalf("slots not in row 0: lead %v trail %v", lead.Frame, trail.Frame)
	}
	var dl render.DrawList
	tr.paint(&dl, c.Text()) // label clipped before the trailing view: no panic, paints
	if len(dl.Vertices) == 0 {
		t.Fatal("tree painted nothing")
	}
}
