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
	c.Invalidate()
	c.Invalidate()
	if posted != 2 {
		t.Fatalf("post called %d times, want 2", posted)
	}

	// nil post must not panic.
	New(nil, NewFocusScope(), nil).Invalidate()
}
