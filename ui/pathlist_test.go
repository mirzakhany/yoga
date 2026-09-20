package ui

import (
	"testing"

	"github.com/mirzakhany/yoga/icons"
	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
	"github.com/mirzakhany/yoga/theme"
)

func pathListFrame(t *testing.T, n *Node, w, h float32) (*Ctx, *layout.Element) {
	t.Helper()
	text, err := shape.NewEngine(1, false)
	if err != nil {
		t.Fatal(err)
	}
	sheet := render.NewSpriteSheet(text.Atlas)
	SetFrameResources(text, sheet, nil)
	c := New(text, NewFocusScope(), nil)
	c.SetIcons(sheet)
	root := BuildFrame(c, func(_ *Ctx) View { return n }, w, h, nil, nil)
	return c, root
}

// clickElement sends a press-and-release at the centre of e.
func clickElement(root, e *layout.Element) {
	layout.Dispatch(root, &input.Mouse{
		X:        e.Frame.X + e.Frame.W/2,
		Y:        e.Frame.Y + e.Frame.H/2,
		Pressed:  true,
		Released: true,
	})
}

// squareElements collects the square controls (icon buttons) in paint order.
func squareElements(e *layout.Element, out *[]*layout.Element) {
	if e == nil {
		return
	}
	if e.Frame.W > 0 && e.Frame.W == e.Frame.H && len(e.Children) == 0 {
		*out = append(*out, e)
	}
	for _, ch := range e.Children {
		squareElements(ch, out)
	}
}

func TestPathListRemoveDropsThatRow(t *testing.T) {
	paths := []string{"/a/one.proto", "/b/two.proto", "/c/three.proto"}
	var got []string
	n := PathList("pl", paths).
		PathIcon(icons.Icon{}).
		OnPaths(func(next []string) { got = next }).
		Width(400)

	_, root := pathListFrame(t, n, 400, 300)

	var squares []*layout.Element
	squareElements(root, &squares)
	if len(squares) < 3 {
		t.Fatalf("expected a remove button per row, found %d squares", len(squares))
	}
	clickElement(root, squares[1])

	if want := []string{"/a/one.proto", "/c/three.proto"}; !equalStrings(got, want) {
		t.Fatalf("after removing row 1: got %v want %v", got, want)
	}
}

func TestPathListRemoveIsIndexedNotValueMatched(t *testing.T) {
	// Duplicate paths are legal; removing one must not drop both.
	paths := []string{"/a/one.proto", "/a/one.proto"}
	var got []string
	n := PathList("pl", paths).
		PathIcon(icons.Icon{}).
		OnPaths(func(next []string) { got = next }).
		Width(400)

	_, root := pathListFrame(t, n, 400, 300)

	var squares []*layout.Element
	squareElements(root, &squares)
	if len(squares) < 2 {
		t.Fatalf("expected 2 remove buttons, found %d", len(squares))
	}
	clickElement(root, squares[0])

	if len(got) != 1 || got[0] != "/a/one.proto" {
		t.Fatalf("got %v, want one copy left", got)
	}
}

func TestPathListAddCallsOnAdd(t *testing.T) {
	added := 0
	n := PathList("pl", nil).
		AddLabel("Add import path…").
		OnAdd(func() { added++ }).
		Width(400)

	c, root := pathListFrame(t, n, 400, 300)

	btn := buttonElement(c, "pl-add")
	if btn == nil {
		t.Fatal("add button not found in the tree")
	}
	clickElement(root, btn)

	if added != 1 {
		t.Fatalf("OnAdd fired %d times, want 1", added)
	}
}

func TestPathListEmptyShowsPlaceholderAndNoAddOptOut(t *testing.T) {
	c, root := pathListFrame(t, PathList("pl", nil).
		PathEmptyText("No import paths").
		NoAdd().
		Width(400), 400, 300)

	if btn := buttonElement(c, "pl-add"); btn != nil {
		t.Fatal("NoAdd should drop the add button")
	}
	// The placeholder is the only thing drawn, so look for a box the exact
	// width of that string plus its padding.
	th := theme.Current()
	tw, _ := c.Text().MeasureAt("No import paths", th.Typography.Body.Size)
	want := tw + 2*th.Spacing.XS
	if findChildByWidth(root, want) == nil {
		t.Fatalf("no element %v wide; the placeholder was not rendered", want)
	}
}

// buttonElement returns the laid-out element of the button with that id, or
// nil when this frame did not build one.
func buttonElement(c *Ctx, id string) *layout.Element {
	st, ok := c.peekWidget(id).(*buttonState)
	if !ok {
		return nil
	}
	return st.el
}

func TestPathListRowsStayInsideANarrowList(t *testing.T) {
	// Long paths must elide rather than push the remove button out of view.
	paths := []string{"/Users/me/src/proj/vendor/google/api/annotations.proto"}
	n := PathList("pl", paths).Width(180)

	_, root := pathListFrame(t, n, 180, 300)

	var squares []*layout.Element
	squareElements(root, &squares)
	if len(squares) == 0 {
		t.Fatal("no remove button")
	}
	rm := squares[len(squares)-1]
	if right := rm.Frame.X + rm.Frame.W; right > 180.5 {
		t.Fatalf("remove button right edge %v overflows the 180px list", right)
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
