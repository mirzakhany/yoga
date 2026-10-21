package ui

import (
	"testing"

	"github.com/mirzakhany/yoga/icons"
	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
)

func secretColTable() *Table {
	tbl := NewTable([]TableColumn{
		{ID: "key", Label: "Key", Kind: TableColEditable},
		{ID: "val", Label: "Value", Kind: TableColSecret,
			SecretWhen: func(r TableRow) bool { return r.Cells["secret"] == "1" }},
		{ID: "tags", Label: "Tags", Kind: TableColBadges},
	}, nil)
	tbl.SetRows([]TableRow{
		{ID: "r1", Cells: map[string]string{"key": "token", "val": "swordfish", "secret": "1", "tags": "auth, prod"}},
		{ID: "r2", Cells: map[string]string{"key": "host", "val": "swordfish.local"}},
	})
	tbl.host.Style = tbl.host.Style.W(600).H(200)
	tbl.host.Calculate(600, 200)
	return tbl
}

// eyeAt is the reveal button rect of a visible row's cell.
func eyeAt(tbl *Table, vi, col int) render.Rect {
	cw, _, _ := tbl.bodyMetrics()
	widths, offsets := tbl.columnLayout(cw)
	y := tbl.host.Frame.Y + tbl.headerH + float32(vi)*tbl.rowH - tbl.scrollY
	return tbl.eyeRect(tbl.cellRect(y, col, widths, offsets))
}

func TestTableSecretColumnMasksAndReveals(t *testing.T) {
	tbl := secretColTable()
	if !tbl.cellMasked("r1", "val") || tbl.cellMasked("r2", "val") {
		t.Fatal("SecretWhen should mask r1 only")
	}
	tbl.SetFilter("swordfish")
	if len(tbl.visible) != 1 || tbl.Rows[tbl.visible[0]].ID != "r2" {
		t.Fatalf("a masked value must not be searchable: %v", tbl.visible)
	}
	tbl.SetFilter("")

	er := eyeAt(tbl, 0, 1)
	tbl.onMouse(tbl.host, &input.Mouse{X: er.X + er.W/2, Y: er.Y + er.H/2, Released: true})
	if tbl.cellMasked("r1", "val") {
		t.Fatal("clicking the eye should reveal the cell")
	}
	if tbl.editingRowID != "" {
		t.Fatal("the eye must not start an edit")
	}
	tbl.ToggleReveal("r1", "val")
	if !tbl.cellMasked("r1", "val") {
		t.Fatal("toggling again should mask it")
	}
}

func TestTableSecretEditCopiesRealValue(t *testing.T) {
	tbl := secretColTable()
	clip := &input.MemClipboard{}
	SetFrameResources(nil, nil, clip)
	defer SetFrameResources(nil, nil, nil)

	tbl.StartCellEdit("r1", "val")
	if !tbl.editField.cfg.Password {
		t.Fatal("a secret cell edits masked")
	}
	tbl.editField.SelectAll()
	if !tbl.editField.Copy() || clip.Get() != "swordfish" {
		t.Fatalf("copy got %q, want the real value", clip.Get())
	}
	tbl.CommitCellEdit()

	// Masked (not secret) cells still never copy.
	tbl.Masked = func(rowID, colID string) bool { return colID == "key" }
	tbl.StartCellEdit("r1", "key")
	if tbl.editField.Copy() {
		t.Fatal("a Masked cell must not copy")
	}
	tbl.CommitCellEdit()
}

func TestTableBadgesSplit(t *testing.T) {
	got := tableBadges(" auth, prod ,, beta")
	if len(got) != 3 || got[0] != "auth" || got[1] != "prod" || got[2] != "beta" {
		t.Fatalf("tableBadges = %q", got)
	}
}

func TestTableActionsOnHover(t *testing.T) {
	tbl := NewTable([]TableColumn{
		{ID: "key", Label: "Key", Kind: TableColEditable},
		{ID: "act", Kind: TableColActions, Width: 40, Locked: true},
	}, []TableAction{{Icon: icons.Trash2}})
	tbl.ActionsOnHover = true
	tbl.SetRows([]TableRow{{ID: "r1"}, {ID: "r2"}})
	tbl.host.Style = tbl.host.Style.W(400).H(160)
	tbl.host.Calculate(400, 160)
	if tbl.actionsShown(0, tbl.Rows[0]) {
		t.Fatal("actions shown with no hover")
	}
	var deleted []string
	tbl.Actions[0].OnClick = func(id string) { deleted = append(deleted, id) }
	clickAction(tbl, 1, 1, 0) // the click hovers the row first, so it lands
	if len(deleted) != 1 || deleted[0] != "r2" {
		t.Fatalf("deleted %v, want r2", deleted)
	}
	if !tbl.actionsShown(1, tbl.Rows[1]) || tbl.actionsShown(0, tbl.Rows[0]) {
		t.Fatal("only the hovered row shows its actions")
	}
	cw, _, _ := tbl.bodyMetrics()
	if w, _ := tbl.columnLayout(cw); w[1] != 40 {
		t.Fatalf("actions column width %v, want 40 kept", w[1])
	}
}

func TestTableCellView(t *testing.T) {
	c := newTestCtx(t)
	tbl := NewTable([]TableColumn{
		{ID: "key", Label: "Key", Kind: TableColText},
		{ID: "status", Label: "Status", Kind: TableColText, Width: 120},
	}, nil)
	tbl.SetRows([]TableRow{{ID: "r1", Cells: map[string]string{"key": "a"}}, {ID: "r2", Cells: map[string]string{"key": "b"}}})
	clicked := ""
	tbl.CellView = func(row TableRow, col TableColumn) View {
		if col.ID != "status" {
			return nil
		}
		return Button("st-"+row.ID, Text("Run")).OnClick(func() { clicked = row.ID })
	}
	m := &input.Mouse{X: -1, Y: -1}
	var root *layout.Element
	for range 2 {
		root = BuildFrame(c, func(*Ctx) View { return Column(ViewOf(tbl).Grow(1)).Grow(1) }, 500, 300, m, &input.Keyboard{})
		c.EndFrame()
	}
	if n := len(tbl.cellLayer.Children); n != 2 {
		t.Fatalf("%d cell views, want 2", n)
	}
	btn := tbl.cellLayer.Children[1].Children[0]
	m.X, m.Y = btn.Frame.X+btn.Frame.W/2, btn.Frame.Y+btn.Frame.H/2
	m.Pressed, m.Down = true, true
	layout.Dispatch(root, m)
	m.Pressed, m.Down, m.Released, m.Consumed = false, false, true, false
	layout.Dispatch(root, m)
	if clicked != "r2" {
		t.Fatalf("clicked %q, want r2's button", clicked)
	}
	// The second row's view sits in the second row.
	if want := tbl.host.Frame.Y + tbl.headerH + tbl.rowH; tbl.cellLayer.Children[1].Frame.Y != want {
		t.Fatalf("view at y %v, want %v", tbl.cellLayer.Children[1].Frame.Y, want)
	}
}
