package ui

import (
	"testing"

	"github.com/mirzakhany/yoga/layout"
)

// Setting one corner or side must leave the others as they were. Each setter
// used to claim all four, so the last call zeroed the rest.
func TestPerCornerRadiiCombine(t *testing.T) {
	c := New(nil, NewFocusScope(), nil)
	c.BeginFrame(400, 300, nil, nil)
	n := Row().Background(TokenChrome).RadiusTopLeft(7).RadiusTopRight(7)
	n.RadiusBottomLeft(0).RadiusBottomRight(0)
	want := layout.Corners{TopLeft: 7, TopRight: 7}
	if got := n.Layout(c).Style.Radii; got != want {
		t.Fatalf("radii = %+v, want %+v", got, want)
	}

	// A corner set after a uniform radius overrides only that corner.
	n = Row().Radius(4).RadiusBottomLeft(0)
	want = layout.Corners{TopLeft: 4, TopRight: 4, BottomRight: 4}
	if got := n.Layout(c).Style.Radii; got != want {
		t.Fatalf("radii = %+v, want %+v", got, want)
	}
}

func TestPerSideBordersCombine(t *testing.T) {
	c := New(nil, NewFocusScope(), nil)
	c.BeginFrame(400, 300, nil, nil)
	n := Row().BorderTop(TokenBorder, 1).BorderBottom(TokenBorder, 2)
	want := layout.Edges{Top: 1, Bottom: 2}
	if got := n.Layout(c).Style.BorderWidths; got != want {
		t.Fatalf("border widths = %+v, want %+v", got, want)
	}

	// The same holds within one Spec.
	s := Spec{}.BorderLeft(TokenBorder, 1).BorderRight(TokenBorder, 3).RadiusTopLeft(5).RadiusBottomRight(6)
	r := s.resolve(c.Theme(), interactState{})
	if r.borderW != (layout.Edges{Left: 1, Right: 3}) || r.radii != (layout.Corners{TopLeft: 5, BottomRight: 6}) {
		t.Fatalf("resolved border %+v radii %+v", r.borderW, r.radii)
	}
}
