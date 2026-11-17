package ui

import (
	"testing"
	"time"

	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/theme"
)

// TestEditorBlockCaretOnTab verifies that a block caret on a tab covers one
// cell at the tab's end, like Vim, however wide the tab is set.
func TestEditorBlockCaretOnTab(t *testing.T) {
	for _, tw := range []int{2, 4, 8} {
		e := newTestEditor(t, []byte("\tx\n"))
		e.SetTabWidth(tw)
		e.CaretStyle = CaretBlock
		e.Update(nil)

		c := New(frameText(), NewFocusScope(), nil)
		c.BeginFrame(900, 600, nil, nil)
		e.Layout(c)
		root := layout.New(layout.Box(), e.host)
		root.Calculate(900, 600)

		e.focused = true
		caretRect := func(at int) (x, w float32) {
			e.caret = at
			e.blinkStart = time.Now()
			dl := &render.DrawList{}
			e.paint(dl, frameText())
			col := theme.Current().Accent
			col.A *= 0.55
			found := false
			for i := 0; i+3 < len(dl.Vertices); i += 4 {
				if v := dl.Vertices[i]; v.Col == [4]float32{col.R, col.G, col.B, col.A} {
					x, w, found = v.Pos[0], dl.Vertices[i+1].Pos[0]-v.Pos[0], true
				}
			}
			if !found {
				t.Fatalf("tab width %d: no block caret painted at %d", tw, at)
			}
			return x, w
		}

		tabX, tabW := caretRect(0)
		nextX, _ := caretRect(1)
		if tabW != e.cellW {
			t.Fatalf("tab width %d: block caret is %.1fpx wide, want one cell (%.1fpx)", tw, tabW, e.cellW)
		}
		if tabX+tabW != nextX {
			t.Fatalf("tab width %d: block caret ends at x=%.1f, want the tab's end at %.1f", tw, tabX+tabW, nextX)
		}
	}
}
