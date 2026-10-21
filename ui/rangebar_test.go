package ui

import (
	"testing"

	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
)

func TestRangeBarSegment(t *testing.T) {
	c := newTestCtx(t)
	th := c.Theme()
	paint := func(n *Node) *render.DrawList {
		root := BuildFrame(c, func(*Ctx) View { return Column(n.Width(179)) }, 400, 100, &input.Mouse{X: -1, Y: -1}, &input.Keyboard{})
		c.EndFrame()
		var dl render.DrawList
		layout.Paint(root, &dl, c.Text())
		return &dl
	}
	dl := paint(RangeBar("ttfb", 65, 166, 179).Tone(BadgeWarning))
	minX, maxX := float32(1e9), float32(-1)
	want := [4]float32{th.Warning.R, th.Warning.G, th.Warning.B, th.Warning.A}
	for _, v := range dl.Vertices {
		if v.Col == want {
			minX, maxX = min(minX, v.Pos[0]), max(maxX, v.Pos[0])
		}
	}
	if minX < 64.5 || minX > 65.5 || maxX < 165.5 || maxX > 166.5 {
		t.Fatalf("segment spans %v..%v, want 65..166", minX, maxX)
	}
	if d := RangeBar("x", 5, 5, 100).extra.(*rangeData); func() bool { lo, hi := d.fraction(); return lo != hi }() {
		t.Fatal("empty span should have no width before the sliver")
	}
	if lo, hi := (&rangeData{start: 150, end: -10, total: 100}).fraction(); lo != 0 || hi != 1 {
		t.Fatalf("out-of-range span %v..%v, want clamped 0..1", lo, hi)
	}
	if !drewColor(paint(RangeBar("h", 0, 10, 100).Hue(HueTeal)), th.Categorical[HueTeal]) {
		t.Fatal("Hue did not color the segment")
	}
}
