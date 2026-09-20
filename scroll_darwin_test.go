package yoga

import (
	"math"
	"testing"
	"time"

	"github.com/mirzakhany/yoga/ui"
)

// pixels converts a normalized wheel delta back into the pixels the ui layer
// will scroll for it.
func pixels(units float32) float64 { return float64(units) * ui.WheelPixelsPerUnit }

func TestScrollNormalizeWheelNotch(t *testing.T) {
	var s scrollNormalizer
	_, dy := s.normalize(0, -2, time.Now())
	if dy != -2 {
		t.Fatalf("notch delta = %v, want -2 (unchanged)", dy)
	}
}

func TestScrollNormalizeTrackpadIsOneToOne(t *testing.T) {
	var s scrollNormalizer
	// GLFW hands back pixels/10 for precise deltas: 24px of finger travel.
	_, dy := s.normalize(0, 2.4, time.Now())
	if got := pixels(dy); math.Abs(got-24) > 0.01 {
		t.Fatalf("trackpad delta scrolled %vpx, want 24px", got)
	}
}

// A precise stream can land on a whole number mid-swing; without the latch that
// one event would scroll 4.2x the rest.
func TestScrollNormalizePreciseLatchHoldsOnWholeNumbers(t *testing.T) {
	var s scrollNormalizer
	now := time.Now()
	s.normalize(0, 0.5, now)
	now = now.Add(8 * time.Millisecond)
	_, dy := s.normalize(0, 1, now)
	if got := pixels(dy); math.Abs(got-10) > 0.01 {
		t.Fatalf("whole-number delta mid-gesture scrolled %vpx, want 10px", got)
	}
}

func TestScrollNormalizeIdleEndsGesture(t *testing.T) {
	var s scrollNormalizer
	now := time.Now()
	s.normalize(0, 0.5, now)
	now = now.Add(gestureIdle + time.Millisecond)
	_, dy := s.normalize(0, 1, now)
	if dy != 1 {
		t.Fatalf("notch after idle = %v, want 1 (unchanged)", dy)
	}
}
