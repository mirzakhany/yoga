package ui

import (
	"testing"

	"github.com/mirzakhany/yoga/theme"
)

func TestDensityScalesWidgets(t *testing.T) {
	setupTreeTest(t)
	defer theme.UseDensity(theme.DensityDefault)
	tbl := NewTable([]TableColumn{{ID: "k", Kind: TableColText}}, nil)
	tr := NewTree(&TreeNode{Children: []*TreeNode{{Label: "a", Leaf: true}}})
	tr.Update(nil)
	defaultTree := tr.rowH

	theme.UseDensity(theme.DensityCompact)
	tbl.syncMetrics()
	tr.Update(nil)
	if tbl.rowH != 26 || tbl.headerH != 26 {
		t.Fatalf("table rows %v / header %v, want 26", tbl.rowH, tbl.headerH)
	}
	if tr.rowH >= defaultTree {
		t.Fatalf("tree row %v, want below %v", tr.rowH, defaultTree)
	}
	if h := NewMenu(100, nil).itemHeight(); h != 26 {
		t.Fatalf("menu item %v, want 26", h)
	}
}
