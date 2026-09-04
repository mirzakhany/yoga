package ui

import (
	"testing"
	"time"
)

func TestSpinnerLayoutSchedulesAnimate(t *testing.T) {
	c := New(nil, NewFocusScope(), nil)
	c.BeginFrame(100, 100, nil, nil)

	n := Spinner("spin", 24)
	if el := n.Layout(c); el == nil {
		t.Fatal("Layout returned nil")
	}
	st := c.Widget("spin", func() any { return &spinnerState{} }).(*spinnerState)
	if st.angle == 0 {
		t.Fatal("Layout should advance the spinner angle")
	}
	d, ok := c.AnimationWait()
	if !ok || d != 50*time.Millisecond {
		t.Fatalf("AnimationWait = (%v, %v), want (50ms, true)", d, ok)
	}
}

func TestToastLayoutSelfRegistersOverlay(t *testing.T) {
	c := New(nil, NewFocusScope(), nil)
	c.BeginFrame(100, 100, nil, nil)

	th := NewToastHost()
	if el := th.Layout(c); el == nil {
		t.Fatal("Layout returned nil")
	}
	if got := len(c.Overlays()); got != 1 {
		t.Fatalf("toast should self-register as overlay, got %d", got)
	}
}
