package yoga

import (
	"math"
	"testing"
)

func TestSetZoomClampsAndWakes(t *testing.T) {
	old := zoom
	defer func() { zoom = old }()
	woke := 0
	zoom.wake = func() { woke++ }
	gen := zoom.gen

	SetZoom(1.5)
	if Zoom() != 1.5 || zoom.gen != gen+1 || woke != 1 {
		t.Fatalf("zoom %v gen %d woke %d", Zoom(), zoom.gen-gen, woke)
	}
	SetZoom(1.5) // unchanged: no new generation
	if zoom.gen != gen+1 || woke != 1 {
		t.Fatal("setting the same zoom should not bump the generation")
	}
	SetZoom(10)
	if Zoom() != MaxZoom {
		t.Fatalf("zoom %v, want MaxZoom", Zoom())
	}
	SetZoom(0)
	if Zoom() != MinZoom {
		t.Fatalf("zoom %v, want MinZoom", Zoom())
	}
	SetZoom(float32(math.NaN()))
	if Zoom() != MinZoom {
		t.Fatal("NaN should be ignored")
	}
}
