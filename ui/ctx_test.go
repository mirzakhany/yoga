package ui

import (
	"testing"
	"time"

	"github.com/mirzakhany/yoga/layout"
)

func TestAnimateAggregatesMin(t *testing.T) {
	c := New(nil, NewFocusScope(), nil)
	c.BeginFrame(100, 100, nil, nil)

	if _, ok := c.AnimationWait(); ok {
		t.Fatal("no Animate called yet: want ok=false")
	}

	c.Animate(50 * time.Millisecond)
	c.Animate(10 * time.Millisecond)
	c.Animate(30 * time.Millisecond)

	d, ok := c.AnimationWait()
	if !ok || d != 10*time.Millisecond {
		t.Fatalf("AnimationWait = (%v, %v), want (10ms, true)", d, ok)
	}
}

func TestBeginFrameResetsAnimateAndOverlays(t *testing.T) {
	c := New(nil, NewFocusScope(), nil)
	c.BeginFrame(100, 100, nil, nil)
	c.Animate(20 * time.Millisecond)
	c.Overlay(layout.New(layout.Box()))
	if len(c.Overlays()) != 1 {
		t.Fatalf("overlays = %d, want 1", len(c.Overlays()))
	}

	c.BeginFrame(100, 100, nil, nil)
	if _, ok := c.AnimationWait(); ok {
		t.Fatal("BeginFrame should clear Animate")
	}
	if len(c.Overlays()) != 0 {
		t.Fatalf("BeginFrame should clear overlays, got %d", len(c.Overlays()))
	}
}

func TestInvalidatePostsWhenSet(t *testing.T) {
	posted := 0
	c := New(nil, NewFocusScope(), func() { posted++ })
	c.ClearNeedsPaint()
	c.Invalidate()
	c.Invalidate()
	if posted != 2 {
		t.Fatalf("post called %d times, want 2", posted)
	}
	c.BeginFrame(100, 100, nil, nil)
	if !c.NeedsPaint() {
		t.Fatal("Invalidate should mark the next frame for paint")
	}

	// nil post must not panic.
	New(nil, NewFocusScope(), nil).Invalidate()
}

// An Invalidate that lands between build and present must survive that
// frame's ClearNeedsPaint so the next wake (which sees the new state) paints.
func TestInvalidateMidFrameSurvivesClear(t *testing.T) {
	c := New(nil, NewFocusScope(), nil)
	c.BeginFrame(100, 100, nil, nil)
	c.Invalidate() // e.g. a goroutine finishing while frame N runs
	c.ClearNeedsPaint()
	c.EndFrame()

	c.BeginFrame(100, 100, nil, nil)
	if !c.NeedsPaint() {
		t.Fatal("mid-frame Invalidate was dropped by ClearNeedsPaint")
	}
	c.ClearNeedsPaint()
	c.EndFrame()

	c.BeginFrame(100, 100, nil, nil)
	if c.NeedsPaint() {
		t.Fatal("Invalidate should paint once, not on every later frame")
	}
}

func TestInvalidateFromGoroutine(t *testing.T) {
	c := New(nil, NewFocusScope(), func() {})
	c.ClearNeedsPaint()
	done := make(chan struct{})
	go func() {
		c.Invalidate()
		close(done)
	}()
	for range 100 {
		c.BeginFrame(100, 100, nil, nil)
		_ = c.NeedsPaint()
		c.ClearNeedsPaint()
		c.EndFrame()
	}
	<-done
	c.BeginFrame(100, 100, nil, nil)
	_ = c.NeedsPaint() // the race detector checks the flag handoff
}

func TestFramePaintPlan(t *testing.T) {
	tests := []struct {
		needs, dirty bool
		paint, rebuild bool
	}{
		{false, false, false, false},
		{false, true, false, false},
		{true, false, true, false},
		{true, true, true, true},
	}
	for _, tt := range tests {
		paint, rebuild := FramePaintPlan(tt.needs, tt.dirty)
		if paint != tt.paint || rebuild != tt.rebuild {
			t.Fatalf("FramePaintPlan(%v,%v)=(%v,%v) want (%v,%v)",
				tt.needs, tt.dirty, paint, rebuild, tt.paint, tt.rebuild)
		}
	}
}

func TestMarkNeedsPaintInputPhase(t *testing.T) {
	c := New(nil, NewFocusScope(), nil)
	c.ClearNeedsPaint()
	c.MarkNeedsPaint()
	if !c.NeedsPaint() || c.InputDirty() {
		t.Fatalf("outside input phase: needs=%v dirty=%v", c.NeedsPaint(), c.InputDirty())
	}
	c.ClearNeedsPaint()
	c.BeginInputPhase()
	c.MarkNeedsPaint()
	c.EndInputPhase()
	if !c.NeedsPaint() || !c.InputDirty() {
		t.Fatalf("inside input phase: needs=%v dirty=%v", c.NeedsPaint(), c.InputDirty())
	}
}

func TestTrackHoverMarksPaint(t *testing.T) {
	c := New(nil, NewFocusScope(), nil)
	c.ClearNeedsPaint()
	c.BeginInputPhase()
	hovered := false
	trackHover(c, &hovered, true)
	if !hovered || !c.NeedsPaint() || !c.InputDirty() {
		t.Fatalf("hover enter: hovered=%v needs=%v dirty=%v", hovered, c.NeedsPaint(), c.InputDirty())
	}
	c.ClearNeedsPaint()
	c.BeginInputPhase()
	trackHover(c, &hovered, true)
	if c.NeedsPaint() {
		t.Fatal("same hover should not MarkNeedsPaint")
	}
	c.EndInputPhase()
}
