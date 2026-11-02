package ui

import "testing"

func TestResponsiveReportsWidth(t *testing.T) {
	c := New(nil, NewFocusScope(), nil)
	var seen []float32
	body := func(_ *Ctx) View {
		return Row(
			Row().Width(100),
			Responsive("r", func(w float32) View { seen = append(seen, w); return Column().Grow(1) }).Grow(1),
		)
	}
	BuildFrame(c, body, 400, 100, nil, nil)
	BuildFrame(c, body, 400, 100, nil, nil)
	if len(seen) < 2 || seen[0] != 0 || seen[len(seen)-1] != 300 {
		t.Fatalf("widths seen %v, want 0 first and then 300", seen)
	}
	BuildFrame(c, body, 250, 100, nil, nil)
	BuildFrame(c, body, 250, 100, nil, nil)
	if got := seen[len(seen)-1]; got != 150 {
		t.Fatalf("after a resize the view saw %v, want 150", got)
	}
}
