package ui

import (
	"testing"
	"time"
)

func TestSpinnerLayoutSchedulesAnimate(t *testing.T) {
	c := New(nil, NewFocusScope(), nil)
	c.BeginFrame(100, 100, nil, nil)

	s := NewSpinner(24)
	before := s.angle
	if el := s.Layout(c); el != s.El {
		t.Fatal("Layout should return the spinner element")
	}
	if s.angle == before {
		t.Fatal("Layout should advance the spinner angle")
	}
	d, ok := c.AnimationWait()
	if !ok || d != 16*time.Millisecond {
		t.Fatalf("AnimationWait = (%v, %v), want (16ms, true)", d, ok)
	}
}

func TestToastLayoutSelfRegistersOverlay(t *testing.T) {
	c := New(nil, NewFocusScope(), nil)
	c.BeginFrame(100, 100, nil, nil)

	th := NewToastHost()
	if el := th.Layout(c); el != th.El {
		t.Fatal("Layout should return the toast overlay element")
	}
	if got := len(c.Overlays()); got != 1 {
		t.Fatalf("toast should self-register as overlay, got %d", got)
	}
}
