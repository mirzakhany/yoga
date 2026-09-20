package ui

import (
	"strings"
	"testing"

	"github.com/mirzakhany/yoga/shape"
)

func testEngine(t *testing.T) *shape.Engine {
	t.Helper()
	eng, err := shape.NewEngine(1, false)
	if err != nil {
		t.Fatal(err)
	}
	return eng
}

func measure(t *testing.T, eng *shape.Engine, s string) float32 {
	t.Helper()
	w, _ := eng.MeasureAtWeight(s, 14, shape.WeightRegular)
	return w
}

func TestTruncateKeepsStringThatFits(t *testing.T) {
	eng := testEngine(t)
	const s = "/tmp/api.proto"
	full := measure(t, eng, s)

	got := truncateToWidth(eng, s, 14, shape.WeightRegular, full+10, EllipsisMiddle)
	if got != s {
		t.Fatalf("got %q, want the string unchanged", got)
	}
}

func TestTruncateMiddleKeepsHeadAndTail(t *testing.T) {
	eng := testEngine(t)
	const s = "/Users/me/src/proj/vendor/google/api/annotations.proto"
	full := measure(t, eng, s)
	avail := full / 2

	got := truncateToWidth(eng, s, 14, shape.WeightRegular, avail, EllipsisMiddle)
	if got == s {
		t.Fatalf("expected %q to be shortened at %v px (full %v)", got, avail, full)
	}
	if w := measure(t, eng, got); w > avail {
		t.Fatalf("truncated %q measures %v, over the %v budget", got, w, avail)
	}
	if !strings.Contains(got, ellipsisRune) {
		t.Fatalf("got %q, want an ellipsis in it", got)
	}
	if !strings.HasPrefix(got, "/") {
		t.Fatalf("got %q, want the head kept", got)
	}
	// The tail is what identifies a path, so it must survive.
	if !strings.HasSuffix(got, "o") {
		t.Fatalf("got %q, want the tail kept", got)
	}
}

func TestTruncateEndKeepsHeadOnly(t *testing.T) {
	eng := testEngine(t)
	const s = "/Users/me/src/proj/vendor/google/api/annotations.proto"
	avail := measure(t, eng, s) / 2

	got := truncateToWidth(eng, s, 14, shape.WeightRegular, avail, EllipsisEnd)
	if !strings.HasSuffix(got, ellipsisRune) {
		t.Fatalf("got %q, want it to end in an ellipsis", got)
	}
	if !strings.HasPrefix(s, strings.TrimSuffix(got, ellipsisRune)) {
		t.Fatalf("got %q, want a prefix of the original", got)
	}
	if w := measure(t, eng, got); w > avail {
		t.Fatalf("truncated %q measures %v, over the %v budget", got, w, avail)
	}
}

func TestTruncateUsesTheWidestFittingCut(t *testing.T) {
	eng := testEngine(t)
	const s = "/Users/me/src/proj/vendor/google/api/annotations.proto"
	avail := measure(t, eng, s) / 2

	got := truncateToWidth(eng, s, 14, shape.WeightRegular, avail, EllipsisMiddle)
	// One more kept rune must not fit, or we gave up too much of the path.
	kept := len([]rune(got)) - 1
	more := elide([]rune(s), kept+1, EllipsisMiddle)
	if w := measure(t, eng, more); w <= avail {
		t.Fatalf("%q (%v px) also fits in %v; truncation was too eager", more, w, avail)
	}
}

func TestTruncateDegradesWhenNothingFits(t *testing.T) {
	eng := testEngine(t)
	const s = "/tmp/api.proto"

	if got := truncateToWidth(eng, s, 14, shape.WeightRegular, 0, EllipsisMiddle); got != "" {
		t.Fatalf("zero width: got %q, want empty", got)
	}
	// Narrower than "…" itself: nothing can be drawn.
	if got := truncateToWidth(eng, s, 14, shape.WeightRegular, 0.5, EllipsisMiddle); got != "" {
		t.Fatalf("sub-ellipsis width: got %q, want empty", got)
	}
}

func TestTruncateNoneIsAPassthrough(t *testing.T) {
	eng := testEngine(t)
	const s = "/tmp/api.proto"
	if got := truncateToWidth(eng, s, 14, shape.WeightRegular, 1, EllipsisNone); got != s {
		t.Fatalf("got %q, want the string unchanged", got)
	}
}

func TestEllipsisTextShrinksInsideNarrowRow(t *testing.T) {
	eng := testEngine(t)
	SetFrameResources(eng, nil, nil)
	c := New(eng, NewFocusScope(), nil)
	c.BeginFrame(120, 40, nil, nil)

	const s = "/Users/me/src/proj/vendor/google/api/annotations.proto"
	plain := Text(s).Layout(c)
	if plain.Style.Shrink != 0 {
		t.Fatalf("plain text shrink: got %v want 0", plain.Style.Shrink)
	}

	root := Row(Text(s).Ellipsis(EllipsisMiddle)).Layout(c)
	el := root.Children[0]
	if el.Style.Shrink != 1 {
		t.Fatalf("ellipsis text shrink: got %v want 1", el.Style.Shrink)
	}
	if el.Style.Width != plain.Style.Width {
		t.Fatalf("flex basis: got %v want the measured width %v", el.Style.Width, plain.Style.Width)
	}
	root.Calculate(120, 40)
	if el.Frame.W > 120 {
		t.Fatalf("frame width %v did not shrink into the 120px row", el.Frame.W)
	}
}
