package ui

import (
	"strings"
	"testing"

	"github.com/mirzakhany/yoga/highlight"
	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
)

// nestedJSON is a document whose line 5 holds a long string value nested four
// levels deep (8 spaces of indent).
const nestedJSON = `{
  "a": {
    "b": {
      "c": {
        "long": "` + "one two three four five six seven eight nine ten eleven twelve thirteen fourteen fifteen sixteen seventeen eighteen nineteen twenty" + `"
      }
    }
  }
}`

func wrapIndentEditor(t *testing.T, mode WrapIndent) (*Editor, func()) {
	t.Helper()
	c := newTestCtx(t)
	ed := NewEditor([]byte(nestedJSON), highlight.Noop{}, WithSoftWrap(true), WithWrapIndent(mode))
	body := func(*Ctx) View { return Column(ViewOf(ed).Grow(1)).Grow(1) }
	frame := func() {
		root := BuildFrame(c, body, 420, 300, &input.Mouse{X: -1, Y: -1}, &input.Keyboard{})
		layout.Paint(root, &render.DrawList{}, c.Text())
		c.EndFrame()
	}
	frame()
	frame()
	t.Cleanup(ed.Close)
	return ed, frame
}

func TestEditorWrapIndentAlignsContinuations(t *testing.T) {
	for _, tc := range []struct {
		mode WrapIndent
		want int32
	}{
		{WrapIndentNone, 0},
		{WrapIndentSame, 8},
		{WrapHanging, 8 + 4},
	} {
		ed, _ := wrapIndentEditor(t, tc.mode)
		ed.tabW = 4
		ed.wrapFull = true
		ed.syncWrap()
		rows := ed.lineRows[4]
		if len(rows) < 2 {
			t.Fatalf("mode %d: long line did not wrap (%d rows, %d cols)", tc.mode, len(rows), ed.wrapCols)
		}
		if rows[0].indent != 0 {
			t.Fatalf("mode %d: first row indented by %d", tc.mode, rows[0].indent)
		}
		for i, r := range rows[1:] {
			if r.indent != tc.want {
				t.Fatalf("mode %d: continuation %d indent %d, want %d", tc.mode, i+1, r.indent, tc.want)
			}
			if w := int(r.end-r.start) + int(r.indent); w > ed.wrapCols {
				t.Fatalf("mode %d: continuation %d is %d cells wide, row holds %d", tc.mode, i+1, w, ed.wrapCols)
			}
		}
	}
}

func TestEditorWrapIndentCappedAtHalfRow(t *testing.T) {
	ed, _ := wrapIndentEditor(t, WrapHanging)
	if got := ed.wrapIndentCells([]byte(strings.Repeat(" ", 200)+"x"), 40); got != 20 {
		t.Fatalf("indent %d, want it capped at 20 of 40 cells", got)
	}
}

func TestEditorWrapIndentCaretGeometry(t *testing.T) {
	ed, _ := wrapIndentEditor(t, WrapHanging)
	rows := ed.lineRows[4]
	if len(rows) < 2 {
		t.Fatal("long line did not wrap")
	}
	ls := ed.pt.LineStart(4)
	contStart := ls + int(rows[1].start)
	ind := float32(rows[1].indent) * ed.cellW

	// The caret at the start of a continuation row sits at the indent.
	if x := ed.xInRow(contStart); x != ind {
		t.Fatalf("caret x %v at continuation start, want %v", x, ind)
	}

	// A click on that spot lands there; a click in the indent snaps to it.
	f := ed.viewport.Frame
	x0 := f.X + ed.gutterW + ed.textPad - ed.ScrollX
	row := ed.rowOfByte(contStart)
	y := f.Y + float32(row)*ed.lineH - ed.ScrollPx + ed.lineH/2
	if off := ed.offsetAtPoint(x0+ind+ed.cellW*2+1, y); off != contStart+2 {
		t.Fatalf("hit at col 2 of continuation = %d, want %d", off, contStart+2)
	}
	if off := ed.offsetAtPoint(x0+1, y); off != contStart {
		t.Fatalf("hit inside the indent = %d, want the row start %d", off, contStart)
	}

	// Down from the first row keeps the caret's x, and Up returns to it.
	ed.moveTo(ls+int(rows[0].start)+20, false)
	ed.goalX = 0
	ed.moveVertical(1, false)
	if ed.rowOfByte(ed.caret) != row {
		t.Fatalf("down moved to row %d, want %d", ed.rowOfByte(ed.caret), row)
	}
	if want := float32(20) * ed.cellW; ed.xInRow(ed.caret) != want {
		t.Fatalf("down landed at x %v, want %v", ed.xInRow(ed.caret), want)
	}
	ed.moveVertical(-1, false)
	if ed.caret != ls+20 {
		t.Fatalf("up returned to %d, want %d", ed.caret, ls+20)
	}

	// Left from the continuation start goes to the end of the first row.
	ed.moveTo(contStart+1, false)
	ed.HandleKeys([]input.KeyEvent{{Key: input.KeyLeft}})
	if ed.caret != contStart || ed.xInRow(ed.caret) != ind {
		t.Fatalf("left: caret %d at x %v, want %d at %v", ed.caret, ed.xInRow(ed.caret), contStart, ind)
	}
}

func TestEditorWrapToggleKeepsCaretRow(t *testing.T) {
	doc := strings.Repeat("a short line\n", 40) + strings.Repeat("word ", 200) + "\n" + strings.Repeat("tail line\n", 40)
	c := newTestCtx(t)
	ed := NewEditor([]byte(doc), highlight.Noop{})
	defer ed.Close()
	body := func(*Ctx) View { return Column(ViewOf(ed).Grow(1)).Grow(1) }
	frame := func() {
		root := BuildFrame(c, body, 400, 300, &input.Mouse{X: -1, Y: -1}, &input.Keyboard{})
		layout.Paint(root, &render.DrawList{}, c.Text())
		c.EndFrame()
	}
	frame()
	frame()
	// Put the caret on a tail line, below the long one, mid-view.
	ed.moveTo(ed.pt.LineStart(60), false)
	ed.ScrollPx = float32(55) * ed.lineH
	frame()
	before := float32(ed.rowOfByte(ed.caret))*ed.lineH - ed.ScrollPx
	caret := ed.caret

	ed.SetSoftWrap(true)
	frame()
	if ed.caret != caret {
		t.Fatalf("caret moved from %d to %d", caret, ed.caret)
	}
	after := float32(ed.rowOfByte(ed.caret))*ed.lineH - ed.ScrollPx
	if after != before {
		t.Fatalf("caret row at y %v after wrapping, was %v", after, before)
	}
	ed.SetSoftWrap(false)
	frame()
	if back := float32(ed.rowOfByte(ed.caret))*ed.lineH - ed.ScrollPx; back != before {
		t.Fatalf("caret row at y %v after unwrapping, was %v", back, before)
	}
}

func TestEditorGutterSpecSizesToDigits(t *testing.T) {
	c := newTestCtx(t)
	ed := NewEditor([]byte(strings.Repeat("x\n", 8)+"x"), highlight.Noop{},
		WithGutter(GutterSpec{MinDigits: 2, Padding: 6}))
	defer ed.Close()
	body := func(*Ctx) View { return Column(ViewOf(ed).Grow(1)).Grow(1) }
	frame := func() {
		BuildFrame(c, body, 400, 300, &input.Mouse{X: -1, Y: -1}, &input.Keyboard{})
		c.EndFrame()
	}
	frame()
	if want := 2*ed.cellW + 2*6; ed.gutterW != want {
		t.Fatalf("9-line gutter %v, want %v", ed.gutterW, want)
	}
	ed.SetText(strings.Repeat("x\n", 120))
	frame()
	if want := 3*ed.cellW + 2*6; ed.gutterW != want {
		t.Fatalf("121-line gutter %v, want %v", ed.gutterW, want)
	}

	plain := NewEditor([]byte("x"), highlight.Noop{})
	defer plain.Close()
	if plain.gutterW != editorGutterW {
		t.Fatalf("editor without a spec has gutter %v, want %v", plain.gutterW, editorGutterW)
	}
}
