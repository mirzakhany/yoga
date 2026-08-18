package ui

import (
	"testing"
)

func testTable(t *testing.T) *Table {
	t.Helper()
	columns := []TableColumn{
		{ID: "sel", Kind: TableColCheckbox, Width: 36},
		{ID: "key", Label: "Key", Kind: TableColEditable, Width: 0},
		{ID: "val", Label: "Value", Kind: TableColEditable, Width: 0},
		{ID: "act", Kind: TableColActions, Width: 40},
	}
	return NewTable(columns, nil)
}

func TestTableSetFilter(t *testing.T) {
	tbl := testTable(t)
	tbl.SetRows([]TableRow{
		{ID: "1", Cells: map[string]string{"key": "Content-Type", "val": "application/json"}},
		{ID: "2", Cells: map[string]string{"key": "Authorization", "val": "Bearer token"}},
		{ID: "3", Cells: map[string]string{"key": "Accept", "val": "text/html"}},
	})
	if len(tbl.visible) != 3 {
		t.Fatalf("visible: %d", len(tbl.visible))
	}
	fullH := tbl.contentH

	tbl.SetFilter("auth")
	if len(tbl.visible) != 1 {
		t.Fatalf("filtered visible: %d", len(tbl.visible))
	}
	if tbl.contentH >= fullH {
		t.Fatalf("contentH should shrink: %v >= %v", tbl.contentH, fullH)
	}

	tbl.SetFilter("")
	if len(tbl.visible) != 3 {
		t.Fatalf("cleared filter visible: %d", len(tbl.visible))
	}
}

func TestTableSelection(t *testing.T) {
	tbl := testTable(t)
	tbl.SetRows([]TableRow{
		{ID: "a", Cells: map[string]string{"key": "k1", "val": "v1"}},
		{ID: "b", Cells: map[string]string{"key": "k2", "val": "v2"}},
	})

	tbl.Rows[0].Selected = true
	if len(tbl.SelectedIDs()) != 1 || tbl.SelectedIDs()[0] != "a" {
		t.Fatalf("selected: %v", tbl.SelectedIDs())
	}

	tbl.SelectAll(true)
	if len(tbl.SelectedIDs()) != 2 {
		t.Fatalf("select all: %v", tbl.SelectedIDs())
	}
	tbl.SelectAll(false)
	if len(tbl.SelectedIDs()) != 0 {
		t.Fatalf("deselect all: %v", tbl.SelectedIDs())
	}
}

func TestTableRemoveRow(t *testing.T) {
	tbl := testTable(t)
	tbl.SetRows([]TableRow{
		{ID: "x", Cells: map[string]string{"key": "a", "val": "1"}},
		{ID: "y", Cells: map[string]string{"key": "b", "val": "2"}},
	})

	if !tbl.RemoveRow("x") {
		t.Fatal("RemoveRow failed")
	}
	if len(tbl.Rows) != 1 || tbl.Rows[0].ID != "y" {
		t.Fatalf("rows after delete: %+v", tbl.Rows)
	}
}

func TestTableDeleteAction(t *testing.T) {
	tbl := testTable(t)
	tbl.SetRows([]TableRow{
		{ID: "x", Cells: map[string]string{"key": "a", "val": "1"}},
	})
	var deleted string
	tbl.OnDelete = func(id string) { deleted = id }
	tbl.Actions[0].OnClick = func(id string) {
		tbl.RemoveRow(id)
		if tbl.OnDelete != nil {
			tbl.OnDelete(id)
		}
	}
	tbl.Actions[0].OnClick("x")
	if len(tbl.Rows) != 0 {
		t.Fatalf("rows after delete action: %+v", tbl.Rows)
	}
	if deleted != "x" {
		t.Fatalf("OnDelete: %q", deleted)
	}
}

func TestTableCellEdit(t *testing.T) {
	tbl := testTable(t)
	tbl.SetRows([]TableRow{
		{ID: "r1", Cells: map[string]string{"key": "Host", "val": "localhost"}},
	})
	var rowID, colID, val string
	tbl.OnCellChange = func(r, c, v string) {
		rowID, colID, val = r, c, v
	}

	tbl.StartCellEdit("r1", "val")
	tbl.editField.setValue("127.0.0.1")
	tbl.CommitCellEdit()

	if tbl.Rows[0].Cells["val"] != "127.0.0.1" {
		t.Fatalf("cell value: %q", tbl.Rows[0].Cells["val"])
	}
	if rowID != "r1" || colID != "val" || val != "127.0.0.1" {
		t.Fatalf("OnCellChange: %q %q %q", rowID, colID, val)
	}
}

func TestTableSort(t *testing.T) {
	tbl := testTable(t)
	tbl.Columns[1].Sortable = true
	tbl.Columns[2].Sortable = true
	tbl.SetRows([]TableRow{
		{ID: "1", Cells: map[string]string{"key": "Zebra", "val": "z"}},
		{ID: "2", Cells: map[string]string{"key": "Alpha", "val": "a"}},
		{ID: "3", Cells: map[string]string{"key": "Mike", "val": "m"}},
	})

	tbl.SortBy("key")
	if tbl.Rows[tbl.visible[0]].Cells["key"] != "Alpha" {
		t.Fatalf("asc first: %q", tbl.Rows[tbl.visible[0]].Cells["key"])
	}
	if tbl.Rows[tbl.visible[2]].Cells["key"] != "Zebra" {
		t.Fatalf("asc last: %q", tbl.Rows[tbl.visible[2]].Cells["key"])
	}

	tbl.SortBy("key")
	if tbl.Rows[tbl.visible[0]].Cells["key"] != "Zebra" {
		t.Fatalf("desc first: %q", tbl.Rows[tbl.visible[0]].Cells["key"])
	}

	tbl.SortBy("val")
	if !tbl.SortAscending() || tbl.SortColumn() != "val" {
		t.Fatalf("sort column: %q asc=%v", tbl.SortColumn(), tbl.SortAscending())
	}
}

func TestTableLockedColumn(t *testing.T) {
	tbl := testTable(t)
	tbl.Columns[3].Locked = true
	tbl.Columns[3].Sortable = true // ignored when locked
	tbl.SetRows([]TableRow{
		{ID: "1", Cells: map[string]string{"key": "b", "val": "2"}},
		{ID: "2", Cells: map[string]string{"key": "a", "val": "1"}},
	})

	tbl.SortBy("act")
	if tbl.SortColumn() != "" {
		t.Fatalf("locked column should not sort: %q", tbl.SortColumn())
	}

	before := tbl.ColumnWidth("act")
	tbl.SetColumnWidth("act", 80)
	if tbl.ColumnWidth("act") != before {
		t.Fatalf("locked width changed: %v -> %v", before, tbl.ColumnWidth("act"))
	}
	if tbl.colResizable(2) {
		t.Fatal("resize handle should not appear left of locked column")
	}
}

func TestTableColumnResize(t *testing.T) {
	tbl := testTable(t)
	tbl.host.Style = tbl.host.Style.W(300).H(120)
	tbl.host.Calculate(300, 120)

	before := tbl.ColumnWidth("key")
	tbl.SetColumnWidth("key", 140)
	after := tbl.ColumnWidth("key")
	if after < 139 || after > 141 {
		t.Fatalf("column width: before=%v after=%v", before, after)
	}
	if after <= before {
		t.Fatalf("expected wider key column: before=%v after=%v", before, after)
	}
}

func TestTableLayoutNoOverlap(t *testing.T) {
	tbl := testTable(t)
	tbl.SetRows([]TableRow{
		{ID: "1", Cells: map[string]string{"key": "one", "val": "1"}},
		{ID: "2", Cells: map[string]string{"key": "two", "val": "2"}},
	})
	tbl.host.Style = tbl.host.Style.W(320).H(120)
	tbl.host.Calculate(320, 120)

	if tbl.host.Frame.H <= 0 {
		t.Fatal("table has no height")
	}
	if tbl.contentH != 2*tbl.rowH {
		t.Fatalf("contentH: %v want %v", tbl.contentH, 2*tbl.rowH)
	}
}
