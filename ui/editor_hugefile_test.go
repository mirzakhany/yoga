package ui

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/mirzakhany/yoga/highlight"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
)

// hugeLineContent builds a document with a single multi-megabyte line, the
// pathological case that must not freeze the editor (minified XML/JSON files).
func hugeLineContent(n int) []byte {
	b := strings.Repeat("<record><field>value</field></record>", n)
	return []byte("<root>" + b + "</root>")
}

func newTestEditor(t *testing.T, content []byte) *Editor {
	t.Helper()
	text, err := shape.NewEngine(1, false)
	if err != nil {
		t.Fatal(err)
	}
	SetFrameResources(text, render.NewSpriteSheet(text.Atlas), nil)
	e := NewEditor(content, highlight.Noop{})
	t.Cleanup(func() { e.Close() })
	return e
}

// TestEditorHugeLineOpensFast verifies that opening a document with one huge
// line does not shape the whole line (which used to take minutes) and that
// painting a frame emits a bounded amount of geometry.
func TestEditorHugeLineOpensFast(t *testing.T) {
	content := hugeLineContent(170000) // ~6.1MB, single line

	t0 := time.Now()
	e := newTestEditor(t, content)
	e.Update(nil)
	open := time.Since(t0)
	if open > 2*time.Second {
		t.Fatalf("opening a 6MB single-line doc took %v; whole-line shaping is back", open)
	}

	// Simulate the paint loop: build the frame, lay it out, paint it.
	c := New(frameText(), NewFocusScope(), nil)
	c.BeginFrame(900, 600, nil, nil)
	host := e.Layout(c)
	root := layout.New(layout.Box(), host)
	root.Calculate(900, 600)

	dl := &render.DrawList{}
	t1 := time.Now()
	layout.Paint(root, dl, frameText())
	paint := time.Since(t1)

	// 4 verts/quad; a frame drawing the whole 6MB line would emit millions.
	maxVerts := 20000
	if len(dl.Vertices) > maxVerts {
		t.Fatalf("paint emitted %d vertices, want <= %d (unwindowed glyph draw)", len(dl.Vertices), maxVerts)
	}
	if paint > time.Second {
		t.Fatalf("painting the huge line took %v", paint)
	}

	// Caret geometry on the huge line must be mono-arithmetic, not shaping.
	t2 := time.Now()
	e.caret = e.pt.Len()
	_ = e.caretXInLine(0, e.caret)
	off := e.offsetAtPoint(400, 12)
	if time.Since(t2) > time.Second {
		t.Fatalf("caret geometry on huge line took %v", time.Since(t2))
	}
	if off <= 0 || off >= e.pt.Len() {
		t.Fatalf("offsetAtPoint on huge line = %d, want mid-document", off)
	}
	_ = open
}

// TestEditorWindowedPaintTracksScroll verifies that scrolling horizontally
// through a huge line shapes different windows (text actually moves) without
// ever shaping the whole line.
func TestEditorWindowedPaintTracksScroll(t *testing.T) {
	content := hugeLineContent(1000) // ~36KB single line
	e := newTestEditor(t, content)
	e.Update(nil)

	if !e.isHuge(0) {
		t.Fatalf("line of %d bytes not treated as huge (threshold %d)", e.pt.LineLen(0), maxShapedLine)
	}

	c := New(frameText(), NewFocusScope(), nil)
	c.BeginFrame(900, 600, nil, nil)
	e.Layout(c)
	root := layout.New(layout.Box(), e.host)
	root.Calculate(900, 600)

	var lastDL *render.DrawList
	paintOnce := func() int {
		dl := &render.DrawList{}
		e.paint(dl, frameText())
		lastDL = dl
		return len(dl.Vertices)
	}

	n1 := paintOnce()
	// Scroll deep into the line and paint again.
	e.ScrollX = e.ContentWidth / 2
	n2 := paintOnce()

	if n1 == 0 || n2 == 0 {
		t.Fatalf("paint emitted no geometry (n1=%d n2=%d)", n1, n2)
	}
	const cap = 40000
	if n1 > cap || n2 > cap {
		t.Fatalf("paint vertices n1=%d n2=%d exceed cap %d", n1, n2, cap)
	}
	// The visible window must land on-screen despite the horizontal scroll.
	_, maxX := glyphXRange(lastDL)
	if maxX < 200 {
		t.Fatalf("scrolled windowed paint max x = %.1f; text never reaches the viewport", maxX)
	}
}

// glyphXRange returns the horizontal extent of emitted quad vertices.
func glyphXRange(dl *render.DrawList) (minX, maxX float32) {
	minX, maxX = 1e9, -1e9
	for _, v := range dl.Vertices {
		if v.Pos[0] < minX {
			minX = v.Pos[0]
		}
		if v.Pos[0] > maxX {
			maxX = v.Pos[0]
		}
	}
	return
}

// TestEditorSoftWrapTable verifies wrap-point computation, including tab
// stops and word-boundary breaking, and that content size reflects the
// wrapped rows.
func TestEditorSoftWrapTable(t *testing.T) {
	// Long single line: must wrap into multiple rows (~118 cols fit the 900px
	// test viewport).
	content := []byte(strings.Repeat("word ", 100)) // 500 bytes, one line
	e := newTestEditor(t, content)
	e.SetSoftWrap(true)
	c := New(frameText(), NewFocusScope(), nil)
	c.BeginFrame(900, 600, nil, nil)
	e.Layout(c)
	root := layout.New(layout.Box(), e.host)
	root.Calculate(900, 600)
	e.Update(nil) // rebuild wrap with the laid-out viewport

	if !e.SoftWrap {
		t.Fatal("SoftWrap not enabled")
	}
	rows := e.lineRows[0]
	if len(rows) < 2 {
		t.Fatalf("expected multiple rows for a long line, got %d", len(rows))
	}
	view := e.lineView(0)
	// Rows must tile the line: contiguous, ending at line end, never wider
	// than the row width, and (with word wrap on) breaking after spaces.
	for i, row := range rows {
		if i+1 < len(rows) && row.end != rows[i+1].start {
			t.Fatalf("row %d end %d != next start %d", i, row.end, rows[i+1].start)
		}
		if w := e.monoLineWidth(view[row.start:row.end]); w > float32(e.wrapCols+e.tabW)*e.cellW {
			t.Fatalf("row %d spans %.0f cells > cols %d", i, w/e.cellW, e.wrapCols)
		}
		if i+1 < len(rows) && view[row.end-1] != ' ' {
			t.Fatalf("row %d breaks mid-word at %d (%q)", i, row.end, view[row.end-10:row.end+10])
		}
	}
	if got := rows[len(rows)-1].end; int(got) != e.pt.LineLen(0) {
		t.Fatalf("last row end %d != line len %d", got, e.pt.LineLen(0))
	}
	total := e.VisualRowCount()
	wantH := float32(total) * e.lineH
	if e.ContentHeight != wantH {
		t.Fatalf("ContentHeight = %v, want %v", e.ContentHeight, wantH)
	}
	if e.rowPrefix[len(e.rowPrefix)-1] != total {
		t.Fatalf("rowPrefix total %d != VisualRowCount %d", e.rowPrefix[len(e.rowPrefix)-1], total)
	}
}

// TestEditorSoftWrapHugeFile verifies that soft wrap keeps the 6MB single-line
// file fast and bounded: building the wrap table is a linear scan and painting
// stays bounded to the visible rows.
func TestEditorSoftWrapHugeFile(t *testing.T) {
	content := hugeLineContent(170000) // ~6.1MB, single line
	e := newTestEditor(t, content)

	t0 := time.Now()
	e.SetSoftWrap(true)
	c := New(frameText(), NewFocusScope(), nil)
	c.BeginFrame(900, 600, nil, nil)
	e.Layout(c)
	root := layout.New(layout.Box(), e.host)
	root.Calculate(900, 600)
	e.Update(nil)
	open := time.Since(t0)
	if open > 2*time.Second {
		t.Fatalf("wrapping a 6MB line took %v", open)
	}
	if e.VisualRowCount() < 1000 {
		t.Fatalf("6MB line should produce thousands of rows, got %d", e.VisualRowCount())
	}

	// Move the caret deep into the line and paint: geometry must stay cheap
	// AND on-screen — every wrapped row starts at the text origin.
	e.caret = e.pt.Len() / 2
	dl := &render.DrawList{}
	t1 := time.Now()
	e.paint(dl, frameText())
	paint := time.Since(t1)
	const cap = 40000
	if len(dl.Vertices) > cap {
		t.Fatalf("wrapped paint emitted %d vertices > cap %d", len(dl.Vertices), cap)
	}
	if paint > time.Second {
		t.Fatalf("wrapped paint took %v", paint)
	}
	minX, maxX := glyphXRange(dl)
	if len(dl.Vertices) == 0 {
		t.Fatal("wrapped paint emitted no vertices")
	}
	// Some glyph quads must sit at the text origin (just right of the gutter).
	atOrigin := false
	for _, v := range dl.Vertices {
		if v.Pos[0] >= e.gutterW && v.Pos[0] <= e.gutterW+300 {
			atOrigin = true
			break
		}
	}
	if !atOrigin {
		t.Fatalf("no glyphs at the text origin (min x %.1f, max x %.1f) — rows drawn off-screen", minX, maxX)
	}
	if maxX > 900+e.cellW {
		t.Fatalf("glyph max x = %.1f, beyond the 900px viewport (rows drawn off-screen)", maxX)
	}

	// Hit-testing and vertical movement on wrapped rows.
	off := e.offsetAtPoint(400, 12)
	if off <= 0 || off >= e.pt.Len() {
		t.Fatalf("offsetAtPoint = %d", off)
	}
	r0 := e.rowOfByte(e.caret)
	e.moveVertical(1, false)
	if r2 := e.rowOfByte(e.caret); r2 != r0+1 {
		t.Fatalf("moveVertical: row %d -> %d, want %d", r0, r2, r0+1)
	}
}

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}

// checkWrapConsistency verifies the wrap tables tile every line exactly:
// rows contiguous per line, prefix totals matching, ends at line ends.
func checkWrapConsistency(t *testing.T, e *Editor, where string) {
	t.Helper()
	lc := e.pt.LineCount()
	if len(e.lineRows) != lc {
		t.Fatalf("%s: lineRows=%d LineCount=%d", where, len(e.lineRows), lc)
	}
	if len(e.rowPrefix) != lc+1 {
		t.Fatalf("%s: rowPrefix=%d want %d", where, len(e.rowPrefix), lc+1)
	}
	for ln := 0; ln < lc; ln++ {
		rows := e.lineRows[ln]
		ll := e.pt.LineLen(ln)
		if len(rows) == 0 {
			t.Fatalf("%s: line %d has no rows", where, ln)
		}
		if rows[0].start != 0 {
			t.Fatalf("%s: line %d first row starts at %d", where, ln, rows[0].start)
		}
		for i := 0; i+1 < len(rows); i++ {
			if rows[i].end != rows[i+1].start {
				t.Fatalf("%s: line %d row %d end %d != next start %d", where, ln, i, rows[i].end, rows[i+1].start)
			}
		}
		if last := rows[len(rows)-1].end; int(last) != ll {
			t.Fatalf("%s: line %d last row end %d != line len %d", where, ln, last, ll)
		}
		if got := e.rowPrefix[ln+1] - e.rowPrefix[ln]; got != len(rows) {
			t.Fatalf("%s: line %d prefix delta %d != %d rows", where, ln, got, len(rows))
		}
	}
	if total := e.rowPrefix[lc]; total != e.VisualRowCount() {
		t.Fatalf("%s: prefix total %d != VisualRowCount %d", where, total, e.VisualRowCount())
	}
}

// TestEditorIncrementalWrap verifies that edits re-wrap only the touched
// lines and keep the tables consistent: typing within a line stays
// incremental, newline insert/delete rebuilds, and results always match a
// full rebuild.
func TestEditorIncrementalWrap(t *testing.T) {
	var sb strings.Builder
	for i := 0; i < 200; i++ {
		sb.WriteString(strings.Repeat("word ", 40)) // 200-byte line
		sb.WriteString("\n")
	}
	e := newTestEditor(t, []byte(sb.String()))
	e.SetSoftWrap(true)
	c := New(frameText(), NewFocusScope(), nil)
	c.BeginFrame(900, 600, nil, nil)
	e.Layout(c)
	root := layout.New(layout.Box(), e.host)
	root.Calculate(900, 600)
	e.Update(nil)
	checkWrapConsistency(t, e, "initial")

	// 1. Type inside line 100: stays incremental, tables consistent.
	e.caret = e.pt.LineStart(100) + 10
	e.applyEdit(e.caret, 0, "hello ", true)
	if e.wrapFull {
		t.Fatal("same-line edit triggered a full rewrap")
	}
	e.Update(nil)
	checkWrapConsistency(t, e, "after same-line edit")

	// Incremental result must match a fresh full rebuild.
	before := e.VisualRowCount()
	e.wrapFull = true
	e.Update(nil)
	if after := e.VisualRowCount(); after != before {
		t.Fatalf("incremental rows %d != full rebuild rows %d", before, after)
	}
	checkWrapConsistency(t, e, "after full rebuild")

	// 2. Insert newlines: full rebuild path, still consistent.
	e.caret = e.pt.LineStart(100) + 10
	e.applyEdit(e.caret, 0, "\n\n\n", false)
	if !e.wrapFull {
		t.Fatal("newline edit did not request a full rewrap")
	}
	e.Update(nil)
	if got := e.pt.LineCount(); got != 204 { // 200 lines + trailing empty + 3 from "\n\n\n"
		t.Fatalf("LineCount=%d want 204", got)
	}
	checkWrapConsistency(t, e, "after newline insert")

	// 3. Join lines (delete a newline): consistent again.
	e.caret = e.pt.LineStart(101)
	e.applyEdit(e.caret-1, 1, "", false)
	e.Update(nil)
	checkWrapConsistency(t, e, "after line join")

	// 4. Undo twice restores prior shape consistently.
	e.Undo()
	e.Undo()
	e.Update(nil)
	checkWrapConsistency(t, e, "after undo")

	// 5. Editing the huge single line re-wraps only that line.
	huge := newTestEditor(t, []byte(strings.Repeat("<a>b</a>", 100000))) // 800KB, one line
	defer huge.Close()
	huge.SetSoftWrap(true)
	huge.Layout(c)
	root2 := layout.New(layout.Box(), huge.host)
	root2.Calculate(900, 600)
	huge.Update(nil)
	huge.caret = huge.pt.Len() / 2
	t0 := time.Now()
	huge.applyEdit(huge.caret, 0, "X", true)
	huge.Update(nil)
	if d := time.Since(t0); d > 200*time.Millisecond {
		t.Fatalf("single-byte edit in the huge line took %v", d)
	}
	checkWrapConsistency(t, huge, "huge line after edit")
}

// TestEditorWrapWordsUnit tests wrapLineRows directly at controlled widths.
func TestEditorWrapWordsUnit(t *testing.T) {
	e := newTestEditor(t, []byte("x"))
	defer e.Close()
	e.tabW = 4 // pin tab width so expectations are exact

	cases := []struct {
		cols  int
		words bool
		in    string
		want  []string
	}{
		{7, true, "aaa bbb ccc", []string{"aaa bbb ", "ccc"}}, // space swallowed at row end
		{3, true, "aaa bbb", []string{"aaa ", "bbb"}},         // space swallowed to row ends
		{10, true, "short superlongwordtokyo short", []string{"short ", "superlongw", "ordtokyo ", "short"}},
		{5, false, "aaa bbb ccc", []string{"aaa b", "bb cc", "c"}},  // hard breaks, no word logic
		{4, true, "\t\t\tdeep", []string{"\t", "\t", "\t", "deep"}}, // tabs never split mid-advance
	}
	for _, tc := range cases {
		e.wrapCols = tc.cols
		e.WrapWords = tc.words
		got := e.wrapLineRows([]byte(tc.in))
		var parts []string
		for _, r := range got {
			parts = append(parts, tc.in[r.start:r.end])
		}
		if len(parts) != len(tc.want) {
			t.Fatalf("cols=%d words=%v in=%q: got %q want %q", tc.cols, tc.words, tc.in, parts, tc.want)
		}
		for i := range parts {
			if parts[i] != tc.want[i] {
				t.Fatalf("cols=%d words=%v in=%q: row %d = %q want %q", tc.cols, tc.words, tc.in, i, parts[i], tc.want[i])
			}
		}
	}
}

// TestEditorRowOfVisualLines is a regression guard for rowOfVisual's binary
// search: with an upper-mid pivot and hi=mid it never terminated when the
// answer was the last line (the gallery app hung on its very first paint).
func TestEditorRowOfVisualLines(t *testing.T) {
	e := newTestEditor(t, []byte("short one\nx\nlast line here"))
	defer e.Close()
	e.SetSoftWrap(true)
	c := New(frameText(), NewFocusScope(), nil)
	c.BeginFrame(900, 600, nil, nil)
	e.Layout(c)
	root := layout.New(layout.Box(), e.host)
	root.Calculate(900, 600)
	e.Update(nil)

	if lc := e.pt.LineCount(); lc != 3 {
		t.Fatalf("LineCount=%d want 3", lc)
	}
	for r := 0; r < e.VisualRowCount(); r++ {
		ln := e.rowOfVisual(r)
		if ln < 0 || ln >= e.pt.LineCount() {
			t.Fatalf("row %d -> line %d out of range", r, ln)
		}
		if r < e.rowPrefix[ln] || r >= e.rowPrefix[ln+1] {
			t.Fatalf("row %d -> line %d, but line covers rows [%d,%d)", r, ln, e.rowPrefix[ln], e.rowPrefix[ln+1])
		}
	}
	// The last visual row must map to the last line.
	last := e.VisualRowCount() - 1
	if ln := e.rowOfVisual(last); ln != e.pt.LineCount()-1 {
		t.Fatalf("last row %d -> line %d, want %d", last, ln, e.pt.LineCount()-1)
	}
}
