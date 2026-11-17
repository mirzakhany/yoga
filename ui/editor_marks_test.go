package ui

import (
	"testing"
	"time"

	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/render"
)

func TestLineMarkHoverAndClick(t *testing.T) {
	h := newFindHarness(t, "a\nb\nc\nd\n")
	ed := h.ed
	ed.SetLineMarks(map[int]LineMarkKind{2: LineModified})
	var got []int
	ed.OnLineMark = func(line int) { got = append(got, line) }
	h.frame(nil, nil)
	if ed.gutterW == 0 {
		t.Skip("no gutter")
	}
	r, ok := ed.LineRect(2)
	if !ok || r.H != ed.lineH {
		t.Fatalf("LineRect(2) = %+v %v", r, ok)
	}
	x, y := ed.markX()+1, r.Y+r.H/2

	// Resting on the bar fires once after the delay; a line without a bar
	// does not.
	h.frame(&input.Mouse{X: x, Y: y}, nil)
	ed.marks.since = ed.marks.since.Add(-time.Second)
	h.frame(&input.Mouse{X: x, Y: y}, nil)
	h.frame(&input.Mouse{X: x, Y: y}, nil)
	if len(got) != 1 || got[0] != 2 {
		t.Fatalf("hover: %v", got)
	}
	h.frame(&input.Mouse{X: x, Y: y - ed.lineH}, nil)
	ed.marks.since = ed.marks.since.Add(-time.Second)
	h.frame(&input.Mouse{X: x, Y: y - ed.lineH}, nil)
	if len(got) != 1 {
		t.Fatalf("hover off the bar: %v", got)
	}

	h.clickMods(x, y, 0)
	if len(got) != 2 || got[1] != 2 {
		t.Fatalf("click: %v", got)
	}
	if ed.Caret() != 0 {
		t.Fatal("clicking a bar should not move the caret")
	}
}

func TestPopoverAtOpensUnderTheAnchor(t *testing.T) {
	c := New(nil, NewFocusScope(), nil)
	anchor := render.Rect{X: 100, Y: 50, W: 400, H: 20}
	closed := false
	c.BeginFrame(1000, 600, nil, nil)
	PopoverAt("pop", anchor, Text("hi")).Open(true).OnOpenChange(func(v bool) { closed = !v }).
		Width(200).Height(80).Layout(c)
	ov := c.Overlays()
	if len(ov) != 1 {
		t.Fatalf("%d overlays", len(ov))
	}
	f := ov[0].Frame
	if f.Y < anchor.Y+anchor.H || f.X != anchor.X {
		t.Fatalf("popover %+v, want under %+v", f, anchor)
	}
	m := &input.Mouse{X: 900, Y: 500, Pressed: true}
	ov[0].OnMouse(ov[0], m)
	if !closed {
		t.Fatal("a click outside should close it")
	}
}
