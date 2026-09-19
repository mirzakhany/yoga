package layout

import "testing"

func TestFlexGrowStopsAtMaxWidth(t *testing.T) {
	capped := New(Box().FlexGrow(1).Max(300, nan))
	root := New(Box().Direction(Row).JustifyContent(JustifyCenter), capped)
	layoutRoot(root, 800, 100)

	if !approx(capped.Frame.W, 300) {
		t.Fatalf("width: got %v want 300", capped.Frame.W)
	}
	if !approx(capped.Frame.X, 250) {
		t.Fatalf("x: got %v want 250 (centered)", capped.Frame.X)
	}
}

func TestFlexGrowGivesCappedSpaceToOthers(t *testing.T) {
	capped := New(Box().FlexGrow(1).Max(100, nan))
	free := New(Box().FlexGrow(1))
	root := New(Box().Direction(Row), capped, free)
	layoutRoot(root, 600, 100)

	if !approx(capped.Frame.W, 100) {
		t.Fatalf("capped width: got %v want 100", capped.Frame.W)
	}
	if !approx(free.Frame.W, 500) {
		t.Fatalf("free width: got %v want 500", free.Frame.W)
	}
}

func TestFlexStretchStopsAtMaxWidth(t *testing.T) {
	capped := New(Box().H(40).Max(300, nan))
	root := New(Box(), capped)
	layoutRoot(root, 800, 200)

	if !approx(capped.Frame.W, 300) {
		t.Fatalf("width: got %v want 300", capped.Frame.W)
	}
}

func TestFlexGrowNarrowerThanMax(t *testing.T) {
	capped := New(Box().FlexGrow(1).Max(900, nan))
	root := New(Box().Direction(Row), capped)
	layoutRoot(root, 500, 100)

	if !approx(capped.Frame.W, 500) {
		t.Fatalf("width: got %v want 500", capped.Frame.W)
	}
}
