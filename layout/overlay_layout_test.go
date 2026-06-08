package layout

import "testing"

func TestOverlayChildrenSkipFlexFlow(t *testing.T) {
	main := New(Box().FlexGrow(1).H(200))
	overlay := New(Box().H(22))
	overlay.Overlay = true
	root := New(Box().Direction(Column).H(200), main, overlay)
	root.Calculate(300, 200)

	if main.Frame.H < 199 {
		t.Fatalf("main pane should fill column: got h=%v", main.Frame.H)
	}
	if overlay.Frame.W != 0 || overlay.Frame.H != 0 {
		t.Fatalf("overlay should be zero-sized until positioned: frame=%v", overlay.Frame)
	}
	if overlay.Frame.Y != 0 {
		t.Fatalf("overlay should sit at origin without covering siblings: y=%v", overlay.Frame.Y)
	}
}
