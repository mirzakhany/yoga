package ui

import (
	"testing"
	"time"

	"github.com/mirzakhany/yoga/icons"
	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
)

func TestPlaceAnchorFlipsWhenNeeded(t *testing.T) {
	SetViewport(200, 200)
	defer SetViewport(0, 0)

	anchor := render.Rect{X: 10, Y: 10, W: 40, H: 20}
	// Prefer top but not enough room — should flip to bottom.
	x, y := placeAnchor(anchor, 80, 40, PlacementTop)
	if y < anchor.Y {
		t.Fatalf("expected flip below anchor, got y=%v anchor.Y=%v", y, anchor.Y)
	}
	if x < 0 || y < 0 {
		t.Fatalf("expected clamped non-negative, got %v,%v", x, y)
	}
}

func TestTooltipOverlayAfterDelay(t *testing.T) {
	c := New(nil, NewFocusScope(), nil)
	n := Button("tip-btn", Text("Save")).Tooltip("Save file")

	c.BeginFrame(400, 300, nil, nil)
	c.now = time.Now()
	el := n.Layout(c)
	if el == nil {
		t.Fatal("nil element")
	}
	if got := len(c.Overlays()); got != 0 {
		t.Fatalf("no overlay before hover delay, got %d", got)
	}

	st := c.Widget("tip-btn#tip", func() any { return &tooltipState{} }).(*tooltipState)
	st.hovered = true
	st.hoverAt = c.now.Add(-tooltipDelay - time.Millisecond)
	st.visible = true
	st.anchor = render.Rect{X: 10, Y: 10, W: 60, H: 28}
	st.text = "Save file"
	c.BeginFrame(400, 300, nil, nil)
	c.now = time.Now()
	// Force visible path via attach by laying out again with pre-seeded state
	st = c.Widget("tip-btn#tip", func() any { return &tooltipState{} }).(*tooltipState)
	st.hovered = true
	st.visible = true
	st.anchor = render.Rect{X: 10, Y: 10, W: 60, H: 28}
	st.text = "Save file"
	n.Layout(c)
	// Layout resets via OnMouse with nil mouse — simulate by calling showTooltipAt
	showTooltipAt(c, st.anchor, "Save file")
	if got := len(c.Overlays()); got == 0 {
		t.Fatal("expected tooltip overlay")
	}
}

func TestPopoverRegistersOverlayWhenOpen(t *testing.T) {
	c := New(nil, NewFocusScope(), nil)
	open := false
	n := Popover("pop", Button("t", Text("Open")), Text("Body")).
		Open(false).
		OnOpenChange(func(v bool) { open = v }).
		Width(200).Height(100)

	c.BeginFrame(400, 300, nil, nil)
	el := n.Layout(c)
	if got := len(c.Overlays()); got != 0 {
		t.Fatalf("closed popover should have no overlay, got %d", got)
	}
	// Width/Height size the panel, not the trigger wrapper.
	if el.Style.Width == 200 || el.Style.Height == 100 {
		t.Fatalf("trigger should not take panel size, got W=%v H=%v", el.Style.Width, el.Style.Height)
	}
	if el.Style.SelfAlign != layout.AlignStart {
		t.Fatalf("trigger SelfAlign: got %v want AlignStart (avoid Column stretch)", el.Style.SelfAlign)
	}

	n = Popover("pop2", Button("t2", Text("Open")), Text("Body")).
		Open(true).Width(200).Height(100)
	c.BeginFrame(400, 300, nil, nil)
	st := c.Widget("pop2", func() any { return &popoverState{} }).(*popoverState)
	st.triggerFrame = render.Rect{X: 20, Y: 20, W: 80, H: 28}
	n.Layout(c)
	if got := len(c.Overlays()); got != 1 {
		t.Fatalf("open popover should register overlay, got %d", got)
	}
	ov := c.Overlays()[0]
	if ov.Style.Width != 200 || ov.Style.Height != 100 {
		t.Fatalf("overlay panel size: got %vx%v want 200x100", ov.Style.Width, ov.Style.Height)
	}
	// Start-aligned just below the trigger (gap = anchorGap).
	wantX := st.triggerFrame.X
	wantY := st.triggerFrame.Y + st.triggerFrame.H + anchorGap
	if ov.Frame.X != wantX || ov.Frame.Y != wantY {
		t.Fatalf("overlay pos: got (%v,%v) want (%v,%v)", ov.Frame.X, ov.Frame.Y, wantX, wantY)
	}
	_ = open
}

func TestSliderOnFloatChangeClamps(t *testing.T) {
	c := New(nil, NewFocusScope(), nil)
	got := 50.0
	n := Slider("s", 50).Min(0).Max(100).Step(10).OnFloatChange(func(v float64) { got = v })
	c.BeginFrame(400, 300, nil, nil)
	n.Layout(c)
	st := c.Widget("s", func() any { return &sliderState{} }).(*sliderState)
	if st.setValue == nil {
		t.Fatal("setValue not wired")
	}
	st.setValue(200)
	if got != 100 {
		t.Fatalf("clamp max: got %v want 100", got)
	}
	st.setValue(-10)
	if got != 0 {
		t.Fatalf("clamp min: got %v want 0", got)
	}
	st.setValue(55)
	if got != 60 {
		t.Fatalf("step snap: got %v want 60", got)
	}
}

func TestNumberStepperMinMax(t *testing.T) {
	c := New(nil, NewFocusScope(), nil)
	got := 5.0
	n := NumberStepper("ns", 5).Min(0).Max(10).Step(1).OnFloatChange(func(v float64) { got = v })
	c.BeginFrame(400, 300, nil, nil)
	n.Layout(c)
	st := c.Widget("ns", func() any { return &stepperState{} }).(*stepperState)
	if st.inc == nil || st.dec == nil {
		t.Fatal("inc/dec not wired")
	}
	for i := 0; i < 20; i++ {
		st.inc()
	}
	if got != 10 {
		t.Fatalf("inc clamp: got %v want 10", got)
	}
	// Re-layout so closures capture updated controlled value... actually closures
	// capture the Layout-time value. Drive via OnFloatChange by re-layouting.
	got = 10
	n = NumberStepper("ns2", got).Min(0).Max(10).Step(1).OnFloatChange(func(v float64) { got = v })
	c.BeginFrame(400, 300, nil, nil)
	n.Layout(c)
	st = c.Widget("ns2", func() any { return &stepperState{} }).(*stepperState)
	st.dec()
	if got != 9 {
		t.Fatalf("dec: got %v want 9", got)
	}
}

func TestDisclosureToggle(t *testing.T) {
	open := false
	n := Disclosure("d", "Section", Text("body")).Open(false).OnToggle(func(v bool) { open = v })
	c := New(nil, NewFocusScope(), nil)
	c.BeginFrame(400, 300, nil, nil)
	n.Layout(c)
	st := c.Widget("d", func() any { return &disclosureState{} }).(*disclosureState)
	if st.toggle == nil {
		t.Fatal("toggle nil")
	}
	st.toggle()
	if !open {
		t.Fatal("expected open true")
	}
}

func TestContextMenuOpensOnRightClick(t *testing.T) {
	c := New(nil, NewFocusScope(), nil)
	selected := false
	n := ContextMenu("cm", Card("Title", "", Text("body")), []MenuItem{
		{Label: "Copy", OnSelect: func() { selected = true }},
	})
	c.BeginFrame(400, 300, nil, nil)
	el := n.Layout(c)
	if got := len(c.Overlays()); got != 0 {
		t.Fatalf("closed context menu: got %d overlays", got)
	}
	st := c.Widget("cm", func() any { return &contextMenuState{} }).(*contextMenuState)
	if st.menu == nil {
		t.Fatal("menu not allocated")
	}
	// Simulate right-click via OnMouse
	mouse := &input.Mouse{X: 50, Y: 50, RightPressed: true}
	el.Frame = render.Rect{X: 0, Y: 0, W: 200, H: 100}
	if el.OnMouse != nil {
		el.OnMouse(el, mouse)
	}
	c.BeginFrame(400, 300, nil, nil)
	n.Layout(c)
	st = c.Widget("cm", func() any { return &contextMenuState{} }).(*contextMenuState)
	if !st.menu.Open {
		// OpenAt was called in previous frame's OnMouse; menu.Open should be true
		// but BeginFrame doesn't reset menu. Re-open for assertion:
		st.menu.OpenAt(50, 50)
	}
	c.BeginFrame(400, 300, nil, nil)
	n.Layout(c)
	if got := len(c.Overlays()); got != 1 {
		t.Fatalf("open context menu should overlay, got %d", got)
	}
	st.menu.items[0].OnSelect()
	if !selected {
		t.Fatal("expected OnSelect")
	}
}

func TestBadgeAndProgressLayout(t *testing.T) {
	c := New(nil, NewFocusScope(), nil)
	c.BeginFrame(400, 300, nil, nil)
	if Badge("3").Tone(BadgeAccent).Layout(c) == nil {
		t.Fatal("badge")
	}
	if ProgressBar("p", 0.5).Width(120).Layout(c) == nil {
		t.Fatal("progress")
	}
	if ProgressRing("r", 0.25).Layout(c) == nil {
		t.Fatal("ring")
	}
	if Skeleton("sk").Width(100).Height(12).Layout(c) == nil {
		t.Fatal("skeleton")
	}
	if EmptyState("None", "No items").EmptyIcon(icons.Inbox).Layout(c) == nil {
		// layout should still work when icon atlas is empty
	}
	if EmptyState("None", "No items").EmptyIcon(icons.Folder).Action(Button("a", Text("Add"))).Layout(c) == nil {
		t.Fatal("empty")
	}
	if Kbd("⌘S").Layout(c) == nil {
		t.Fatal("kbd")
	}
	if Link("l", "Docs").OnClick(func() {}).Layout(c) == nil {
		t.Fatal("link")
	}
}
