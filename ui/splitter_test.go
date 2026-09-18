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
	// Percents must resolve on the very first frame; a later frame must not move them.
	want := float32(0.25 * (800 - splitHandleHit))
	for frame := 1; frame <= 2; frame++ {
		row := findSplitRow(BuildFrame(c, body, 800, 400, nil, nil))
		if row == nil || len(row.Children) != 3 {
			t.Fatalf("want pane/handle/pane, got %v children", childCount(row))
		}
		if got := row.Children[0].Frame.W; math.Abs(float64(got-want)) > 1 {
			t.Fatalf("frame %d: left pane width = %v want ~%v", frame, got, want)
		}
	}
}

func TestSplitterPercentsIgnoreContent(t *testing.T) {
	c := New(nil, NewFocusScope(), nil)
	root := BuildFrame(c, func(_ *Ctx) View {
		return Splitter("pct-content", Horizontal,
			Text("a"),
			Text("a much longer piece of text that is wider than its sibling"),
		).Percents(50, 50).Grow(1)
	}, 1000, 400, nil, nil)
	row := findSplitRow(root)
	l, r := row.Children[0].Frame.W, row.Children[2].Frame.W
	if math.Abs(float64(l-r)) > 1 {
		t.Fatalf("50/50 split uneven on first frame: %v vs %v", l, r)
	}
}

func TestSplitterNestedPercentsFirstFrame(t *testing.T) {
	c := New(nil, NewFocusScope(), nil)
	root := BuildFrame(c, func(_ *Ctx) View {
		inner := Splitter("nest-inner", Horizontal, Text("a"), Text("b")).Percents(50, 50).Grow(1)
		return Splitter("nest-outer", Horizontal, Text("side"), inner).Percents(20, 80).Grow(1)
	}, 1000, 400, nil, nil)
	outer := findSplitRow(root)
	inner := findSplitRow(outer.Children[2])
	if inner == outer || inner == nil {
		t.Fatal("inner splitter row not found")
	}
	avail := float32(1000 - splitHandleHit)
	if got := outer.Children[0].Frame.W; math.Abs(float64(got-0.2*avail)) > 1 {
		t.Fatalf("outer left = %v want ~%v", got, 0.2*avail)
	}
	innerAvail := 0.8*avail - splitHandleHit
	for _, i := range []int{0, 2} {
		if got := inner.Children[i].Frame.W; math.Abs(float64(got-innerAvail/2)) > 1 {
			t.Fatalf("inner pane %d = %v want ~%v", i, got, innerAvail/2)
		}
	}
}

func TestSplitterPercentsRespectMinMax(t *testing.T) {
	c := New(nil, NewFocusScope(), nil)
	body := func(_ *Ctx) View {
		return Splitter("pct-min", Horizontal, Text("a"), Text("b"), Text("c")).
			Percents(10, 45, 45).
			MinSizes(200).
			MaxSizes(0, 300).
			Grow(1)
	}
	row := BuildFrame(c, body, 1000, 400, nil, nil).Children[0]
	if len(row.Children) != 5 {
		t.Fatalf("want 3 panes + 2 handles, got %v children", len(row.Children))
	}
	a, b, cc := row.Children[0].Frame.W, row.Children[2].Frame.W, row.Children[4].Frame.W
	if math.Abs(float64(a-200)) > 1 {
		t.Fatalf("pane a = %v want min 200", a)
	}
	if math.Abs(float64(b-300)) > 1 {
		t.Fatalf("pane b = %v want max 300", b)
	}
	want := float32(1000-2*splitHandleHit) - 200 - 300
	if math.Abs(float64(cc-want)) > 1 {
		t.Fatalf("pane c = %v want remaining %v", cc, want)
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

func TestSplitterPercentDrag(t *testing.T) {
	c := New(nil, NewFocusScope(), nil)
	body := func(_ *Ctx) View {
		return Splitter("pct-drag", Horizontal, Text("a"), Text("b")).Percents(50, 50).Grow(1)
	}
	row := findSplitRow(BuildFrame(c, body, 1000, 400, nil, nil))
	startL := row.Children[0].Frame.W
	h := row.Children[1]
	x, y := h.Frame.X+1, h.Frame.Y+1
	h.OnMouse(h, &input.Mouse{X: x, Y: y, Pressed: true, Down: true})
	h.OnMouse(h, &input.Mouse{X: x + 100, Y: y, Down: true})

	row = findSplitRow(BuildFrame(c, body, 1000, 400, nil, nil))
	l, r := row.Children[0].Frame.W, row.Children[2].Frame.W
	if math.Abs(float64(l-(startL+100))) > 1 {
		t.Fatalf("left after drag = %v want ~%v", l, startL+100)
	}
	if math.Abs(float64(l+r-float32(1000-splitHandleHit))) > 1 {
		t.Fatalf("panes no longer fill splitter: %v + %v", l, r)
	}
	// Resizing the window keeps the dragged ratio.
	row = findSplitRow(BuildFrame(c, body, 500, 400, nil, nil))
	wantL := (startL + 100) / float32(1000-splitHandleHit) * float32(500-splitHandleHit)
	if got := row.Children[0].Frame.W; math.Abs(float64(got-wantL)) > 1 {
		t.Fatalf("left after resize = %v want ~%v", got, wantL)
	}
}
