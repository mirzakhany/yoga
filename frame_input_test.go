package yoga

import (
	"testing"

	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/ui"
)

// A scroll container zeroes the wheel delta it consumes, so the frame loop must
// sample it before dispatch. Reading it afterwards left every consumed scroll
// looking like an idle frame, and the view only moved when something else
// (a hover change, an animation) happened to dirty the frame.
func TestInputImpliesPaintWhenScrollConsumed(t *testing.T) {
	c := ui.New(nil, ui.NewFocusScope(), nil)
	blocks := make([]ui.View, 0, 10)
	for i := 0; i < 10; i++ {
		blocks = append(blocks, ui.Raw(layout.New(layout.Box().H(40).FlexShrink(0))))
	}
	body := func(_ *ui.Ctx) ui.View {
		return ui.Scroll("s", ui.Column(blocks...).Gap(8)).Grow(1)
	}

	m := &input.Mouse{X: 200, Y: 100}
	m.AddScroll(-1)
	root := ui.BuildFrame(c, body, 400, 200, m, &input.Keyboard{})

	scrolled := m.ScrollX != 0 || m.ScrollY != 0 // as the frame loop samples it
	layout.Dispatch(root, m)

	if m.ScrollY != 0 {
		t.Fatalf("precondition: scroll view did not consume the wheel, ScrollY=%v", m.ScrollY)
	}
	if !inputImpliesPaint(m, scrolled, false, false) {
		t.Fatal("consumed scroll did not imply a repaint")
	}
}

func TestInputImpliesPaint(t *testing.T) {
	tests := []struct {
		name                     string
		m                        input.Mouse
		scrolled, typed, dragged bool
		want                     bool
	}{
		{name: "idle"},
		{name: "press", m: input.Mouse{Pressed: true}, want: true},
		{name: "release", m: input.Mouse{Released: true}, want: true},
		{name: "right press", m: input.Mouse{RightPressed: true}, want: true},
		{name: "right release", m: input.Mouse{RightReleased: true}, want: true},
		{name: "scroll", scrolled: true, want: true},
		{name: "typing", typed: true, want: true},
		{name: "drag", dragged: true, want: true},
		{name: "hover move only", m: input.Mouse{X: 5, Y: 5}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := tt.m
			if got := inputImpliesPaint(&m, tt.scrolled, tt.typed, tt.dragged); got != tt.want {
				t.Fatalf("inputImpliesPaint = %v, want %v", got, tt.want)
			}
		})
	}
}
