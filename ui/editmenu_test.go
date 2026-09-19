package ui

import (
	"strings"
	"testing"

	"github.com/mirzakhany/yoga/highlight"
	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
)

// menuHarness drives frames the way the app loop does: build, dispatch the
// mouse, grant click focus, then route keys.
type menuHarness struct {
	t     *testing.T
	c     *Ctx
	focus *FocusScope
	clip  *input.MemClipboard
	mouse input.Mouse
	kb    input.Keyboard
	body  func(*Ctx) View
	root  *layout.Element
}

func newMenuHarness(t *testing.T, body func(*Ctx) View) *menuHarness {
	t.Helper()
	eng, err := shape.NewEngine(1, false)
	if err != nil {
		t.Skip(err)
	}
	h := &menuHarness{t: t, focus: NewFocusScope(), clip: &input.MemClipboard{}, body: body}
	h.c = New(eng, h.focus, nil)
	h.c.SetIcons(render.NewSpriteSheet(eng.Atlas))
	h.c.SetClipboard(h.clip)
	t.Cleanup(func() { SetFrameResources(nil, nil, nil) })
	h.step()
	return h
}

func (h *menuHarness) step() {
	h.root = BuildFrame(h.c, h.body, 600, 600, &h.mouse, &h.kb)
	h.c.BeginInputPhase()
	layout.Dispatch(h.root, &h.mouse)
	h.focus.HandleMouse(&h.mouse)
	h.focus.Route(&h.kb)
	h.mouse.EndFrame()
	h.kb.EndFrame()
	h.c.EndInputPhase()
	// Settle: rebuild so overlays opened during dispatch are in the tree.
	h.root = BuildFrame(h.c, h.body, 600, 600, &h.mouse, &h.kb)
}

func (h *menuHarness) rightClick(x, y float32) {
	h.mouse.SetPos(x, y)
	h.mouse.SetRightButton(true)
	h.step()
	h.mouse.SetRightButton(false)
	h.step()
}

func (h *menuHarness) click(x, y float32) {
	h.mouse.SetPos(x, y)
	h.mouse.SetButton(true)
	h.step()
	h.mouse.SetButton(false)
	h.step()
}

func (h *menuHarness) press(k input.Key) {
	h.kb.PressKey(k, 0)
	h.step()
}

// clickItem clicks the menu row labelled label.
func (h *menuHarness) clickItem(mu *Menu, label string) {
	h.t.Helper()
	x, y, ok := menuItemCenter(mu, label)
	if !ok {
		h.t.Fatalf("menu has no %q item: %v", label, menuLabels(mu))
	}
	h.click(x, y)
}

func menuItemCenter(mu *Menu, label string) (float32, float32, bool) {
	f := mu.host.Frame
	y := f.Y
	for i, it := range mu.items {
		rh := mu.rowHeight(i)
		if !it.Separator && it.Label == label {
			return f.X + f.W/2, y + rh/2, true
		}
		y += rh
	}
	return 0, 0, false
}

func menuLabels(mu *Menu) string {
	var parts []string
	for _, it := range mu.items {
		switch {
		case it.Separator:
			parts = append(parts, "|")
		case it.Disabled:
			parts = append(parts, "("+it.Label+")")
		default:
			parts = append(parts, it.Label)
		}
	}
	return strings.Join(parts, " ")
}

func center(el *layout.Element) (float32, float32) {
	return el.Frame.X + el.Frame.W/2, el.Frame.Y + el.Frame.H/2
}

func TestTextFieldContextMenu(t *testing.T) {
	a, b := "hello world", "other"
	h := newMenuHarness(t, func(c *Ctx) View {
		return Column(
			TextField("a", a).OnChange(func(s string) { a = s }).Width(300),
			TextField("b", b).OnChange(func(s string) { b = s }).Width(300),
		)
	})
	fa := h.c.Widget("a", nil).(*TextInput)
	fb := h.c.Widget("b", nil).(*TextInput)

	h.clip.Set("")
	h.rightClick(center(fa.host))
	if !fa.menu.isOpen() {
		t.Fatal("right-click did not open the menu")
	}
	if h.focus.Current() != fa {
		t.Fatal("right-click did not focus the field")
	}
	if got, want := menuLabels(fa.menu.menu), "(Undo) (Redo) | (Cut) (Copy) (Paste) | Select All"; got != want {
		t.Fatalf("items = %s, want %s", got, want)
	}

	// Redo is disabled and sits over field b: the click keeps the menu open
	// and must not move focus to b.
	x, y, _ := menuItemCenter(fa.menu.menu, "Redo")
	if !fb.host.Frame.Contains(x, y) {
		t.Fatalf("test layout: Redo row (%v,%v) is not over field b %v", x, y, fb.host.Frame)
	}
	h.click(x, y)
	if !fa.menu.isOpen() || h.focus.Current() != fa {
		t.Fatalf("disabled item: open=%v focus=%v", fa.menu.isOpen(), h.focus.Current() == fa)
	}

	h.clickItem(fa.menu.menu, "Select All")
	if fa.menu.isOpen() || !fa.HasSelection() || h.focus.Current() != fa {
		t.Fatalf("Select All: open=%v selected=%v", fa.menu.isOpen(), fa.HasSelection())
	}

	h.rightClick(center(fa.host)) // inside the selection: it stays
	h.clickItem(fa.menu.menu, "Cut")
	if a != "" || h.clip.Get() != "hello world" {
		t.Fatalf("Cut: value=%q clip=%q", a, h.clip.Get())
	}

	h.rightClick(center(fa.host))
	if got, want := menuLabels(fa.menu.menu), "Undo (Redo) | (Cut) (Copy) Paste | Select All"; got != want {
		t.Fatalf("after cut: items = %s, want %s", got, want)
	}
	h.clickItem(fa.menu.menu, "Paste")
	h.rightClick(center(fa.host))
	h.clickItem(fa.menu.menu, "Undo")
	if a != "" {
		t.Fatalf("Undo after paste: value=%q", a)
	}

	// Escape closes the menu without reaching the field.
	h.rightClick(center(fa.host))
	h.press(input.KeyEscape)
	if fa.menu.isOpen() {
		t.Fatal("Escape did not close the menu")
	}
}

func TestTextFieldContextMenuMovesCaretOutsideSelection(t *testing.T) {
	v := "hello world"
	h := newMenuHarness(t, func(c *Ctx) View {
		return TextField("f", v).OnChange(func(s string) { v = s }).Width(300)
	})
	f := h.c.Widget("f", nil).(*TextInput)
	f.selAnchor, f.caret = 0, 2 // "he"
	_, y := center(f.host)
	h.rightClick(f.host.Frame.X+f.host.Frame.W-4, y) // past the text
	if f.HasSelection() || f.caret != len(v) {
		t.Fatalf("caret=%d selected=%v, want caret at end", f.caret, f.HasSelection())
	}
}

func TestPasswordFieldMenuHidesText(t *testing.T) {
	h := newMenuHarness(t, func(c *Ctx) View {
		return TextField("p", "secret").Password(true).Width(300)
	})
	f := h.c.Widget("p", nil).(*TextInput)
	f.SelectAll()
	h.rightClick(center(f.host))
	if got := menuLabels(f.menu.menu); !strings.Contains(got, "(Cut) (Copy)") {
		t.Fatalf("password menu offers copying: %s", got)
	}
}

func TestTableCellContextMenu(t *testing.T) {
	tbl := NewTable([]TableColumn{
		{ID: "k", Label: "Key", Kind: TableColEditable},
		{ID: "v", Label: "Value", Kind: TableColEditable},
	}, nil)
	tbl.SetRows([]TableRow{{ID: "r", Cells: map[string]string{"k": "token", "v": "abc"}}})
	h := newMenuHarness(t, func(c *Ctx) View { return ViewOf(tbl).Width(400).Height(200) })

	cw, _, _ := tbl.bodyMetrics()
	widths, offsets := tbl.columnLayout(cw)
	y := tbl.host.Frame.Y + tbl.headerH + tbl.rowH/2
	cr := tbl.cellRect(tbl.host.Frame.Y+tbl.headerH, 1, widths, offsets)
	x := cr.X + cr.W/2

	h.rightClick(x, y)
	if tbl.editingRowID != "r" || tbl.editingColID != "v" {
		t.Fatalf("right-click did not start editing: %q/%q", tbl.editingRowID, tbl.editingColID)
	}
	mu := tbl.editField.menu.menu
	if mu == nil || !mu.Open {
		t.Fatal("cell menu did not open")
	}
	h.clickItem(mu, "Copy")
	if h.clip.Get() != "abc" {
		t.Fatalf("Copy: clip=%q", h.clip.Get())
	}

	h.clip.Set("xyz")
	h.rightClick(x, y) // now over the edit field itself
	h.clickItem(tbl.editField.menu.menu, "Select All")
	h.rightClick(x, y)
	h.clickItem(tbl.editField.menu.menu, "Paste")
	tbl.CommitCellEdit()
	if got := tbl.Rows[0].Cells["v"]; got != "xyz" {
		t.Fatalf("pasted cell = %q", got)
	}
}

func TestEditableLabelContextMenu(t *testing.T) {
	var saved string
	h := newMenuHarness(t, func(c *Ctx) View {
		return EditableLabel("l", "My request").OnSubmit(func(s string) { saved = s })
	})
	st := h.c.Widget("l", nil).(*editableLabelState)
	h.rightClick(center(st.el))
	if !st.editing || !st.field.menu.isOpen() || !st.field.HasSelection() {
		t.Fatalf("editing=%v open=%v selected=%v", st.editing, st.field.menu.isOpen(), st.field.HasSelection())
	}
	h.press(input.KeyEscape) // closes the menu only
	if !st.editing || st.field.menu.isOpen() {
		t.Fatalf("Escape: editing=%v open=%v", st.editing, st.field.menu.isOpen())
	}
	h.clip.Set("Renamed")
	h.rightClick(center(st.el))
	h.clickItem(st.field.menu.menu, "Paste")
	h.press(input.KeyEnter)
	if saved != "Renamed" {
		t.Fatalf("saved = %q", saved)
	}
}

func TestEditorContextMenu(t *testing.T) {
	var ed *Editor
	h := newMenuHarness(t, func(c *Ctx) View {
		if ed == nil {
			ed = NewEditor([]byte("hello world"), highlight.Noop{}, WithoutGutter())
			t.Cleanup(ed.Close)
		}
		return ViewOf(ed).Width(400).Height(300)
	})
	vp := ed.viewport.Frame
	h.rightClick(vp.X+20, vp.Y+10)
	if !ed.menu.isOpen() || h.focus.Current() != ed {
		t.Fatalf("open=%v focused=%v", ed.menu.isOpen(), h.focus.Current() == ed)
	}
	h.clickItem(ed.menu.menu, "Select All")
	h.rightClick(vp.X+20, vp.Y+10)
	h.clickItem(ed.menu.menu, "Cut")
	if got := string(ed.Bytes()); got != "" || h.clip.Get() != "hello world" {
		t.Fatalf("Cut: content=%q clip=%q", got, h.clip.Get())
	}
	h.rightClick(vp.X+20, vp.Y+10)
	h.clickItem(ed.menu.menu, "Undo")
	if got := string(ed.Bytes()); got != "hello world" {
		t.Fatalf("Undo: content=%q", got)
	}
}

func TestEditorContextMenuHookAndReadOnly(t *testing.T) {
	var ed *Editor
	saved := false
	h := newMenuHarness(t, func(c *Ctx) View {
		if ed == nil {
			ed = NewEditor([]byte("body"), highlight.Noop{}, WithReadOnly(), WithoutGutter())
			ed.ContextMenu = func(items []MenuItem) []MenuItem {
				return append(items, MenuItem{Label: "Save…", OnSelect: func() { saved = true }})
			}
			t.Cleanup(ed.Close)
		}
		return ViewOf(ed).Width(400).Height(300)
	})
	vp := ed.viewport.Frame
	h.rightClick(vp.X+20, vp.Y+10)
	if got, want := menuLabels(ed.menu.menu), "(Copy) | Select All Save…"; got != want {
		t.Fatalf("items = %s, want %s", got, want)
	}
	h.clickItem(ed.menu.menu, "Save…")
	if !saved {
		t.Fatal("hook item did not run")
	}

	// A hook returning nothing turns the menu off and leaves the click
	// unconsumed for a wrapper behind the editor.
	ed.ContextMenu = func([]MenuItem) []MenuItem { return nil }
	h.mouse.SetPos(vp.X+20, vp.Y+10)
	h.mouse.SetRightButton(true)
	h.root = BuildFrame(h.c, h.body, 600, 600, &h.mouse, &h.kb)
	layout.Dispatch(h.root, &h.mouse)
	if h.mouse.Consumed || ed.menu.isOpen() {
		t.Fatalf("empty hook: consumed=%v open=%v", h.mouse.Consumed, ed.menu.isOpen())
	}
}
