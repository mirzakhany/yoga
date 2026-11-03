package ui

import (
	"image"
	"testing"

	"github.com/mirzakhany/yoga/highlight"
	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
	"github.com/mirzakhany/yoga/theme"
)

// TestFindBarSnapshot renders the find bar, with and without the replace
// row, to PNGs when YOGA_SNAPSHOT_DIR is set.
func TestFindBarSnapshot(t *testing.T) {
	const scale = 2
	text, err := shape.NewEngine(scale, false)
	if err != nil {
		t.Skip(err)
	}
	prev := theme.Current().Name
	theme.Use("tokyo-night")
	defer theme.Use(prev)
	sheet := render.NewSpriteSheet(text.Atlas)
	SetFrameResources(text, sheet, &input.MemClipboard{})
	c := New(text, NewFocusScope(), nil)
	c.SetIcons(sheet)
	ed := NewEditor([]byte("package main\n\nfunc ok() bool { return true }\nvar b bool\nvar c = bool(false)\n"), highlight.Noop{})
	defer ed.Close()
	body := func(cc *Ctx) View { return Column(ViewOf(ed).Grow(1)).Grow(1) }
	const w, h = 700, 240
	frame := func(m *input.Mouse) *render.DrawList {
		if m == nil {
			m = &input.Mouse{X: -1, Y: -1}
		}
		root := BuildFrame(c, body, w, h, m, &input.Keyboard{})
		layout.Dispatch(root, m)
		dl := &render.DrawList{}
		layout.Paint(root, dl, text)
		c.EndFrame()
		return dl
	}
	frame(nil)
	c.Focus().Focus(ed)
	ed.SetSelection(52, 52)
	ed.openSearch(false)
	typeFind(ed, "bool")
	ed.searchNext()
	frame(nil)
	shot := func(name string) {
		dl := frame(nil)
		dl = frame(nil) // icons baked on the first paint
		g := ed.findGeom()
		crop := image.Rect(int(g.bar.X-8)*scale, int(g.bar.Y-6)*scale, int(g.bar.X+g.bar.W+8)*scale, int(g.bar.Y+g.bar.H+8)*scale)
		saveSnapshot(t, name, rasterize(dl, text, w, h, scale, theme.Current().Background), crop)
	}
	shot("find.png")
	ed.openSearch(true)
	ed.search.caseSens = true
	shot("find_replace.png")
}
