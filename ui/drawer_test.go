package ui

import (
	"testing"
	"time"

	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/layout"
)

func markerPage(id string) View {
	return Raw(layout.New(layout.Box().W(100).H(80)))
}

func markerPanel(id string) View {
	return Raw(layout.New(layout.Box().W(60).H(40)))
}

func setDrawerOpen(c *Ctx, id string) {
	st := c.Widget(id, func() any { return &drawerState{} }).(*drawerState)
	st.progress = 1
	st.animating = false
	st.animTarget = 1
}

func drawerRoot(c *Ctx, d *drawerView) *layout.Element {
	return BuildFrame(c, func(_ *Ctx) View { return d.Grow(1) }, 400, 300, nil, nil)
}

func TestDrawerEdgeTopBottomOverlaySlideOrigin(t *testing.T) {
	c := New(nil, NewFocusScope(), nil)
	panelMarker := Raw(layout.New(layout.Box().W(60).H(40)))
	pageMarker := Raw(layout.New(layout.Box().W(100).H(80)))

	cases := []struct {
		id         string
		edge       Edge
		wantAbove  bool
		wantBelow  bool
	}{
		{"s-top", EdgeTop, true, false},
		{"s-bottom", EdgeBottom, false, true},
	}
	for _, tc := range cases {
		d := Drawer(tc.id, panelMarker, pageMarker).Edge(tc.edge).Overlay().Open(true).Size(120)
		_ = drawerContentRoot(drawerRoot(c, d))
		st := c.Widget(tc.id, func() any { return &drawerState{} }).(*drawerState)
		st.progress = 0
		st.swiping = true
		st.animating = false
		root := drawerContentRoot(drawerRoot(c, d))

		var panel *layout.Element
		var walk func(*layout.Element)
		walk = func(e *layout.Element) {
			if e != nil && approxEq(e.Frame.H, 120, 2) && e.Frame.W >= 350 {
				panel = e
			}
			for _, ch := range e.Children {
				walk(ch)
			}
		}
		walk(root)
		if panel == nil {
			t.Fatalf("%s: closed overlay panel not found", tc.id)
		}
		if tc.wantAbove && panel.Frame.Y+panel.Frame.H > 0 {
			t.Fatalf("%s: closed top drawer should start above viewport, Y=%v H=%v", tc.id, panel.Frame.Y, panel.Frame.H)
		}
		if tc.wantBelow && panel.Frame.Y < 300 {
			t.Fatalf("%s: closed bottom drawer should start below viewport, Y=%v", tc.id, panel.Frame.Y)
		}
	}
}

func TestDrawerEdgeTopBottomPushPosition(t *testing.T) {
	c := New(nil, NewFocusScope(), nil)
	panelMarker := Raw(layout.New(layout.Box().W(60).H(40)))
	pageMarker := Raw(layout.New(layout.Box().W(100).H(80)))

	cases := []struct {
		id      string
		edge    Edge
		wantTop bool
	}{
		{"p-top", EdgeTop, true},
		{"p-bottom", EdgeBottom, false},
	}
	for _, tc := range cases {
		d := Drawer(tc.id, panelMarker, pageMarker).Edge(tc.edge).Push().Open(true).Size(120).Resizable(true)
		_ = drawerContentRoot(drawerRoot(c, d))
		setDrawerOpen(c, tc.id)
		root := drawerContentRoot(drawerRoot(c, d))

		var panel *layout.Element
		var walk func(*layout.Element)
		walk = func(e *layout.Element) {
			if e != nil && approxEq(e.Frame.H, 120, 2) && e.Frame.W >= 350 {
				panel = e
			}
			for _, ch := range e.Children {
				walk(ch)
			}
		}
		walk(root)
		if panel == nil {
			t.Fatalf("%s: panel not found", tc.id)
		}
		if tc.wantTop && panel.Frame.Y >= 20 {
			t.Fatalf("%s: push panel Y=%v expected near top", tc.id, panel.Frame.Y)
		}
		if !tc.wantTop && panel.Frame.Y+panel.Frame.H <= 280 {
			t.Fatalf("%s: push panel Y=%v H=%v expected near bottom", tc.id, panel.Frame.Y, panel.Frame.H)
		}
	}
}

func TestDrawerEdgeTopBottomOverlayPosition(t *testing.T) {
	c := New(nil, NewFocusScope(), nil)
	panelMarker := Raw(layout.New(layout.Box().W(60).H(40)))
	pageMarker := Raw(layout.New(layout.Box().W(100).H(80)))

	cases := []struct {
		id      string
		edge    Edge
		wantTop bool
	}{
		{"d-top", EdgeTop, true},
		{"d-bottom", EdgeBottom, false},
	}
	for _, tc := range cases {
		d := Drawer(tc.id, panelMarker, pageMarker).Edge(tc.edge).Overlay().Open(true).Size(120)
		_ = drawerContentRoot(drawerRoot(c, d))
		setDrawerOpen(c, tc.id)
		root := drawerContentRoot(drawerRoot(c, d))

		var panel *layout.Element
		var walk func(*layout.Element)
		walk = func(e *layout.Element) {
			if e != nil && e.Frame.H >= 100 && e.Frame.W >= 350 {
				panel = e
			}
			for _, ch := range e.Children {
				walk(ch)
			}
		}
		walk(root)
		if panel == nil {
			t.Fatalf("%s: panel not found", tc.id)
		}
		atTop := panel.Frame.Y < 20
		atBottom := panel.Frame.Y+panel.Frame.H > 280
		if tc.wantTop && !atTop {
			t.Fatalf("%s: panel Y=%v H=%v expected near top", tc.id, panel.Frame.Y, panel.Frame.H)
		}
		if !tc.wantTop && !atBottom {
			t.Fatalf("%s: panel Y=%v H=%v expected near bottom", tc.id, panel.Frame.Y, panel.Frame.H)
		}
	}
}

func TestDrawerPushRightShrinksPage(t *testing.T) {
	c := New(nil, NewFocusScope(), nil)
	d := Drawer("dr-push", markerPanel("p"), markerPage("page")).
		Edge(EdgeRight).Push().Open(true).Size(320)
	_ = drawerRoot(c, d)
	setDrawerOpen(c, "dr-push")
	root := drawerRoot(c, d)

	page := findChildByWidth(root, 400-320)
	if page == nil {
		t.Fatalf("page not found; root children: %d", len(root.Children))
	}
	if got := page.Frame.W; got < 75 || got > 85 {
		t.Fatalf("page width = %v want ~80", got)
	}
	panel := findDrawerChromePanel(root, func(e *layout.Element) bool {
		return approxEq(e.Frame.W, 320, 8)
	})
	if panel == nil {
		t.Fatal("drawer panel not found")
	}
	if panel.Frame.W < 310 {
		t.Fatalf("panel width = %v want ~320", panel.Frame.W)
	}
}

func TestDrawerOverlayRightKeepsPageFullWidth(t *testing.T) {
	c := New(nil, NewFocusScope(), nil)
	d := Drawer("dr-ov", markerPanel("p"), markerPage("page")).
		Edge(EdgeRight).Overlay().Open(true).Size(320)
	_ = drawerRoot(c, d)
	setDrawerOpen(c, "dr-ov")
	root := drawerRoot(c, d)

	page := root.Children[0]
	if page.Frame.W < 390 {
		t.Fatalf("overlay page width = %v want full host ~400", page.Frame.W)
	}
}

func drawerContentRoot(root *layout.Element) *layout.Element {
	if len(root.Children) >= 1 {
		return root.Children[0]
	}
	return root
}

func TestDrawerPushClosedPageFillsHost(t *testing.T) {
	c := New(nil, NewFocusScope(), nil)
	d := Drawer("dr-cl", markerPanel("p"), markerPage("page")).
		Edge(EdgeRight).Push().Open(false).Size(320)
	root := drawerContentRoot(drawerRoot(c, d))
	if len(root.Children) != 1 {
		t.Fatalf("closed push drawer row should have one child, got %d", len(root.Children))
	}
	if root.Children[0].Frame.W < 390 {
		t.Fatalf("page width = %v want ~400", root.Children[0].Frame.W)
	}
}

func TestDrawerNestedRightBottom(t *testing.T) {
	c := New(nil, NewFocusScope(), nil)
	inner := Drawer("term", markerPanel("term"), markerPage("editor")).
		Edge(EdgeBottom).Push().Open(true).Size(180)
	outer := Drawer("chat", markerPanel("chat"), inner).
		Edge(EdgeRight).Push().Open(true).Size(200)
	_ = drawerRoot(c, outer)
	setDrawerOpen(c, "term")
	setDrawerOpen(c, "chat")
	root := drawerRoot(c, outer)

	chatHandle := findRightmostResizeHandle(root)
	if chatHandle == nil {
		t.Fatal("chat resize handle not found")
	}
	chatPanel := findDrawerChromePanel(root, func(e *layout.Element) bool {
		return approxEq(e.Frame.W, 200, 8) && containsElement(e, chatHandle)
	})
	if chatPanel == nil {
		t.Fatal("chat panel not found")
	}
	if chatPanel.Frame.H < 290 {
		t.Fatalf("chat panel height = %v want full host ~300", chatPanel.Frame.H)
	}
	if chatPanel.Frame.X < 190 {
		t.Fatalf("chat panel x = %v want ~200", chatPanel.Frame.X)
	}
}

func dragDrawerPanel(t *testing.T, root, handle *layout.Element, edge Edge, delta float32) {
	t.Helper()
	startX := handle.Frame.X + handle.Frame.W/2
	startY := handle.Frame.Y + handle.Frame.H/2
	if edge == EdgeRight || edge == EdgeLeft {
		startY = handle.Frame.Y + handle.Frame.H/2
		if edge == EdgeRight {
			startX = handle.Frame.X + 3
		} else {
			startX = handle.Frame.X + handle.Frame.W - 3
		}
	} else {
		startX = handle.Frame.X + handle.Frame.W/2
		if edge == EdgeBottom {
			startY = handle.Frame.Y + 3
		} else {
			startY = handle.Frame.Y + handle.Frame.H - 3
		}
	}

	host := drawerContentRoot(root)
	if host != nil && host.OnMouse != nil {
		host.OnMouse(host, &input.Mouse{})
	}

	m := &input.Mouse{X: startX, Y: startY, Pressed: true, Down: true}
	handle.OnMouse(handle, m)
	m.EndFrame()

	m.Pressed = false
	if edge == EdgeRight || edge == EdgeLeft {
		m.X += delta
	} else {
		m.Y += delta
	}
	if host != nil && host.OnMouse != nil {
		host.OnMouse(host, m)
	}
	m.EndFrame()

	m.Down = false
	if host != nil && host.OnMouse != nil {
		host.OnMouse(host, m)
	}
}

func TestDrawerResizeHandleUpdatesSize(t *testing.T) {
	c := New(nil, NewFocusScope(), nil)
	d := Drawer("dr-rz", markerPanel("p"), markerPage("page")).
		Edge(EdgeRight).Push().Open(true).Size(320).Resizable(true)
	_ = drawerRoot(c, d)
	setDrawerOpen(c, "dr-rz")
	root := drawerRoot(c, d)

	handle := findDrawerResizeHandle(root)
	if handle == nil {
		t.Fatal("drawer resize handle not found")
	}
	if handle.OnMouse == nil {
		t.Fatal("resize handle has no OnMouse")
	}
	st := c.Widget("dr-rz", func() any { return &drawerState{} }).(*drawerState)
	before := st.size

	dragDrawerPanel(t, root, handle, EdgeRight, 40) // toward page shrinks

	if st.size >= before {
		t.Fatalf("size should shrink when dragging handle toward page, before=%v after=%v", before, st.size)
	}
}

func TestDrawerResizeHandleEdgeBottom(t *testing.T) {
	c := New(nil, NewFocusScope(), nil)
	d := Drawer("dr-rz-b", markerPanel("p"), markerPage("page")).
		Edge(EdgeBottom).Push().Open(true).Size(200).Resizable(true)
	_ = drawerRoot(c, d)
	setDrawerOpen(c, "dr-rz-b")
	root := drawerRoot(c, d)

	handle := findDrawerResizeHandle(root)
	if handle == nil {
		t.Fatal("drawer resize handle not found")
	}
	st := c.Widget("dr-rz-b", func() any { return &drawerState{} }).(*drawerState)
	before := st.size

	dragDrawerPanel(t, root, handle, EdgeBottom, 40) // toward page shrinks

	if st.size >= before {
		t.Fatalf("bottom drawer should shrink when dragging handle toward page, before=%v after=%v", before, st.size)
	}
}

func TestDrawerPanelContentFillsPanel(t *testing.T) {
	c := New(nil, NewFocusScope(), nil)
	d := Drawer("dr-fill", markerPanel("p"), markerPage("page")).
		Edge(EdgeRight).Push().Open(true).Size(320).Resizable(true)
	_ = drawerRoot(c, d)
	setDrawerOpen(c, "dr-fill")
	root := drawerRoot(c, d)

	panel := findDrawerChromePanel(root, func(e *layout.Element) bool {
		return approxEq(e.Frame.W, 320, 8)
	})
	if panel == nil {
		t.Fatal("drawer panel not found")
	}
	if len(panel.Children) == 0 {
		t.Fatal("panel has no children")
	}
	viewport := findDrawerViewport(panel)
	if viewport == nil {
		t.Fatal("drawer viewport not found")
	}
	if viewport.Frame.W < 310 {
		t.Fatalf("viewport width = %v want ~314", viewport.Frame.W)
	}
}

func TestDrawerPanelContentClampsWhenSmall(t *testing.T) {
	c := New(nil, NewFocusScope(), nil)
	d := Drawer("dr-clamp", markerPanel("p"), markerPage("page")).
		Edge(EdgeRight).Overlay().Open(true).Size(120).Resizable(true)
	_ = drawerRoot(c, d)
	setDrawerOpen(c, "dr-clamp")
	root := drawerRoot(c, d)

	panel := findDrawerChromePanel(root, func(e *layout.Element) bool {
		return approxEq(e.Frame.W, 120, 8)
	})
	if panel == nil {
		t.Fatal("drawer panel not found")
	}
	if !panel.Clip {
		t.Fatal("panel should clip overflowing content")
	}
	viewport := findDrawerViewport(panel)
	if viewport == nil {
		t.Fatal("drawer viewport not found")
	}
	if viewport.Frame.W > 125 {
		t.Fatalf("viewport width = %v want ~120", viewport.Frame.W)
	}
}

func TestDrawerOverlayRightResizeHandleOnInnerEdge(t *testing.T) {
	c := New(nil, NewFocusScope(), nil)
	d := Drawer("dr-ov-rz", markerPanel("p"), markerPage("page")).
		Edge(EdgeRight).Overlay().Open(true).Size(320).Resizable(true)
	_ = drawerRoot(c, d)
	setDrawerOpen(c, "dr-ov-rz")
	root := drawerRoot(c, d)

	handle := findDrawerResizeHandle(root)
	if handle == nil {
		t.Fatal("overlay resize handle not found")
	}
	// Inner edge for a right drawer is the panel's left side.
	if handle.Frame.X > 90 {
		t.Fatalf("handle X=%v want near panel left (~80)", handle.Frame.X)
	}
	if handle.Frame.W < 5 || handle.Frame.W > 7 {
		t.Fatalf("handle width = %v want ~6", handle.Frame.W)
	}

	st := c.Widget("dr-ov-rz", func() any { return &drawerState{} }).(*drawerState)
	before := st.size
	dragDrawerPanel(t, root, handle, EdgeRight, 40)
	if st.size >= before {
		t.Fatalf("overlay resize should shrink toward page, before=%v after=%v", before, st.size)
	}
}

func TestDrawerOverlayBottomResizeHandleOnInnerEdge(t *testing.T) {
	c := New(nil, NewFocusScope(), nil)
	d := Drawer("dr-ov-b", markerPanel("p"), markerPage("page")).
		Edge(EdgeBottom).Overlay().Open(true).Size(200).Resizable(true)
	_ = drawerRoot(c, d)
	setDrawerOpen(c, "dr-ov-b")
	root := drawerRoot(c, d)

	handle := findDrawerResizeHandle(root)
	if handle == nil {
		t.Fatal("overlay resize handle not found")
	}
	// Inner edge for a bottom drawer is the panel's top side.
	if handle.Frame.Y > 110 {
		t.Fatalf("handle Y=%v want near panel top (~100)", handle.Frame.Y)
	}
	if handle.Frame.H < 5 || handle.Frame.H > 7 {
		t.Fatalf("handle height = %v want ~6", handle.Frame.H)
	}

	st := c.Widget("dr-ov-b", func() any { return &drawerState{} }).(*drawerState)
	before := st.size
	dragDrawerPanel(t, root, handle, EdgeBottom, 40)
	if st.size >= before {
		t.Fatalf("overlay bottom resize should shrink toward page, before=%v after=%v", before, st.size)
	}
}

func TestDrawerGeomApplySizeDelta(t *testing.T) {
	cases := []struct {
		edge      Edge
		start     float32
		delta     float32
		want      float32
		shrinkDir string // description for failure
	}{
		{EdgeRight, 320, 40, 280, "right shrink toward page"},
		{EdgeRight, 320, -40, 360, "right expand away from page"},
		{EdgeLeft, 320, -40, 280, "left shrink toward page"},
		{EdgeLeft, 320, 40, 360, "left expand away from page"},
		{EdgeBottom, 200, 40, 160, "bottom shrink toward page"},
		{EdgeBottom, 200, -40, 240, "bottom expand away from page"},
		{EdgeTop, 200, -40, 160, "top shrink toward page"},
		{EdgeTop, 200, 40, 240, "top expand away from page"},
	}
	for _, tc := range cases {
		g := drawerGeom{edge: tc.edge}
		got := g.applySizeDelta(tc.start, tc.delta)
		if got != tc.want {
			t.Fatalf("%s: applySizeDelta(%v, %v) = %v want %v", tc.shrinkDir, tc.start, tc.delta, got, tc.want)
		}
	}
}

func TestDrawerHandleFlexSiblingRight(t *testing.T) {
	c := New(nil, NewFocusScope(), nil)
	d := Drawer("dr-h-r", markerPanel("p"), markerPage("page")).
		Edge(EdgeRight).Push().Open(true).Size(320).Resizable(true)
	setDrawerOpen(c, "dr-h-r")
	root := drawerRoot(c, d)
	panel := findDrawerChromePanel(root, func(e *layout.Element) bool {
		return approxEq(e.Frame.W, 320, 8)
	})
	if panel == nil {
		t.Fatal("panel not found")
	}
	if len(panel.Children) < 2 {
		t.Fatalf("panel should have handle+viewport, got %d children", len(panel.Children))
	}
	handle := panel.Children[0]
	viewport := panel.Children[1]
	if handle.Style.Pos == layout.PositionAbsolute {
		t.Fatal("handle should be flex sibling, not absolute")
	}
	if !approxEq(handle.Frame.W, drawerHandleHit, 1) {
		t.Fatalf("handle width = %v want ~6", handle.Frame.W)
	}
	if viewport.Frame.X <= handle.Frame.X {
		t.Fatalf("viewport should be right of handle, handle.X=%v viewport.X=%v", handle.Frame.X, viewport.Frame.X)
	}
}

func TestDrawerHandleFlexSiblingBottom(t *testing.T) {
	c := New(nil, NewFocusScope(), nil)
	d := Drawer("dr-h-b", markerPanel("p"), markerPage("page")).
		Edge(EdgeBottom).Push().Open(true).Size(200).Resizable(true)
	setDrawerOpen(c, "dr-h-b")
	root := drawerRoot(c, d)
	panel := findDrawerChromePanel(root, func(e *layout.Element) bool {
		return approxEq(e.Frame.H, 200, 8) && e.Frame.W >= 350
	})
	if panel == nil {
		t.Fatal("panel not found")
	}
	handle := panel.Children[0]
	viewport := panel.Children[1]
	if handle.Style.Pos == layout.PositionAbsolute {
		t.Fatal("handle should be flex sibling, not absolute")
	}
	if !approxEq(handle.Frame.H, drawerHandleHit, 1) {
		t.Fatalf("handle height = %v want ~6", handle.Frame.H)
	}
	if viewport.Frame.Y <= handle.Frame.Y {
		t.Fatalf("viewport should be below handle, handle.Y=%v viewport.Y=%v", handle.Frame.Y, viewport.Frame.Y)
	}
}

func TestDrawerResizeDragContinuesOffHandle(t *testing.T) {
	c := New(nil, NewFocusScope(), nil)
	d := Drawer("dr-drag", markerPanel("p"), markerPage("page")).
		Edge(EdgeRight).Push().Open(true).Size(320).Resizable(true)
	_ = drawerRoot(c, d)
	setDrawerOpen(c, "dr-drag")
	root := drawerRoot(c, d)
	handle := findDrawerResizeHandle(root)
	host := drawerContentRoot(root)
	st := c.Widget("dr-drag", func() any { return &drawerState{} }).(*drawerState)
	before := st.size

	m := &input.Mouse{
		X:       handle.Frame.X + 3,
		Y:       handle.Frame.Y + handle.Frame.H/2,
		Pressed: true,
		Down:    true,
	}
	handle.OnMouse(handle, m)
	m.EndFrame()

	// Move far off the 6px handle strip; root should continue the drag.
	m.Pressed = false
	m.X = handle.Frame.X + 100
	host.OnMouse(host, m)
	m.EndFrame()

	if st.size >= before {
		t.Fatalf("root drag continuation should shrink drawer, before=%v after=%v", before, st.size)
	}
}

func TestDrawerSwipeOpensFromEdge(t *testing.T) {
	c := New(nil, NewFocusScope(), nil)
	open := false
	body := func(_ *Ctx) View {
		return Drawer("dr-sw", markerPanel("p"), markerPage("page")).
			Edge(EdgeRight).Overlay().Swipe(true).Open(open).
			OnOpenChange(func(v bool) { open = v }).
			Grow(1)
	}

	root := BuildFrame(c, body, 400, 300, nil, nil)
	strip := findEdgeStrip(root)
	if strip == nil {
		t.Fatal("edge strip not found")
	}
	m := &input.Mouse{
		X:       strip.Frame.X + strip.Frame.W/2,
		Y:       strip.Frame.Y + strip.Frame.H/2,
		Pressed: true,
		Down:    true,
	}
	strip.OnMouse(strip, m)
	m.Pressed = false
	m.X -= 160
	strip.OnMouse(strip, m)
	m.Released = true
	m.Down = false
	strip.OnMouse(strip, m)

	if !open {
		t.Fatal("swipe past threshold should open drawer")
	}
}

func TestDrawerSwipeScrollDoesNotOpen(t *testing.T) {
	c := New(nil, NewFocusScope(), nil)
	open := false
	d := Drawer("dr-wh", markerPanel("p"), markerPage("page")).
		Edge(EdgeRight).Overlay().Swipe(true).Open(open).
		OnOpenChange(func(v bool) { open = v })
	body := func(_ *Ctx) View { return d.Grow(1) }

	root := BuildFrame(c, body, 400, 300, nil, nil)
	m := &input.Mouse{X: 200, Y: 150, ScrollY: -4}
	layout.Dispatch(root, m)
	if open {
		t.Fatal("wheel scroll should not open drawer")
	}
}

func TestDrawerAnimationSchedulesRepaint(t *testing.T) {
	c := New(nil, NewFocusScope(), nil)
	c.BeginFrame(400, 300, nil, nil)
	c.now = time.Now()
	d := Drawer("dr-anim", markerPanel("p"), markerPage("page")).Open(true)
	d.Layout(c)
	if d2, ok := c.AnimationWait(); !ok || d2 <= 0 {
		t.Fatalf("open drawer should schedule animation, got (%v, %v)", d2, ok)
	}
}

func findChildByWidth(root *layout.Element, want float32) *layout.Element {
	var found *layout.Element
	var walk func(*layout.Element)
	walk = func(e *layout.Element) {
		if e == nil {
			return
		}
		if approxEq(e.Frame.W, want, 8) {
			found = e
		}
		for _, ch := range e.Children {
			walk(ch)
		}
	}
	walk(root)
	return found
}

func findRightmostLeaf(root *layout.Element) *layout.Element {
	var best *layout.Element
	var walk func(*layout.Element)
	walk = func(e *layout.Element) {
		if e == nil {
			return
		}
		if len(e.Children) == 0 && e.Frame.W > 50 {
			if best == nil || e.Frame.X > best.Frame.X {
				best = e
			}
		}
		for _, ch := range e.Children {
			walk(ch)
		}
	}
	walk(root)
	return best
}

func findDrawerChromePanel(root *layout.Element, match func(*layout.Element) bool) *layout.Element {
	var found *layout.Element
	var walk func(*layout.Element)
	walk = func(e *layout.Element) {
		if e == nil {
			return
		}
		if e.Clip && match(e) && drawerPanelHasResizeHandle(e) {
			found = e
		}
		for _, ch := range e.Children {
			walk(ch)
		}
	}
	walk(root)
	return found
}

func drawerPanelHasResizeHandle(panel *layout.Element) bool {
	for _, ch := range panel.Children {
		f := ch.Frame
		if ch.OnMouse != nil &&
			(approxEq(f.W, drawerHandleHit, 1) || approxEq(f.H, drawerHandleHit, 1)) {
			return true
		}
	}
	return false
}

func findRightmostResizeHandle(root *layout.Element) *layout.Element {
	var best *layout.Element
	var walk func(*layout.Element)
	walk = func(e *layout.Element) {
		if e == nil {
			return
		}
		f := e.Frame
		if e.OnMouse != nil &&
			(approxEq(f.W, drawerHandleHit, 1) || approxEq(f.H, drawerHandleHit, 1)) {
			if best == nil || f.X > best.Frame.X {
				best = e
			}
		}
		for _, ch := range e.Children {
			walk(ch)
		}
	}
	walk(root)
	return best
}

func containsElement(root, target *layout.Element) bool {
	if root == nil || target == nil {
		return false
	}
	if root == target {
		return true
	}
	for _, ch := range root.Children {
		if containsElement(ch, target) {
			return true
		}
	}
	return false
}

func findDrawerViewport(panel *layout.Element) *layout.Element {
	for _, ch := range panel.Children {
		f := ch.Frame
		if ch.Clip && !approxEq(f.W, drawerHandleHit, 1) && !approxEq(f.H, drawerHandleHit, 1) {
			return ch
		}
	}
	return nil
}

func findDrawerResizeHandle(root *layout.Element) *layout.Element {
	var found *layout.Element
	var walk func(*layout.Element)
	walk = func(e *layout.Element) {
		if e == nil {
			return
		}
		f := e.Frame
		if e.OnMouse != nil &&
			(approxEq(f.W, drawerHandleHit, 1) || approxEq(f.H, drawerHandleHit, 1)) {
			found = e
		}
		for _, ch := range e.Children {
			walk(ch)
		}
	}
	walk(root)
	return found
}

func findDrawerPanel(root *layout.Element) *layout.Element {
	hostW := root.Frame.W
	hostH := root.Frame.H
	var best *layout.Element
	var walk func(*layout.Element)
	walk = func(e *layout.Element) {
		if e == nil {
			return
		}
		f := e.Frame
		isPanel := e.OnMouse != nil && f.W >= 100 &&
			(f.W < hostW-20 || f.H < hostH-20)
		if isPanel {
			if best == nil || f.W*f.H > best.Frame.W*best.Frame.H {
				best = e
			}
		}
		for _, ch := range e.Children {
			walk(ch)
		}
	}
	walk(root)
	return best
}

func findEdgeStrip(root *layout.Element) *layout.Element {
	var found *layout.Element
	var walk func(*layout.Element)
	walk = func(e *layout.Element) {
		if e == nil {
			return
		}
		if e.OnMouse != nil && approxEq(e.Frame.W, drawerEdgeStrip, 2) {
			found = e
		}
		for _, ch := range e.Children {
			walk(ch)
		}
	}
	walk(root)
	return found
}

func approxEq(a, b, tol float32) bool {
	if a > b {
		return a-b <= tol
	}
	return b-a <= tol
}
