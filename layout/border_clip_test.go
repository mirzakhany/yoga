package layout

import (
	"testing"

	"github.com/mirzakhany/yoga/render"
)

// A child that fills its parent used to paint its background over the parent's
// left and right border stroke; it is clipped to the inside of the stroke now.
func TestBorderedParentClipsFillingChild(t *testing.T) {
	child := New(Box().FlexGrow(1))
	child.Style.BgColor = render.RGBA8(0, 0, 255, 255)
	root := New(Box().Border(render.RGBA8(255, 0, 0, 255), 1, 0), child)
	layoutRoot(root, 200, 100)

	dl := &render.DrawList{}
	Paint(root, dl, nil)

	var clip render.Rect
	found := false
	for _, cmd := range dl.Commands {
		if cmd.Clip.W >= 0 {
			clip = cmd.Clip
			found = true
		}
	}
	if !found {
		t.Fatalf("child was not clipped to the inside of the border")
	}
	want := render.Rect{X: 1, Y: 1, W: 198, H: 98}
	if clip != want {
		t.Fatalf("clip: got %+v want %+v", clip, want)
	}
}

// A child that already sits inside the stroke needs no clip, so the frame stays
// a single draw command.
func TestBorderedParentSkipsClipForInsetChild(t *testing.T) {
	child := New(Box().FlexGrow(1))
	child.Style.BgColor = render.RGBA8(0, 0, 255, 255)
	root := New(Box().Border(render.RGBA8(255, 0, 0, 255), 1, 0).PaddingAll(4), child)
	layoutRoot(root, 200, 100)

	dl := &render.DrawList{}
	Paint(root, dl, nil)

	for _, cmd := range dl.Commands {
		if cmd.Clip.W >= 0 {
			t.Fatalf("unexpected clip %+v for an inset child", cmd.Clip)
		}
	}
}
