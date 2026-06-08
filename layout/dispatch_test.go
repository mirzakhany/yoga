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
