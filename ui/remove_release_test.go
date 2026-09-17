package ui

import (
	"runtime"
	"testing"

	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
)

// heapMB reports the live heap, which is what distinguishes a retained
// reference from memory the allocator simply has not returned to the OS.
func heapMB() float64 {
	runtime.GC()
	runtime.GC()
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	return float64(ms.HeapAlloc) / 1e6
}

const payloadSize = 4 << 20

// TestRemoveAtReleasesTail is the core guard: removing an element must not
// leave it reachable through the backing array.
func TestRemoveAtReleasesTail(t *testing.T) {
	s := make([][]byte, 0, 4)
	for i := 0; i < 4; i++ {
		s = append(s, make([]byte, payloadSize))
	}
	before := heapMB()

	// Remove from the end, the order that parks each removed element in the
	// trailing slot.
	for i := len(s) - 1; i >= 0; i-- {
		s = removeAt(s, i)
	}
	after := heapMB()
	runtime.KeepAlive(s)

	freed := before - after
	want := float64(4*payloadSize) / 1e6 * 0.9
	t.Logf("4 x %d MB: heap %.1f -> %.1f MB (freed %.1f MB)", payloadSize>>20, before, after, freed)
	if freed < want {
		t.Errorf("removeAt freed only %.1f MB of %.1f MB", freed, float64(4*payloadSize)/1e6)
	}
}

// TestTreeRemoveChildReleasesSubtree covers a detached subtree, which carries
// both its children and the application's opaque Data payload.
func TestTreeRemoveChildReleasesSubtree(t *testing.T) {
	root := &TreeNode{Label: "root"}
	kids := make([]*TreeNode, 0, 3)
	for i := 0; i < 3; i++ {
		k := &TreeNode{Label: "child", Data: make([]byte, payloadSize)}
		root.Children = append(root.Children, k)
		kids = append(kids, k)
	}
	kids = kids[:0]
	_ = kids

	before := heapMB()
	for i := len(root.Children) - 1; i >= 0; i-- {
		if !root.RemoveChild(root.Children[i]) {
			t.Fatal("RemoveChild reported not found")
		}
	}
	after := heapMB()
	runtime.KeepAlive(root)

	freed := before - after
	want := float64(3*payloadSize) / 1e6 * 0.9
	t.Logf("3 detached subtrees: heap %.1f -> %.1f MB (freed %.1f MB)", before, after, freed)
	if freed < want {
		t.Errorf("RemoveChild freed only %.1f MB of %.1f MB", freed, float64(3*payloadSize)/1e6)
	}
}

// TestTableRemoveRowReleasesCells covers a removed row's cell map.
func TestTableRemoveRowReleasesCells(t *testing.T) {
	tbl := NewTable([]TableColumn{{ID: "a", Label: "A"}}, nil)
	ids := []string{"r0", "r1", "r2"}
	for _, id := range ids {
		tbl.AddRow(TableRow{ID: id, Cells: map[string]string{"a": string(make([]byte, payloadSize))}})
	}

	before := heapMB()
	for i := len(ids) - 1; i >= 0; i-- {
		if !tbl.RemoveRow(ids[i]) {
			t.Fatalf("RemoveRow(%q) reported not found", ids[i])
		}
	}
	after := heapMB()
	runtime.KeepAlive(tbl)

	freed := before - after
	want := float64(len(ids)*payloadSize) / 1e6 * 0.9
	t.Logf("%d removed rows: heap %.1f -> %.1f MB (freed %.1f MB)", len(ids), before, after, freed)
	if freed < want {
		t.Errorf("RemoveRow freed only %.1f MB of %.1f MB", freed, float64(len(ids)*payloadSize)/1e6)
	}
}

// TestListViewClearReleasesItems covers Clear, which truncated without
// dropping the element references.
func TestListViewClearReleasesItems(t *testing.T) {
	lv := NewListView(ListViewConfig{})
	items := make([]*layout.Element, 0, 3)
	for i := 0; i < 3; i++ {
		el := layout.New(layout.Box())
		el.Paint = holdPayload(make([]byte, payloadSize))
		items = append(items, el)
	}
	lv.SetItems(items)
	items = nil

	before := heapMB()
	lv.Clear()
	after := heapMB()
	runtime.KeepAlive(lv)

	freed := before - after
	want := float64(3*payloadSize) / 1e6 * 0.9
	t.Logf("ListView.Clear of 3 items: heap %.1f -> %.1f MB (freed %.1f MB)", before, after, freed)
	if freed < want {
		t.Errorf("Clear freed only %.1f MB of %.1f MB", freed, float64(3*payloadSize)/1e6)
	}
}

// TestListViewRemoveReleasesItem covers Remove.
func TestListViewRemoveReleasesItem(t *testing.T) {
	lv := NewListView(ListViewConfig{})
	items := make([]*layout.Element, 0, 3)
	for i := 0; i < 3; i++ {
		el := layout.New(layout.Box())
		el.Paint = holdPayload(make([]byte, payloadSize))
		items = append(items, el)
	}
	lv.SetItems(items)
	items = nil

	before := heapMB()
	for i := lv.Len() - 1; i >= 0; i-- {
		if !lv.Remove(i) {
			t.Fatal("Remove reported failure")
		}
	}
	after := heapMB()
	runtime.KeepAlive(lv)

	freed := before - after
	want := float64(3*payloadSize) / 1e6 * 0.9
	t.Logf("ListView.Remove of 3 items: heap %.1f -> %.1f MB (freed %.1f MB)", before, after, freed)
	if freed < want {
		t.Errorf("Remove freed only %.1f MB of %.1f MB", freed, float64(3*payloadSize)/1e6)
	}
}

// holdPayload returns a paint func that closes over buf, so the element keeps
// it alive exactly as an application's item content would.
func holdPayload(buf []byte) func(*render.DrawList, *shape.Engine) {
	return func(*render.DrawList, *shape.Engine) { _ = buf }
}
