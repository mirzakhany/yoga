package ui

import (
	"testing"

	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/layout"
)

func TestTitleBarCenterIgnoresSideWidths(t *testing.T) {
	c := New(nil, NewFocusScope(), nil)
	var mid *layout.Element
	body := func(_ *Ctx) View {
		mid = layout.New(layout.Box().W(100).H(20))
		return Column(TitleBar(
			Raw(layout.New(layout.Box().W(300).H(20))),
			Spacer(),
			Raw(layout.New(layout.Box().W(40).H(20))),
		).Center(Raw(mid)), Spacer())
	}
	BuildFrame(c, body, 1000, 400, &input.Mouse{}, nil)
	if got := mid.Frame.X + mid.Frame.W/2; got < 499 || got > 501 {
		t.Fatalf("center at x=%v (frame %+v), want 500", got, mid.Frame)
	}
}
