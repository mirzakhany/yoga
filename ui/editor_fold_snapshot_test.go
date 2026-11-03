package ui

import (
	"image"
	"testing"

	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
	"github.com/mirzakhany/yoga/theme"
)

// TestFoldSnapshot renders folded code with the pointer over the editor, so
// chevrons show, when YOGA_SNAPSHOT_DIR is set.
func TestFoldSnapshot(t *testing.T) {
	const scale = 2
	text, err := shape.NewEngine(scale, false)
	if err != nil {
		t.Skip(err)
	}
	prev := theme.Current().Name
	theme.Use("tokyo-night")
	defer theme.Use(prev)
	theme.SetReducedMotion(true) // the settled layout, not a frame of the slide
	defer theme.SetReducedMotion(false)
	sheet := render.NewSpriteSheet(text.Atlas)
	SetFrameResources(text, sheet, &input.MemClipboard{})
	c := New(text, NewFocusScope(), nil)
	c.SetIcons(sheet)
	ed := NewEditorFor("x.go", []byte(foldGo))
	defer ed.Close()
	body := func(cc *Ctx) View { return Column(ViewOf(ed).Grow(1)).Grow(1) }
	const w, h = 520, 300
	frame := func(m *input.Mouse) *render.DrawList {
		root := BuildFrame(c, body, w, h, m, &input.Keyboard{})
		layout.Dispatch(root, m)
		dl := &render.DrawList{}
		layout.Paint(root, dl, text)
		c.EndFrame()
		return dl
	}
	over := &input.Mouse{X: 200, Y: 100}
	frame(over)
	ed.Fold(6)
	ed.Fold(14)
	ed.Fold(11)
	for i := 0; i < 3; i++ {
		frame(over)
	}
	dl := frame(over)
	saveSnapshot(t, "fold.png", rasterize(dl, text, w, h, scale, theme.Current().Background), image.Rectangle{})
}
