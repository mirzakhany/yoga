package ui

import (
	"strings"
	"testing"
	"time"

	"github.com/mirzakhany/yoga/highlight"
	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
)

// newFindEditor is an editor with a memory clipboard, for find bar tests.
func newFindEditor(t *testing.T, content string) (*Editor, *input.MemClipboard) {
	t.Helper()
	text, err := shape.NewEngine(1, false)
	if err != nil {
		t.Skip(err)
	}
	clip := &input.MemClipboard{}
	SetFrameResources(text, render.NewSpriteSheet(text.Atlas), clip)
	t.Cleanup(func() { SetFrameResources(nil, nil, nil) })
	ed := NewEditor([]byte(content), highlight.Noop{})
	t.Cleanup(ed.Close)
	return ed, clip
}

func typeFind(ed *Editor, s string) { ed.HandleText([]rune(s)) }

// TestEditorSearchPaste checks that Cmd/Ctrl+V with the find bar focused fills
// the query instead of editing the document.
func TestEditorSearchPaste(t *testing.T) {
	ed, clip := newFindEditor(t, "alpha beta\ngamma beta\n")
	ed.openSearch(false)

	clip.Set("beta\nignored")
	ed.HandleKeys([]input.KeyEvent{{Key: input.KeyV, Mods: input.ModSuper}})
	if ed.FindQuery() != "beta" {
		t.Fatalf("query = %q, want %q", ed.FindQuery(), "beta")
	}
	if got := len(ed.search.matches); got != 2 {
		t.Fatalf("matches = %d, want 2", got)
	}
	if got := string(ed.pt.Bytes()); got != "alpha beta\ngamma beta\n" {
		t.Fatalf("document changed: %q", got)
	}

	// The menu path (Editor.Paste) lands in the query too.
	ed.Paste()
	if ed.FindQuery() != "betabeta" {
		t.Fatalf("query after Paste = %q", ed.FindQuery())
	}
}

// TestFindFieldNativeEditing is the select-all bug: with the bar focused,
// ⌘A, shift+arrows, copy, cut and undo act on the find field, and the
// document and its selection are left alone.
func TestFindFieldNativeEditing(t *testing.T) {
	ed, clip := newFindEditor(t, "one two\nthree two\n")
	ed.SetSelection(0, 0)
	ed.openSearch(false)
	typeFind(ed, "two")

	ed.HandleKeys(key(input.KeyA, input.ModSuper))
	q := ed.search.query
	if lo, hi := q.selRange(); lo != 0 || hi != 3 {
		t.Fatalf("⌘A selected %d..%d in the field, want 0..3", lo, hi)
	}
	ed.HandleKeys(key(input.KeyC, input.ModSuper))
	if clip.Get() != "two" {
		t.Fatalf("⌘C copied %q", clip.Get())
	}
	ed.HandleKeys(key(input.KeyRight, 0))
	ed.HandleKeys(key(input.KeyLeft, input.ModShift))
	ed.HandleKeys(key(input.KeyLeft, input.ModShift))
	if lo, hi := q.selRange(); lo != 1 || hi != 3 {
		t.Fatalf("shift+← selected %d..%d, want 1..3", lo, hi)
	}
	ed.HandleKeys(key(input.KeyX, input.ModSuper))
	if q.Value != "t" || clip.Get() != "wo" {
		t.Fatalf("⌘X left %q, clipboard %q", q.Value, clip.Get())
	}
	ed.HandleKeys(key(input.KeyZ, input.ModSuper))
	if q.Value != "two" {
		t.Fatalf("⌘Z in the field gave %q, want two", q.Value)
	}
	if got := string(ed.Bytes()); got != "one two\nthree two\n" {
		t.Fatalf("document changed: %q", got)
	}

	// Editor shortcuts stay off the document while the bar has the keys.
	ed.HandleKeys(key(input.KeyD, input.ModSuper))
	if ed.Cursors() != 1 {
		t.Fatalf("⌘D added cursors from the find field")
	}
	// The public methods an app's menu or commands call follow the bar too.
	ed.SelectAll()
	if lo, hi := q.selRange(); lo != 0 || hi != 3 {
		t.Fatalf("Editor.SelectAll did not select the field: %d..%d", lo, hi)
	}
	if lo, hi := ed.Selection(); hi-lo == ed.pt.Len() {
		t.Fatalf("Editor.SelectAll selected the document")
	}
}

// findHarness drives real frames, so clicks reach the bar as in the app.
type findHarness struct {
	c  *Ctx
	ed *Editor
}

func newFindHarness(t *testing.T, content string) *findHarness {
	c := newTestCtx(t)
	ed := NewEditor([]byte(content), highlight.Noop{})
	t.Cleanup(ed.Close)
	h := &findHarness{c: c, ed: ed}
	h.frame(nil, nil)
	return h
}

func (h *findHarness) frame(m *input.Mouse, kb *input.Keyboard) {
	if m == nil {
		m = &input.Mouse{X: -1, Y: -1}
	}
	if kb == nil {
		kb = &input.Keyboard{}
	}
	body := func(cc *Ctx) View { return Column(ViewOf(h.ed).Grow(1)).Grow(1) }
	root := BuildFrame(h.c, body, 1000, 600, m, kb)
	// Same order as the app loop.
	layout.Dispatch(root, m)
	h.c.Focus().HandleMouse(m)
	h.c.Focus().Route(kb)
	layout.Paint(root, &render.DrawList{}, h.c.Text())
	h.c.EndFrame()
}

func (h *findHarness) click(x, y float32) {
	h.frame(&input.Mouse{X: x, Y: y}, nil)
	h.frame(&input.Mouse{X: x, Y: y, Down: true, Pressed: true}, nil)
	h.frame(&input.Mouse{X: x, Y: y, Released: true}, nil)
}

// TestFindClicks covers focus moving between the text and the bar, and
// double- and triple-click selection inside the find field.
func TestFindClicks(t *testing.T) {
	h := newFindHarness(t, "alpha beta\ngamma beta\n")
	ed := h.ed
	ed.openSearch(true)
	typeFind(ed, "abc def")
	h.frame(nil, nil)

	f := ed.viewport.Frame
	h.click(f.X+80, f.Y+f.H-20)
	if ed.search.focused {
		t.Fatal("text-area click left the find bar focused")
	}

	g := ed.findGeom()
	in := g.r[hitQuery]
	x := ed.search.query.textLeft() + 3 // inside "abc"
	y := in.Y + in.H/2
	h.click(x, y)
	if !ed.search.focused || ed.search.focus != hitQuery {
		t.Fatalf("find input not focused: focused=%v focus=%d", ed.search.focused, ed.search.focus)
	}
	h.click(x, y)
	if lo, hi := ed.search.query.selRange(); lo != 0 || hi != 3 {
		t.Fatalf("double-click selected %d..%d, want the word 0..3", lo, hi)
	}
	h.click(x, y)
	if lo, hi := ed.search.query.selRange(); lo != 0 || hi != 7 {
		t.Fatalf("triple-click selected %d..%d, want all", lo, hi)
	}

	// Click in the replace input, after the multi-click window.
	time.Sleep(doubleClickInterval)
	rin := g.r[hitReplace]
	h.click(rin.X+20, rin.Y+rin.H/2)
	if ed.search.focus != hitReplace {
		t.Fatalf("replace input not focused: focus=%d", ed.search.focus)
	}
	h.frame(nil, &input.Keyboard{Chars: []rune{'r'}})
	if ed.search.repl.Value != "r" {
		t.Fatalf("replace = %q, want r", ed.search.repl.Value)
	}

	// The case toggle inside the find field flips on click, and keeps the
	// keyboard where it was.
	c := g.r[hitCase]
	h.click(c.X+c.W/2, c.Y+c.H/2)
	if !ed.search.caseSens {
		t.Fatal("clicking the case toggle did not turn it on")
	}
	if ed.search.focus != hitReplace {
		t.Fatal("clicking a toggle moved the keyboard focus")
	}
}

func TestFindOptions(t *testing.T) {
	ed, _ := newFindEditor(t, "Foo foo food\nfoo_bar FOO\n")
	ed.openSearch(false)
	typeFind(ed, "foo")
	if _, n := ed.FindMatches(); n != 5 {
		t.Fatalf("case-insensitive matches %d, want 5", n)
	}
	ed.Run(ActionFindToggleCase)
	if _, n := ed.FindMatches(); n != 3 {
		t.Fatalf("match-case matches %d, want 3", n)
	}
	ed.Run(ActionFindToggleCase)
	ed.Run(ActionFindToggleWord)
	if _, n := ed.FindMatches(); n != 3 { // Foo, foo, FOO; not food or foo_bar
		t.Fatalf("whole-word matches %d, want 3", n)
	}
	ed.Run(ActionFindToggleWord)
	ed.search.query.SelectAll()
	typeFind(ed, `fo+d|bar`)
	if _, n := ed.FindMatches(); n != 0 {
		t.Fatalf("regex off: %d matches of a literal", n)
	}
	ed.Run(ActionFindToggleRegex)
	if _, n := ed.FindMatches(); n != 2 {
		t.Fatalf("regex matches %d, want 2", n)
	}
	ed.search.query.SelectAll()
	typeFind(ed, `(`)
	if s, bad := ed.findCountText(); !bad || s != "Invalid regex" {
		t.Fatalf("bad regex counter %q", s)
	}
}

func TestFindCounterAndNavigation(t *testing.T) {
	ed, _ := newFindEditor(t, "x a x b x\n")
	ed.SetSelection(0, 0)
	ed.openSearch(false)
	if s, _ := ed.findCountText(); s != "" {
		t.Fatalf("empty query counter %q", s)
	}
	typeFind(ed, "x")
	if s, _ := ed.findCountText(); s != "1 of 3" {
		t.Fatalf("counter %q, want 1 of 3", s)
	}
	ed.HandleKeys(key(input.KeyEnter, 0))
	if s, _ := ed.findCountText(); s != "2 of 3" {
		t.Fatalf("after Enter %q", s)
	}
	if lo, hi := ed.Selection(); lo != 4 || hi != 5 {
		t.Fatalf("Enter selected %d..%d, want 4..5", lo, hi)
	}
	ed.HandleKeys(key(input.KeyEnter, input.ModShift))
	ed.HandleKeys(key(input.KeyEnter, input.ModShift))
	if s, _ := ed.findCountText(); s != "3 of 3" {
		t.Fatalf("Shift+Enter twice %q, want wrap to 3 of 3", s)
	}
	ed.search.query.SelectAll()
	typeFind(ed, "zz")
	if s, bad := ed.findCountText(); s != "No results" || !bad {
		t.Fatalf("counter %q", s)
	}
}

func TestFindEscapeSelectsMatch(t *testing.T) {
	ed, _ := newFindEditor(t, "aa bb aa\n")
	ed.SetSelection(1, 1)
	ed.openSearch(false)
	typeFind(ed, "aa")
	ed.HandleKeys(key(input.KeyEscape, 0))
	if ed.search.open || ed.FindFocused() {
		t.Fatal("Escape left the bar open")
	}
	if lo, hi := ed.Selection(); lo != 6 || hi != 8 {
		t.Fatalf("selection %d..%d, want the match 6..8", lo, hi)
	}
	// Keys reach the text again.
	ed.HandleText([]rune("Z"))
	if got := string(ed.Bytes()); got != "aa bb Z\n" {
		t.Fatalf("typed after Escape: %q", got)
	}
}

func TestFindPrefill(t *testing.T) {
	ed, _ := newFindEditor(t, "alpha beta\ngamma\n")
	ed.SetSelection(6, 10)
	ed.openSearch(false)
	if ed.FindQuery() != "beta" {
		t.Fatalf("prefill %q", ed.FindQuery())
	}
	if lo, hi := ed.search.query.selRange(); lo != 0 || hi != 4 {
		t.Fatal("prefilled query not selected, so typing would append")
	}
	ed.closeSearch()

	// A multi-line selection becomes the search range instead.
	ed.SetSelection(0, 14)
	ed.openSearch(false)
	if ed.FindQuery() != "beta" || !ed.search.inSel {
		t.Fatalf("multi-line selection: query %q inSel %v", ed.FindQuery(), ed.search.inSel)
	}
}

func TestFindInSelection(t *testing.T) {
	ed, _ := newFindEditor(t, "a a\na a\na a\n")
	ed.SetSelection(4, 7) // the middle line
	ed.Run(ActionFind)
	ed.Run(ActionFindToggleInSel)
	typeFind(ed, "a")
	if _, n := ed.FindMatches(); n != 2 {
		t.Fatalf("in-selection matches %d, want 2", n)
	}
	ed.Run(ActionFindToggleInSel)
	if _, n := ed.FindMatches(); n != 6 {
		t.Fatalf("whole-document matches %d, want 6", n)
	}
}

func TestReplaceOneAndAll(t *testing.T) {
	ed, _ := newFindEditor(t, "cat cat cat\n")
	ed.SetSelection(0, 0)
	ed.openSearch(true)
	typeFind(ed, "cat")
	ed.focusFind(hitReplace, true)
	typeFind(ed, "dog")
	ed.HandleKeys(key(input.KeyEnter, 0))
	if got := string(ed.Bytes()); got != "dog cat cat\n" {
		t.Fatalf("replace one: %q", got)
	}
	if lo, hi := ed.Selection(); lo != 4 || hi != 7 {
		t.Fatalf("replace moved to %d..%d, want the next match 4..7", lo, hi)
	}
	ed.HandleKeys(key(input.KeyEnter, input.ModSuper))
	if got := string(ed.Bytes()); got != "dog dog dog\n" {
		t.Fatalf("replace all: %q", got)
	}
	ed.search.focused = false
	ed.Undo()
	if got := string(ed.Bytes()); got != "dog cat cat\n" {
		t.Fatalf("one undo after Replace All gave %q", got)
	}
}

func TestReplaceRegexGroupsAndCase(t *testing.T) {
	ed, _ := newFindEditor(t, "key=val other=thing\n")
	ed.openSearch(true)
	ed.Run(ActionFindToggleRegex)
	typeFind(ed, `(\w+)=(\w+)`)
	ed.focusFind(hitReplace, true)
	typeFind(ed, `$2x:$1`)
	ed.Run(ActionReplaceAll)
	if got := string(ed.Bytes()); got != "valx:key thingx:other\n" {
		t.Fatalf("regex replace all: %q", got)
	}

	ed2, _ := newFindEditor(t, "Hello hello HELLO\n")
	ed2.openSearch(true)
	typeFind(ed2, "hello")
	ed2.focusFind(hitReplace, true)
	typeFind(ed2, "bye")
	ed2.Run(ActionFindTogglePreserve)
	ed2.Run(ActionReplaceAll)
	if got := string(ed2.Bytes()); got != "Bye bye BYE\n" {
		t.Fatalf("preserve case: %q", got)
	}
}

// TestFindTabOrder walks the bar with Tab and activates a button with Space.
func TestFindTabOrder(t *testing.T) {
	ed, _ := newFindEditor(t, "abc\n")
	ed.openSearch(true)
	ed.HandleKeys(key(input.KeyTab, 0))
	if ed.search.focus != hitCase {
		t.Fatalf("Tab from the query went to %d, want the case toggle", ed.search.focus)
	}
	ed.HandleKeys(key(input.KeySpace, 0))
	if !ed.search.caseSens {
		t.Fatal("Space did not press the focused toggle")
	}
	ed.HandleKeys(key(input.KeyTab, input.ModShift))
	ed.HandleKeys(key(input.KeyTab, input.ModShift))
	if ed.search.focus != hitChevron {
		t.Fatalf("Shift+Tab twice went to %d, want the chevron", ed.search.focus)
	}
}

// TestFindDebouncesLargeDocuments checks that typing into the find field of
// a big document defers the search, and that it runs once due.
func TestFindDebouncesLargeDocuments(t *testing.T) {
	var b strings.Builder
	for i := 0; i < 20000; i++ {
		b.WriteString("the quick brown fox jumps\n")
	}
	ed, _ := newFindEditor(t, b.String())
	ed.openSearch(false)
	typeFind(ed, "fox")
	if !ed.search.pending {
		t.Fatal("search ran on every keystroke of a large document")
	}
	if _, ok := ed.findWake(); !ok {
		t.Fatal("no wake scheduled for the pending search")
	}
	ed.search.due = time.Now().Add(-time.Millisecond)
	ed.findUpdate()
	if s, _ := ed.findCountText(); s != "1 of 10000+" {
		t.Fatalf("counter %q, want the capped 1 of 10000+", s)
	}
	// Replace All still reaches every match past the cap.
	ed.openSearch(true)
	ed.focusFind(hitReplace, true)
	typeFind(ed, "cat")
	ed.Run(ActionReplaceAll)
	if n := strings.Count(string(ed.Bytes()), "cat"); n != 20000 {
		t.Fatalf("replaced %d, want 20000", n)
	}
}

func TestFindKeymapPerPlatform(t *testing.T) {
	defer func(m bool) { editorMac = m }(editorMac)
	for _, mac := range []bool{true, false} {
		editorMac = mac
		ed, _ := newFindEditor(t, "abc\n")
		ev := input.KeyEvent{Key: input.KeyH, Mods: input.ModCtrl}
		wantLabel := "Alt+C"
		if mac {
			ev = input.KeyEvent{Key: input.KeyF, Mods: input.ModSuper | input.ModAlt}
			wantLabel = "⌥⌘C"
		}
		ed.HandleKeys([]input.KeyEvent{ev})
		if !ed.search.open || !ed.search.replaceMode {
			t.Fatalf("mac=%v: replace shortcut did not open replace", mac)
		}
		if got := ed.ShortcutLabel(ActionFindToggleCase); got != wantLabel {
			t.Fatalf("mac=%v: case toggle label %q, want %q", mac, got, wantLabel)
		}
	}
}

func TestEditorSetKeymapOverrides(t *testing.T) {
	ed, _ := newFindEditor(t, "abc\n")
	km := DefaultEditorKeymap()
	for i := range km {
		if km[i].Action == ActionFind && km[i].When == ScopeText {
			km[i].Mac, km[i].Other = "F4", "F4"
		}
	}
	ed.SetKeymap(km)
	ed.HandleKeys(key(input.KeyF, input.ModSuper))
	if ed.search.open {
		t.Fatal("old binding still opens find")
	}
	ed.HandleKeys(key(input.KeyF4, 0))
	if !ed.search.open {
		t.Fatal("new binding did not open find")
	}
	if ed.Run("no.such.action") {
		t.Fatal("Run accepted an unknown action")
	}
	tbl := EditorKeymapTable(nil)
	if len(tbl) == 0 || tbl[0].Title == "" || tbl[0].Mac == "" || tbl[0].Other == "" {
		t.Fatalf("keymap table row %+v", tbl[0])
	}
}
