package main

import (
	"os"
	"testing"

	"github.com/mirzakhany/yoga"
	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
	"github.com/mirzakhany/yoga/ui"
)

// TestGalleryOpenXMLWrapped drives the real gallery editor page through the
// real ui.BuildFrame flow and verifies that opening the big single-line XML
// file produces a soft-wrapped layout (many visual rows, scrollable content)
// instead of the old two-line unwrapped view.
func TestGalleryOpenXMLWrapped(t *testing.T) {
	xmlPath := "facility-monitoring.xml"
	if _, err := os.Stat(xmlPath); err != nil {
		t.Skip("sample xml not available")
	}

	text, err := shape.NewEngine(1, false)
	if err != nil {
		t.Fatal(err)
	}
	sheet := render.NewSpriteSheet(text.Atlas)
	clip := &input.MemClipboard{}
	yoga.SetResources(text, sheet, clip)

	ws := buildEditorPage()
	defer ws.close()

	const w, h = 1100, 720
	c := ui.New(text, ui.NewFocusScope(), nil)
	c.SetIcons(sheet)
	c.SetClipboard(clip)
	mouse := &input.Mouse{}
	keyboard := &input.Keyboard{}

	var lastVerts int
	var lastDL *render.DrawList
	for frame := 0; frame < 3; frame++ {
		root := ui.BuildFrame(c, ws.Layout, w, h, mouse, keyboard)
		layout.Dispatch(root, mouse)
		mouse.EndFrame()
		keyboard.EndFrame()

		dl := &render.DrawList{}
		layout.Paint(root, dl, text)
		lastVerts = len(dl.Vertices)
		lastDL = dl

		ed := ws.activeDoc()
		t.Logf("frame %d: doc=%s visualRows=%d ContentH=%.0f verts=%d", frame, ed.Path, ed.VisualRowCount(), ed.ContentHeight, lastVerts)

		if frame == 0 {
			// Open the XML through the page's real open path.
			ws.openFile(xmlPath)
		}
	}

	ed := ws.activeDoc()
	if ed.Path == "" || !ed.SoftWrap || ed.VisualRowCount() < 1000 {
		t.Fatalf("XML doc not soft-wrapped: path=%q wrap=%v visualRows=%d", ed.Path, ed.SoftWrap, ed.VisualRowCount())
	}
	if ed.ContentHeight <= h {
		t.Fatalf("ContentHeight=%.0f should exceed viewport %d so the doc can scroll", ed.ContentHeight, h)
	}
	if lastVerts == 0 || lastVerts > 80000 {
		t.Fatalf("paint verts=%d out of expected range", lastVerts)
	}
	// Wrapped rows must be drawn at the text origin (on-screen), not offset
	// by their line-relative x (which is megabytes of pixels deep in the line).
	_, maxX := float32(1e9), float32(-1)
	for _, v := range lastDL.Vertices {
		if v.Pos[0] > maxX {
			maxX = v.Pos[0]
		}
	}
	atOrigin := false
	for _, v := range lastDL.Vertices {
		if v.Pos[0] >= 60 && v.Pos[0] <= 300 {
			atOrigin = true
			break
		}
	}
	if !atOrigin {
		t.Fatalf("no glyphs near the text origin (max x %.1f) — rows drawn off-screen", maxX)
	}
	if maxX > w+40 {
		t.Fatalf("glyph max x = %.1f, beyond the %dpx viewport", maxX, w)
	}
}
