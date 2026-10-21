package ui

import (
	"testing"

	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
)

func drewColor(dl *render.DrawList, c render.Color) bool {
	want := [4]float32{c.R, c.G, c.B, c.A}
	for _, v := range dl.Vertices {
		if v.Col == want {
			return true
		}
	}
	return false
}

func TestFocusVisibleFollowsModality(t *testing.T) {
	c := newTestCtx(t)
	body := func(*Ctx) View {
		return Column(Button("b", Text("Save")), TextField("f", "x").Width(200))
	}
	m := &input.Mouse{X: -1, Y: -1}
	kb := &input.Keyboard{}
	frame := func() *layout.Element {
		root := BuildFrame(c, body, 400, 300, m, kb)
		c.EndFrame()
		return root
	}
	root := frame()
	fs := c.Focus()
	if fs.Modality() != ModalityKeyboard || !fs.FocusVisible() {
		t.Fatal("modality should start as keyboard")
	}

	// Click the button: it takes focus, but the ring stays hidden.
	btn := root.Children[0]
	m.X, m.Y, m.Pressed, m.Down = btn.Frame.X+4, btn.Frame.Y+4, true, true
	layout.Dispatch(root, m)
	fs.HandleMouse(m)
	m.Pressed, m.Down = false, false
	if fs.Modality() != ModalityPointer {
		t.Fatal("a click should switch to pointer modality")
	}
	bst := c.Widget("b", nil).(*buttonState)
	if !bst.focused {
		t.Fatal("click did not focus the button")
	}
	root = frame()
	th := c.Theme()
	var dl render.DrawList
	layout.Paint(root, &dl, c.Text())
	if focusVisible(bst.focused) || drewColor(&dl, th.FocusRingOn(th.Chrome)) && drewColor(&dl, th.FocusRing) {
		t.Fatal("button drew a focus ring after a click")
	}

	// Tab: keyboard modality, and the ring is back.
	kb.PressKey(input.KeyTab, 0)
	fs.Route(kb)
	kb.EndFrame()
	if !fs.FocusVisible() {
		t.Fatal("Tab should switch to keyboard modality")
	}
	// A shortcut does not count as keyboard navigation.
	fs.HandleMouse(&input.Mouse{Pressed: true, X: -5, Y: -5})
	kb.PressKey(input.KeyS, input.ModSuper)
	fs.Route(kb)
	kb.EndFrame()
	if fs.FocusVisible() {
		t.Fatal("Cmd+S should not switch to keyboard modality")
	}
}

func TestTextFieldKeepsRingOnClick(t *testing.T) {
	c := newTestCtx(t)
	tf := NewTextInput(TextFieldConfig{})
	tf.host.Frame = render.Rect{W: 200, H: 32}
	c.Focus().setModality(ModalityPointer)
	c.BeginFrame(400, 300, nil, nil)
	tf.focused = true
	var dl render.DrawList
	tf.paint(&dl, c.Text())
	if !drewColor(&dl, c.Theme().FocusRing) {
		t.Fatal("a text field should show focus after a click")
	}
}

func TestSpecFocusVisibleCondition(t *testing.T) {
	c := newTestCtx(t)
	c.BeginFrame(400, 300, nil, nil)
	th := c.Theme()
	s := Spec{}.When(FocusVisible, Background(TokenAccent))
	if r := s.resolve(th, interactStateFor(false, false, false, true)); !r.hasBg {
		t.Fatal("FocusVisible patch not applied under keyboard modality")
	}
	c.Focus().setModality(ModalityPointer)
	if r := s.resolve(th, interactStateFor(false, false, false, true)); r.hasBg {
		t.Fatal("FocusVisible patch applied under pointer modality")
	}
}
