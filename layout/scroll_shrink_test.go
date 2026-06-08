package layout

import "testing"

func TestScrollContentDoesNotShrink(t *testing.T) {
	tall := New(Box().H(40))
	sections := New(Box().Direction(Column).Gap(8))
	for i := 0; i < 10; i++ {
		sections.Children = append(sections.Children, tall)
	}
	sections.Style.Shrink = 0
	viewport := New(Box().FlexGrow(1), sections)
	layoutRoot(viewport, 400, 200)

	// 10 blocks of 40 + 9 gaps of 8 = 472
	want := float32(472)
	if !approx(sections.Frame.H, want) {
		t.Fatalf("sections height: got %v want %v (content was flex-shrunk)", sections.Frame.H, want)
	}
	if !approx(tall.Frame.H, 40) {
		t.Fatalf("child height: got %v want 40", tall.Frame.H)
	}
}
