package ui

import (
	"testing"

	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/theme"
)

func TestSpecBorderTopOverridesUniform(t *testing.T) {
	s := Spec{}.Border(TokenBorder, 2).BorderTop(TokenBorder, 4)
	if s.borderW.Top != 4 || s.borderW.Right != 2 {
		t.Fatalf("border widths: top=%v right=%v", s.borderW.Top, s.borderW.Right)
	}
}

func TestSpecRadiusTopLeftOnly(t *testing.T) {
	s := Spec{}.Radius(6).RadiusTopLeft(12)
	if s.radii.TopLeft != 12 || s.radii.TopRight != 6 {
		t.Fatalf("radii: tl=%v tr=%v", s.radii.TopLeft, s.radii.TopRight)
	}
}

func TestColumnBorderBottomLayoutStyle(t *testing.T) {
	c := New(nil, NewFocusScope(), nil)
	c.BeginFrame(200, 80, nil, nil)
	th := c.Theme()

	el := Column(Text("hello")).
		BorderBottom(TokenBorder, th.Stroke.Thin).
		RadiusTopRight(th.Radius.Large).
		Layout(c)
	if el == nil {
		t.Fatal("nil element")
	}
	if el.Style.BorderWidths.Bottom != th.Stroke.Thin {
		t.Fatalf("bottom border = %v want %v", el.Style.BorderWidths.Bottom, th.Stroke.Thin)
	}
	if el.Style.Radii.TopRight != th.Radius.Large {
		t.Fatalf("top-right radius = %v want %v", el.Style.Radii.TopRight, th.Radius.Large)
	}
}

func TestSpecBorderStyleResolve(t *testing.T) {
	s := Background(TokenChrome).Border(TokenBorder, 1).BorderStyle(BorderDotted)
	th := theme.Current()
	r := s.resolve(th, interactState{})
	if !r.hasBorderStyle || r.borderStyle != BorderDotted {
		t.Fatalf("border style = %v has=%v", r.borderStyle, r.hasBorderStyle)
	}
	if r.borderW.Top != 1 {
		t.Fatalf("border width = %v", r.borderW.Top)
	}
}

func TestApplyVisualSpecPerSideBorder(t *testing.T) {
	th := theme.Current()
	el := layout.New(layout.Box())
	s := Spec{}.BorderBottom(TokenBorder, th.Stroke.Thick).BorderStyle(BorderDotted)
	applyVisualSpec(el, s, th, interactState{})
	if el.Style.BorderWidths.Bottom != th.Stroke.Thick {
		t.Fatalf("bottom=%v", el.Style.BorderWidths.Bottom)
	}
	if el.Style.BorderStyle != BorderDotted {
		t.Fatalf("style=%v", el.Style.BorderStyle)
	}
}
