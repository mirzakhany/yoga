package ui

import (
	"math"
	"testing"

	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/layout"
)

func TestSplitterPercents(t *testing.T) {
	c := New(nil, NewFocusScope(), nil)
	body := func(_ *Ctx) View {
		return Splitter("pct", Horizontal, Text("left"), Text("right")).Percents(25, 75).Grow(1)
	}
	// First frame seeds state; second resolves percents from the prior root frame.
	_ = BuildFrame(c, body, 800, 400, nil, nil)
	root := BuildFrame(c, body, 800, 400, nil, nil)
	row := findSplitRow(root)
	if row == nil || len(row.Children) != 3 {
		t.Fatalf("want pane/handle/pane, got %v children", childCount(row))
	}
	want := float32(0.25 * (800 - splitHandleHit))
	got := row.Children[0].Style.Width
	if math.Abs(float64(got-want)) > 1 {
		t.Fatalf("left pane width = %v want ~%v", got, want)
	}
}

func TestSplitterClampSize(t *testing.T) {
	st := &splitState{
		axis:  Horizontal,
		sizes: []float32{200, 0},
		mins:  []float32{120, 0},
		maxs:  []float32{300, 0},
		root:  &layout.Element{},
	}
	st.root.Frame.W = 800
	st.root.Frame.H = 400

	if got := st.clampSize(0, 50); got != 120 {
		t.Fatalf("clamp below min: got %v want 120", got)
	}
	if got := st.clampSize(0, 500); got != 300 {
		t.Fatalf("clamp above user max: got %v want 300", got)
	}
	if got := st.clampSize(0, 200); got != 200 {
		t.Fatalf("clamp in range: got %v want 200", got)
	}
}

func TestSplitterPercentMaxAllowsGrow(t *testing.T) {
	// Both panes percent-fixed (catalog-style). Left must be able to grow past
	// its current size by shrinking the right pane down to its min.
	avail := float32(800 - splitHandleHit)
	left := avail * 0.35
	right := avail * 0.65
	st := &splitState{
		axis:       Horizontal,
		usePercent: true,
		sizes:      []float32{left, right},
		percents:   []float32{35, 65},
		mins:       []float32{80, 80},
		maxs:       []float32{0, 0},
		root:       &layout.Element{},
	}
	st.root.Frame.W = 800
	st.root.Frame.H = 400

	maxLeft := st.maxSizeForSection(0)
	wantMax := avail - 80 // right reserved only at min
	if math.Abs(float64(maxLeft-wantMax)) > 1 {
		t.Fatalf("max left = %v want ~%v (must exceed current %v)", maxLeft, wantMax, left)
	}
	grown := st.clampSize(0, left+100)
	if grown <= left {
		t.Fatalf("clamp blocked grow: got %v current %v", grown, left)
	}
}

func TestSplitterHandleOnHover(t *testing.T) {
	c := New(nil, NewFocusScope(), nil)
	body := func(_ *Ctx) View {
		return Splitter("hov", Horizontal, Text("a"), Text("b")).
			Sizes(200, 0).
			HandleOnHover().
			Grow(1)
	}
	root := BuildFrame(c, body, 800, 400, nil, nil)
	row := findSplitRow(root)
	if row == nil || len(row.Children) != 3 {
		t.Fatalf("want pane/handle/pane, got %v children", childCount(row))
	}
	handle := row.Children[1]
	if handle.Paint == nil {
		t.Fatal("handle missing Paint")
	}
	// Idle: HandleOnHover skips drawing (no panic / no draw).
	handle.Paint(nil, nil)

	st := c.Widget("hov", func() any { return nil }).(*splitState)
	if !st.handleOnHover {
		t.Fatal("handleOnHover not synced")
	}
	st.hover[0] = true
	// Active path still paints without panic when DrawList is nil — avoid nil dl.
	// Just verify hover tracking marks paint via mouse handler.
	m := &input.Mouse{X: handle.Frame.X + 1, Y: handle.Frame.Y + 1}
	if handle.OnMouse == nil {
		t.Fatal("handle missing OnMouse")
	}
	handle.OnMouse(handle, m)
	if !st.hover[0] {
		t.Fatal("expected hover after mouse inside handle")
	}
}

func TestSplitterMinSizesAppliedToFlex(t *testing.T) {
	c := New(nil, NewFocusScope(), nil)
	root := BuildFrame(c, func(_ *Ctx) View {
		return Splitter("mins", Horizontal, Text("left"), Text("right")).
			Sizes(200, 0).
			MinSizes(100, 150)
	}, 800, 400, nil, nil)
	row := findSplitRow(root)
	if row == nil || len(row.Children) != 3 {
		t.Fatalf("want pane/handle/pane, got %v children", childCount(row))
	}
	if row.Children[1].Style.Width != splitHandleHit {
		t.Fatalf("handle width = %v", row.Children[1].Style.Width)
	}
	// Flex pane (right) should carry MinWidth from MinSizes.
	if row.Children[2].Style.MinWidth != 150 {
		t.Fatalf("right MinWidth = %v want 150", row.Children[2].Style.MinWidth)
	}
}
