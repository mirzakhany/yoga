package ui

import (
	"testing"

	"github.com/mirzakhany/yoga/highlight"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
)

func TestEditorSelectionAccess(t *testing.T) {
	text, err := shape.NewEngine(1, false)
	if err != nil {
		t.Skip(err)
	}
	SetFrameResources(text, render.NewSpriteSheet(text.Atlas), nil)
	defer SetFrameResources(nil, nil, nil)

	ed := NewEditor([]byte("one\nhéllo world\nthree"), highlight.Noop{}, WithoutGutter())
	defer ed.Close()

	if lo, hi := ed.Selection(); lo != 0 || hi != 0 || ed.SelectedText() != "" {
		t.Fatalf("fresh editor: Selection() = %d,%d text=%q", lo, hi, ed.SelectedText())
	}

	// "world" on line 1: é is two bytes, so rune col 6 is byte 4+7.
	start := ed.Offset(1, 6)
	end := ed.Offset(1, 11)
	if start != 11 || end != 16 {
		t.Fatalf("Offset = %d,%d, want 11,16", start, end)
	}
	// Selected backwards: Selection still reports start <= end.
	ed.SetSelection(end, start)
	if lo, hi := ed.Selection(); lo != start || hi != end {
		t.Fatalf("Selection() = %d,%d, want %d,%d", lo, hi, start, end)
	}
	if got := ed.SelectedText(); got != "world" {
		t.Fatalf("SelectedText() = %q, want world", got)
	}
	if ed.Caret() != start {
		t.Fatalf("Caret() = %d, want %d", ed.Caret(), start)
	}
	if line, col := ed.LineCol(ed.Caret()); line != 1 || col != 6 {
		t.Fatalf("LineCol = %d,%d, want 1,6", line, col)
	}

	// Equal offsets clear the selection.
	ed.SetSelection(3, 3)
	if ed.HasSelection() || ed.Caret() != 3 {
		t.Fatalf("SetSelection(3,3): HasSelection=%v Caret=%d", ed.HasSelection(), ed.Caret())
	}

	// Out-of-range and mid-rune offsets clamp.
	ed.SetSelection(-5, 1000)
	if got := ed.SelectedText(); got != string(ed.Bytes()) {
		t.Fatalf("clamped SelectedText() = %q", got)
	}
	ed.SetSelection(6, 6) // inside é
	if ed.Caret() != 5 {
		t.Fatalf("mid-rune caret = %d, want 5", ed.Caret())
	}
	if got := ed.Offset(9, -1); got != ed.Offset(2, 0) {
		t.Fatalf("Offset(9,-1) = %d, want start of last line", got)
	}
}
