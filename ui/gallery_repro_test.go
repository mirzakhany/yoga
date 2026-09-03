package ui

import (
	"os"
	"testing"

	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
)

// TestGalleryXMLWrapRepro reproduces the gallery app flow: open the real XML
// file via NewEditorFor, enable wrap via field assignment (as EditorPage does),
// and run several frames with layout in between.
func TestGalleryXMLWrapRepro(t *testing.T) {
	data, err := os.ReadFile("../example/gallery/facility-monitoring.xml")
	if err != nil {
		t.Skip("sample file not available")
	}
	text, err := shape.NewEngine(1, false)
	if err != nil {
		t.Fatal(err)
	}
	SetFrameResources(text, render.NewSpriteSheet(text.Atlas), nil)

	ed := NewEditorFor("../example/gallery/facility-monitoring.xml", data)
	ed.SoftWrap = true // field set, like EditorPage.openFile
	defer ed.Close()

	for frame := 0; frame < 3; frame++ {
		c := New(frameText(), NewFocusScope(), nil)
		c.BeginFrame(1200, 800, nil, nil)
		_ = ed.Layout(c)
		root := layout.New(layout.Box(), ed.host)
		root.Calculate(1200, 800)
		dl := &render.DrawList{}
		layout.Paint(root, dl, frameText())
		t.Logf("frame %d: wrapRows=%d wrapCols=%d ContentH=%.0f ContentW=%.0f ScrollPx=%.0f verts=%d",
			frame, ed.VisualRowCount(), ed.wrapCols, ed.ContentHeight, ed.ContentWidth, ed.ScrollPx, len(dl.Vertices))
	}

	if ed.VisualRowCount() < 1000 {
		t.Fatalf("wrap table has %d rows, want thousands", ed.VisualRowCount())
	}
	// Scroll down and paint; vertices must stay bounded and rows must advance.
	ed.ScrollPx = float32(100) * ed.lineH
	root2 := layout.New(layout.Box(), ed.host)
	root2.Calculate(1200, 800)
	dl2 := &render.DrawList{}
	layout.Paint(root2, dl2, frameText())
	if len(dl2.Vertices) == 0 || len(dl2.Vertices) > 60000 {
		t.Fatalf("scrolled paint verts=%d", len(dl2.Vertices))
	}
}
