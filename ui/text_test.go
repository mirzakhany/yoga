package ui

import (
	"testing"

	"github.com/mirzakhany/yoga/shape"
)

func TestTextPaddingLeftInsetsPaintBox(t *testing.T) {
	text, err := shape.NewEngine(1, false)
	if err != nil {
		t.Fatal(err)
	}
	SetFrameResources(text, nil, nil)

	c := New(text, NewFocusScope(), nil)
	c.BeginFrame(400, 100, nil, nil)

	plain := Text("Hi").Layout(c)
	padded := Text("Hi").PaddingLeft(16).Layout(c)

	if padded.Style.Padding.Left != 16 {
		t.Fatalf("padding left: got %v want 16", padded.Style.Padding.Left)
	}
	// Border-box grows by padding; content (glyph) width stays the same.
	wantW := plain.Style.Width + 16
	if padded.Style.Width != wantW {
		t.Fatalf("border width: got %v want %v (plain=%v)", padded.Style.Width, wantW, plain.Style.Width)
	}
	if padded.Style.Height != plain.Style.Height {
		t.Fatalf("height should match unpadded glyph height: got %v want %v", padded.Style.Height, plain.Style.Height)
	}
}

func TestTextPaddingInsideRow(t *testing.T) {
	text, err := shape.NewEngine(1, false)
	if err != nil {
		t.Fatal(err)
	}
	SetFrameResources(text, nil, nil)

	c := New(text, NewFocusScope(), nil)
	root := Column(
		Caption("Content").PaddingLeft(12),
	).Layout(c)
	root.Calculate(200, 100)

	cap := root.Children[0]
	if cap.Frame.X != 0 {
		t.Fatalf("caption frame x: got %v want 0", cap.Frame.X)
	}
	if cap.Style.Padding.Left != 12 {
		t.Fatalf("caption padding: got %v want 12", cap.Style.Padding.Left)
	}
	// Paint reads Frame.X + Padding.Left; ensure layout reserved the inset.
	if cap.Frame.W <= 12 {
		t.Fatalf("caption frame too narrow for padding+glyph: %+v", cap.Frame)
	}
}

func TestSpecPaddingLeftPreservesOtherEdges(t *testing.T) {
	s := Spec{}.Padding(8).PaddingLeft(16)
	if s.pad.Left != 16 || s.pad.Right != 8 || s.pad.Top != 8 || s.pad.Bottom != 8 {
		t.Fatalf("edges: %+v want L16 R8 T8 B8", s.pad)
	}
	s2 := Spec{}.PaddingXY(10, 4)
	if s2.pad.Left != 10 || s2.pad.Top != 4 {
		t.Fatalf("PaddingXY: %+v", s2.pad)
	}
}
