package ui

import (
	"strings"
	"testing"

	"github.com/mirzakhany/yoga/input"
)

const foldGo = `package x

func a() {
	s := "{ not a block"
	if s != "" {
		println(s)
	} else {
		println("}")
	}
}

/*
a comment
*/
var b = []int{
	1,
	2,
}
`

func foldEditor(t *testing.T, doc, path string) *Editor {
	e := newUndoEditor(t, doc)
	e.Path = path
	return e
}

func TestFoldRanges(t *testing.T) {
	e := foldEditor(t, foldGo, "x.go")
	for _, tc := range []struct {
		line, end int
		tail      string
		ok        bool
	}{
		{2, 9, "}", true},    // func a: the brace in the string does not count
		{4, 5, "", true},     // if: its closing line goes on with "} else {"
		{6, 8, "}", true},    // else
		{11, 13, "*/", true}, // block comment
		{14, 17, "}", true},  // composite literal
		{0, 0, "", false},
		{3, 0, "", false},
	} {
		r := e.foldRangeAt(tc.line)
		if r.ok != tc.ok || (tc.ok && (r.end != tc.end || r.tail != tc.tail)) {
			t.Errorf("line %d: got %+v, want end %d tail %q ok %v", tc.line, r, tc.end, tc.tail, tc.ok)
		}
	}
	py := foldEditor(t, "def f():\n    x = 1\n\n    return x\n\ny = 2\n", "x.py")
	if r := py.foldRangeAt(0); !r.ok || r.end != 3 {
		t.Fatalf("indentation fold %+v, want lines 0..3", r)
	}
}

func TestFoldRowsAndMovement(t *testing.T) {
	e := foldEditor(t, foldGo, "x.go")
	lc := e.pt.LineCount()
	if !e.Fold(2) {
		t.Fatal("Fold(2) failed")
	}
	if got, want := e.VisualRowCount(), lc-7; got != want {
		t.Fatalf("rows %d, want %d", got, want)
	}
	if r := e.rowOfByte(e.pt.LineStart(10)); r != 3 {
		t.Fatalf("line 10 on row %d, want 3 (right below the fold)", r)
	}
	// Down from the fold's line lands on the line after it.
	e.SetSelection(e.pt.LineStart(2)+3, e.pt.LineStart(2)+3)
	e.HandleKeys(key(input.KeyDown, 0))
	if ln := e.lineOf(e.Caret()); ln != 10 {
		t.Fatalf("Down went to line %d, want 10", ln)
	}
	e.HandleKeys(key(input.KeyUp, 0))
	if ln := e.lineOf(e.Caret()); ln != 2 {
		t.Fatalf("Up went to line %d, want 2", ln)
	}
	// Right at the end of the fold's line skips the hidden lines.
	end := e.lineEndOf(e.pt.LineStart(2))
	e.SetSelection(end, end)
	e.HandleKeys(key(input.KeyRight, 0))
	if ln := e.lineOf(e.Caret()); ln != 10 {
		t.Fatalf("Right went to line %d, want 10", ln)
	}
	e.HandleKeys(key(input.KeyLeft, 0))
	if e.Caret() != end {
		t.Fatalf("Left went to %d, want the fold line's end %d", e.Caret(), end)
	}
	if !e.Folded(2) {
		t.Fatal("moving around the fold opened it")
	}
	// A selection across the fold copies the hidden text too.
	clip := &input.MemClipboard{}
	SetFrameResources(frameText(), nil, clip)
	e.SetSelection(e.pt.LineStart(2), e.pt.LineStart(10))
	e.Copy()
	if want := strings.Join(strings.Split(foldGo, "\n")[2:10], "\n") + "\n"; clip.Get() != want {
		t.Fatalf("copied %q, want the whole block", clip.Get())
	}
}

func TestFoldsFollowEdits(t *testing.T) {
	e := foldEditor(t, foldGo, "x.go")
	e.Fold(14)
	e.Fold(2)
	// Lines added above move the folds down.
	e.SetSelection(0, 0)
	e.HandleText([]rune("// a\n// b\n"))
	if got := e.Folds(); len(got) != 2 || got[0] != [2]int{4, 11} || got[1] != [2]int{16, 19} {
		t.Fatalf("folds after insert %v", got)
	}
	// Typing on a fold's own line keeps it.
	e.SetSelection(e.pt.LineStart(4)+4, e.pt.LineStart(4)+4)
	e.HandleText([]rune("X"))
	if !e.Folded(4) {
		t.Fatal("typing on the fold's line opened it")
	}
	// An edit inside a fold (here a replace) opens it.
	at := e.pt.LineStart(17)
	e.Replace(at, at+1, "\t3")
	if e.Folded(16) {
		t.Fatal("editing inside the fold left it closed")
	}
	if e.lineHidden(17) {
		t.Fatal("line 17 still hidden")
	}
	e.Undo()
	e.Undo()
	e.Undo()
	if got := string(e.Bytes()); got != foldGo {
		t.Fatalf("undo did not restore the text: %q", got)
	}
}

func TestFoldRevealsCaret(t *testing.T) {
	e := foldEditor(t, foldGo, "x.go")
	e.Fold(2)
	e.openSearch(false)
	typeFind(e, "println")
	if e.Folded(2) {
		t.Fatal("a search match inside the fold did not open it")
	}
	e.closeSearch()
	e.Fold(2)
	e.SetSelection(e.pt.LineStart(5), e.pt.LineStart(5))
	if e.Folded(2) {
		t.Fatal("SetSelection into the fold did not open it")
	}
}

func TestFoldKeys(t *testing.T) {
	defer func(m bool) { editorMac = m }(editorMac)
	editorMac = true
	e := foldEditor(t, foldGo, "x.go")
	e.SetSelection(e.pt.LineStart(5)+2, e.pt.LineStart(5)+2)
	e.HandleKeys(key(input.KeyLeftBracket, input.ModSuper|input.ModAlt))
	if !e.Folded(4) {
		t.Fatalf("⌥⌘[ folded %v, want the if block around the caret", e.Folds())
	}
	if ln := e.lineOf(e.Caret()); ln != 4 {
		t.Fatalf("caret on hidden line %d", ln)
	}
	e.HandleKeys(key(input.KeyRightBracket, input.ModSuper|input.ModAlt))
	if len(e.Folds()) != 0 {
		t.Fatalf("⌥⌘] left %v", e.Folds())
	}
	e.HandleKeys(key(input.KeyK, input.ModSuper))
	e.HandleKeys(key(input.Key0, input.ModSuper))
	if len(e.Folds()) < 4 {
		t.Fatalf("⌘K ⌘0 folded %v", e.Folds())
	}
	if got := string(e.Bytes()); got != foldGo {
		t.Fatal("⌘K ⌘0 typed into the document")
	}
	e.HandleKeys(key(input.KeyK, input.ModSuper))
	e.HandleKeys(key(input.KeyJ, input.ModSuper))
	if len(e.Folds()) != 0 {
		t.Fatalf("⌘K ⌘J left %v", e.Folds())
	}
}

func TestFoldWithSoftWrap(t *testing.T) {
	e := foldEditor(t, foldGo, "x.go")
	e.SetSoftWrap(true)
	e.wrapCols = 20
	e.wrapFull = true
	e.syncWrap()
	before := e.VisualRowCount()
	e.Fold(2)
	checkWrapConsistency(t, e, "folded")
	if e.VisualRowCount() >= before {
		t.Fatal("folding did not remove rows with soft wrap on")
	}
	if ln := e.rowOfVisual(e.rowOfByte(e.pt.LineStart(2)) + 1); ln != 10 {
		t.Fatalf("row after the fold shows line %d, want 10", ln)
	}
	e.HandleText([]rune("x")) // an edit elsewhere keeps the tables right
	e.syncWrap()
	checkWrapConsistency(t, e, "after edit")
}

// TestFoldGutterClicks folds and opens a block with the gutter chevron and
// the placeholder, through real frames.
func TestFoldGutterClicks(t *testing.T) {
	h := newFindHarness(t, foldGo)
	ed := h.ed
	ed.Path = "x.go"
	h.frame(nil, nil)
	f := ed.viewport.Frame
	y := f.Y + 2*ed.lineH + ed.lineH/2
	h.clickMods(f.X+ed.gutterW-foldGutterW/2, y, 0)
	if !ed.Folded(2) {
		t.Fatal("chevron click did not fold line 2")
	}
	h.frame(nil, nil)
	if len(ed.fold.pills) != 1 {
		t.Fatalf("%d placeholders painted, want 1", len(ed.fold.pills))
	}
	p := ed.fold.pills[0].r
	h.clickMods(p.X+p.W/2, p.Y+p.H/2, 0)
	if ed.Folded(2) {
		t.Fatal("placeholder click did not open the fold")
	}
}

func BenchmarkFoldAllLargeFile(b *testing.B) {
	var sb strings.Builder
	for i := 0; i < 2500; i++ {
		sb.WriteString("func f() {\n\tif x {\n\t\ty()\n\t}\n}\n")
	}
	e := newUndoEditor(b, sb.String())
	e.Path = "x.go"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		e.revision++ // drop the cache, as an edit would
		e.FoldAll()
		_ = e.VisualRowCount()
		e.UnfoldAll()
	}
}
