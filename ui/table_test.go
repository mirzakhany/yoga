package ui

import (
	"testing"

	"github.com/mirzakhany/yoga/icons"
	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/theme"
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

func TestTableCellEditUnchangedDoesNotReport(t *testing.T) {
	tbl := testTable(t)
	tbl.SetRows([]TableRow{
		{ID: "r1", Cells: map[string]string{"key": "Host", "val": "localhost"}},
	})
	calls := 0
	tbl.OnCellChange = func(string, string, string) { calls++ }

	tbl.StartCellEdit("r1", "val")
	tbl.CommitCellEdit()
	tbl.StartCellEdit("r1", "val")
	tbl.editField.setValue("x")
	tbl.editField.setValue("localhost") // edited back to the original
	tbl.CommitCellEdit()
	if calls != 0 {
		t.Fatalf("OnCellChange fired %d times for an unchanged cell", calls)
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

func TestTableRowSelectAndActivate(t *testing.T) {
	tbl := NewTable([]TableColumn{
		{ID: "name", Label: "Name", Kind: TableColText, Width: 0},
	}, nil)
	tbl.Selectable = true
	tbl.SetRows([]TableRow{
		{ID: "a", Cells: map[string]string{"name": "A"}, Icon: icons.Folder},
		{ID: "b", Cells: map[string]string{"name": "B"}, Icon: icons.File},
		{ID: "c", Cells: map[string]string{"name": "C"}},
	})
	tbl.host.Style = tbl.host.Style.W(300).H(160)
	tbl.host.Calculate(300, 160)

	var clicked, activated string
	tbl.OnRowClick = func(id string) { clicked = id }
	tbl.OnRowActivate = func(id string) { activated = id }

	clickRow(tbl, 0, 0)
	if got := tbl.SelectedIDs(); len(got) != 1 || got[0] != "a" {
		t.Fatalf("single select: %v", got)
	}
	if clicked != "a" {
		t.Fatalf("OnRowClick: %q", clicked)
	}

	clickRow(tbl, 1, 0)
	if got := tbl.SelectedIDs(); len(got) != 1 || got[0] != "b" {
		t.Fatalf("click replaces selection: %v", got)
	}

	clickRow(tbl, 1, 0)
	if activated != "b" {
		t.Fatalf("double-click should activate, got %q", activated)
	}
}

func TestTableMultiSelect(t *testing.T) {
	tbl := NewTable([]TableColumn{
		{ID: "name", Label: "Name", Kind: TableColText, Width: 0},
	}, nil)
	tbl.Selectable = true
	tbl.MultiSelect = true
	tbl.SetRows([]TableRow{
		{ID: "a", Cells: map[string]string{"name": "A"}},
		{ID: "b", Cells: map[string]string{"name": "B"}},
		{ID: "c", Cells: map[string]string{"name": "C"}},
	})
	tbl.host.Style = tbl.host.Style.W(300).H(160)
	tbl.host.Calculate(300, 160)

	clickRow(tbl, 0, 0)
	clickRow(tbl, 2, input.ModCtrl)
	got := tbl.SelectedIDs()
	if len(got) != 2 || got[0] != "a" || got[1] != "c" {
		t.Fatalf("ctrl toggle: %v", got)
	}

	clickRow(tbl, 0, 0)
	clickRow(tbl, 2, input.ModShift)
	got = tbl.SelectedIDs()
	if len(got) != 3 {
		t.Fatalf("shift range: %v", got)
	}
}

func TestTableActivateEnter(t *testing.T) {
	tbl := NewTable([]TableColumn{
		{ID: "name", Label: "Name", Kind: TableColText, Width: 0},
	}, nil)
	tbl.Selectable = true
	tbl.SetRows([]TableRow{
		{ID: "a", Cells: map[string]string{"name": "A"}},
	})
	var activated string
	tbl.OnRowActivate = func(id string) { activated = id }
	tbl.Rows[0].Selected = true
	tbl.HandleKeys([]input.KeyEvent{{Key: input.KeyEnter}})
	if activated != "a" {
		t.Fatalf("Enter activate: %q", activated)
	}
}

func clickRow(tbl *Table, visibleIdx int, mods input.Mod) {
	y := tbl.host.Frame.Y + tbl.headerH + float32(visibleIdx)*tbl.rowH + tbl.rowH/2
	x := tbl.host.Frame.X + 20
	tbl.onMouse(tbl.host, &input.Mouse{X: x, Y: y, Released: true, Mods: mods})
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

func TestTableEditableOff(t *testing.T) {
	tbl := testTable(t)
	tbl.Editable = false
	tbl.SetRows([]TableRow{
		{ID: "r1", Cells: map[string]string{"key": "Host", "val": "localhost"}},
	})
	tbl.host.Style = tbl.host.Style.W(300).H(160)
	tbl.host.Calculate(300, 160)

	tbl.StartCellEdit("r1", "val")
	if tbl.editingRowID != "" {
		t.Fatal("StartCellEdit should no-op when Editable is false")
	}

	// Click editable cell (val is second flex column after checkbox).
	y := tbl.host.Frame.Y + tbl.headerH + tbl.rowH/2
	x := tbl.host.Frame.X + 120
	tbl.onMouse(tbl.host, &input.Mouse{X: x, Y: y, Released: true})
	if tbl.editingRowID != "" {
		t.Fatalf("click should not edit when Editable=false, got %q", tbl.editingRowID)
	}
}

func TestTableHighlightSelectedDefault(t *testing.T) {
	tbl := testTable(t)
	if !tbl.HighlightSelected {
		t.Fatal("HighlightSelected should default true")
	}
	tbl.HighlightSelected = false
	tbl.SetRows([]TableRow{
		{ID: "a", Cells: map[string]string{"key": "k", "val": "v"}},
	})
	tbl.Rows[0].Selected = true
	if got := tbl.SelectedIDs(); len(got) != 1 || got[0] != "a" {
		t.Fatalf("selection still works: %v", got)
	}
}

func TestTableCollapseEmpty(t *testing.T) {
	tbl := testTable(t)
	tbl.CollapseEmpty = true
	tbl.syncMetrics()
	tbl.applyHostSize()
	if tbl.host.Style.Height != tbl.headerH {
		t.Fatalf("empty collapsed height: got %v want %v", tbl.host.Style.Height, tbl.headerH)
	}
	if tbl.host.Style.Grow != 0 {
		t.Fatalf("empty collapsed grow: got %v want 0", tbl.host.Style.Grow)
	}

	tbl.SetRows([]TableRow{
		{ID: "1", Cells: map[string]string{"key": "a", "val": "1"}},
	})
	if tbl.host.Style.Height != tbl.MinHeight {
		t.Fatalf("with rows height: got %v want %v", tbl.host.Style.Height, tbl.MinHeight)
	}
	if tbl.host.Style.Grow != 1 {
		t.Fatalf("with rows grow: got %v want 1", tbl.host.Style.Grow)
	}

	tbl.SetRows(nil)
	if tbl.host.Style.Height != tbl.headerH {
		t.Fatalf("cleared rows height: got %v want %v", tbl.host.Style.Height, tbl.headerH)
	}
}

func TestTableBackgroundNilSafe(t *testing.T) {
	tbl := testTable(t)
	if tbl.Background != nil {
		t.Fatal("Background should default nil (transparent)")
	}
	chrome := theme.Current().Chrome
	tbl.Background = &chrome
	if tbl.Background == nil || *tbl.Background != chrome {
		t.Fatal("Background override should stick")
	}
}

func TestTableRowMetricsMatchControlHeight(t *testing.T) {
	tbl := testTable(t)
	th := theme.Current()
	if tbl.rowH != th.Metrics.ControlHeight || tbl.headerH != th.Metrics.ControlHeight {
		t.Fatalf("rowH=%v headerH=%v want ControlHeight=%v", tbl.rowH, tbl.headerH, th.Metrics.ControlHeight)
	}
}

func secretTable() *Table {
	return NewTable([]TableColumn{
		{ID: "key", Label: "Key", Kind: TableColEditable, Width: 0},
		{ID: "val", Label: "Value", Kind: TableColEditable, Width: 0},
		{
			ID: "sec", Kind: TableColToggle, Width: 40, Locked: true,
			IconOn: icons.Lock, IconOff: icons.LockOpen,
			TooltipOn: "Secret", TooltipOff: "Mark as secret",
		},
	}, nil)
}

// clickToggle clicks the toggle column of a visible row, the way a mouse does.
func clickToggle(tbl *Table, visibleIdx, colIdx int) {
	cw, _, _ := tbl.bodyMetrics()
	widths, offsets := tbl.columnLayout(cw)
	y := tbl.host.Frame.Y + tbl.headerH + float32(visibleIdx)*tbl.rowH - tbl.scrollY
	cr := tbl.cellRect(y, colIdx, widths, offsets)
	_, slot := tbl.actionSlotSize()
	tr := tbl.toggleSlotRect(cr, slot)
	tbl.onMouse(tbl.host, &input.Mouse{X: tr.X + tr.W/2, Y: tr.Y + tr.H/2, Released: true})
}

func TestTableToggleColumn(t *testing.T) {
	tbl := secretTable()
	tbl.SetRows([]TableRow{
		{ID: "r1", Cells: map[string]string{"key": "token", "val": "abc"}},
	})
	tbl.host.Style = tbl.host.Style.W(400).H(160)
	tbl.host.Calculate(400, 160)

	var rowID, colID, val string
	calls := 0
	tbl.OnCellChange = func(r, c, v string) {
		rowID, colID, val = r, c, v
		calls++
	}

	clickToggle(tbl, 0, 2)
	if tbl.Rows[0].Cells["sec"] != "1" {
		t.Fatalf("toggle on: %q", tbl.Rows[0].Cells["sec"])
	}
	if calls != 1 || rowID != "r1" || colID != "sec" || val != "1" {
		t.Fatalf("OnCellChange: %d %q %q %q", calls, rowID, colID, val)
	}

	clickToggle(tbl, 0, 2)
	if tbl.Rows[0].Cells["sec"] != "" {
		t.Fatalf("toggle off: %q", tbl.Rows[0].Cells["sec"])
	}
	if calls != 2 || val != "" {
		t.Fatalf("second OnCellChange: %d %q", calls, val)
	}
}

func TestTableToggleDoesNotStartEdit(t *testing.T) {
	tbl := secretTable()
	tbl.SetRows([]TableRow{
		{ID: "r1", Cells: map[string]string{"key": "token", "val": "abc"}},
	})
	tbl.host.Style = tbl.host.Style.W(400).H(160)
	tbl.host.Calculate(400, 160)

	clickToggle(tbl, 0, 2)
	if tbl.editingRowID != "" {
		t.Fatalf("toggle click started an edit on %q/%q", tbl.editingRowID, tbl.editingColID)
	}
}

func TestTableMaskedCellEditsAsPassword(t *testing.T) {
	tbl := secretTable()
	tbl.SetRows([]TableRow{
		{ID: "r1", Cells: map[string]string{"key": "token", "val": "abc", "sec": "1"}},
		{ID: "r2", Cells: map[string]string{"key": "host", "val": "localhost"}},
	})
	tbl.Masked = func(rowID, colID string) bool {
		if colID != "val" {
			return false
		}
		idx, ok := tbl.rowByID(rowID)
		return ok && tbl.Rows[idx].Cells["sec"] == "1"
	}

	tbl.StartCellEdit("r1", "val")
	if !tbl.editField.cfg.Password {
		t.Fatal("masked cell should edit as a password field")
	}
	if got := tbl.editField.Value; got != "abc" {
		t.Fatalf("edit field value: %q", got)
	}
	tbl.CommitCellEdit()

	tbl.StartCellEdit("r2", "val")
	if tbl.editField.cfg.Password {
		t.Fatal("unmasked cell should not edit as a password field")
	}
	tbl.CommitCellEdit()
}

func TestTableFilterSkipsMaskedCells(t *testing.T) {
	tbl := secretTable()
	tbl.SetRows([]TableRow{
		{ID: "r1", Cells: map[string]string{"key": "token", "val": "swordfish", "sec": "1"}},
		{ID: "r2", Cells: map[string]string{"key": "host", "val": "swordfish.local"}},
	})
	tbl.Masked = func(rowID, colID string) bool {
		if colID != "val" {
			return false
		}
		idx, ok := tbl.rowByID(rowID)
		return ok && tbl.Rows[idx].Cells["sec"] == "1"
	}

	tbl.SetFilter("swordfish")
	if len(tbl.visible) != 1 || tbl.Rows[tbl.visible[0]].ID != "r2" {
		t.Fatalf("a hidden value must not be searchable: %v", tbl.visible)
	}
}

func TestMaskText(t *testing.T) {
	if got := maskText("abc"); got != "•••" {
		t.Fatalf("maskText: %q", got)
	}
	if got := maskText("héé"); got != "•••" {
		t.Fatalf("maskText counts runes, not bytes: %q", got)
	}
	if got := maskText(""); got != "" {
		t.Fatalf("maskText empty: %q", got)
	}
}

func TestTableActionVisibleOnlyOnSomeRows(t *testing.T) {
	tbl := NewTable([]TableColumn{
		{ID: "key", Label: "Key", Kind: TableColEditable, Width: 0},
		{ID: "act", Kind: TableColActions, Width: 80, Locked: true},
	}, []TableAction{
		{Icon: icons.Eye, Tooltip: "Reveal", Visible: func(rowID string) bool { return rowID == "r1" }},
		{Icon: icons.Trash2, Tooltip: "Delete"},
	})
	tbl.SetRows([]TableRow{
		{ID: "r1", Cells: map[string]string{"key": "token"}},
		{ID: "r2", Cells: map[string]string{"key": "host"}},
	})
	tbl.host.Style = tbl.host.Style.W(400).H(160)
	tbl.host.Calculate(400, 160)

	var revealed []string
	tbl.Actions[0].OnClick = func(id string) { revealed = append(revealed, id) }

	clickAction(tbl, 0, 1, 0)
	clickAction(tbl, 1, 1, 0)
	if len(revealed) != 1 || revealed[0] != "r1" {
		t.Fatalf("a hidden action must not be clickable: %v", revealed)
	}
}

// clickAction clicks action slot actionIdx of a visible row.
func clickAction(tbl *Table, visibleIdx, colIdx, actionIdx int) {
	cw, _, _ := tbl.bodyMetrics()
	widths, offsets := tbl.columnLayout(cw)
	y := tbl.host.Frame.Y + tbl.headerH + float32(visibleIdx)*tbl.rowH - tbl.scrollY
	cr := tbl.cellRect(y, colIdx, widths, offsets)
	_, slot := tbl.actionSlotSize()
	ax := tbl.actionSlotX(cr, actionIdx)
	tbl.onMouse(tbl.host, &input.Mouse{X: ax + slot/2, Y: cr.Y + tbl.rowH/2, Released: true})
}
