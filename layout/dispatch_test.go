package layout

import (
	"testing"

	"github.com/mirzakhany/yoga/input"
)

func TestDispatchSkipsWhenConsumed(t *testing.T) {
	var calls int
	leaf := New(Box().Size(40, 40))
	leaf.OnMouse = func(_ *Element, _ *input.Mouse) { calls++ }
	root := New(Box().Size(100, 100), leaf)
	root.Calculate(100, 100)

	m := &input.Mouse{X: 20, Y: 20, Pressed: true}
	m.Consumed = true
	Dispatch(root, m)
	if calls != 0 {
		t.Fatalf("consumed event should not dispatch: calls=%d", calls)
	}
}

func TestDispatchOverlaysFrontToBack(t *testing.T) {
	var order []string
	back := New(Box().Absolute(0, 0).Size(100, 100))
	back.Overlay = true
	back.OnMouse = func(_ *Element, m *input.Mouse) {
		order = append(order, "back")
		m.Consumed = true
	}
	front := New(Box().Absolute(0, 0).Size(100, 100))
	front.Overlay = true
	front.OnMouse = func(_ *Element, m *input.Mouse) {
		order = append(order, "front")
		m.Consumed = true
	}
	root := New(Box().Size(100, 100), back, front)
	root.Calculate(100, 100)

	Dispatch(root, &input.Mouse{X: 50, Y: 50, Released: true})
	if len(order) != 1 || order[0] != "front" {
		t.Fatalf("front overlay should consume first: order=%v", order)
	}
}
