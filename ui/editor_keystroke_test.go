package ui

import (
	"testing"

	"github.com/mirzakhany/yoga/highlight"
	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
)

// BenchmarkEditorKeystroke measures a full typed character end to end: the
// piece-table insert, the highlighter hand-off, and a laid-out, painted frame.
func BenchmarkEditorKeystroke(b *testing.B) {
	orig := highlight.MaxBytes
	highlight.MaxBytes = 1 << 30
	defer func() { highlight.MaxBytes = orig }()

	text, err := shape.NewEngine(1, false)
	if err != nil {
		b.Skip(err)
	}
	SetFrameResources(text, render.NewSpriteSheet(text.Atlas), &input.MemClipboard{})

	for _, s := range []struct {
		label string
		lines int
	}{{"1k-lines", 1000}, {"50k-lines", 50000}} {
		b.Run(s.label, func(b *testing.B) {
			c := New(text, NewFocusScope(), nil)
			c.SetIcons(render.NewSpriteSheet(text.Atlas))
			c.SetClipboard(&input.MemClipboard{})

			ed := NewEditor(bigJSONDoc(s.lines), highlight.NewJSON())
			defer ed.Close()
			body := func(cc *Ctx) View { return Column(ViewOf(ed).Grow(1)).Grow(1) }
			// One DrawList reused across frames, as the runtime does.
			var dl render.DrawList
			frame := func() {
				root := BuildFrame(c, body, 1000, 600, &input.Mouse{}, &input.Keyboard{})
				dl.Reset()
				layout.Paint(root, &dl, text)
				c.EndFrame()
			}
			// Settle layout and the first highlight.
			for i := 0; i < 3; i++ {
				frame()
			}

			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				ed.HandleText([]rune{'x'})
				frame()
			}
		})
	}
}
