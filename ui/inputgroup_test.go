package ui

import (
	"testing"

	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
)

var groupMethods = []SelectOption{{Label: "GET", Value: "GET"}, {Label: "DELETE", Value: "DELETE"}}

func groupBody(sel int) func(*Ctx) View {
	return func(*Ctx) View {
		return Column(Row(
			InputGroup("req",
				Select("m", groupMethods).Selected(sel).FitContent(),
				TextField("url", "https://example.com").Grow(1),
				Button("send", Text("Send")).Primary(),
			).Grow(1),
		)).Grow(1)
	}
}

func findByKids(e *layout.Element, n int) *layout.Element {
	if e == nil {
		return nil
	}
	if len(e.Children) == n && e.Paint != nil {
		return e
	}
	for _, ch := range e.Children {
		if f := findByKids(ch, n); f != nil {
			return f
		}
	}
	return nil
}

func TestInputGroupJoinsParts(t *testing.T) {
	c := newTestCtx(t)
	root := BuildFrame(c, groupBody(0), 600, 200, &input.Mouse{X: -1, Y: -1}, &input.Keyboard{})
	host := findByKids(root, 4) // three parts + the frame
	if host == nil {
		t.Fatal("group host not found")
	}
	sel, field, btn := host.Children[0], host.Children[1], host.Children[2]
	if sel.Frame.X != host.Frame.X || btn.Frame.X+btn.Frame.W != host.Frame.X+host.Frame.W {
		t.Fatalf("parts do not span the group: %v %v in %v", sel.Frame, btn.Frame, host.Frame)
	}
	if field.Frame.X != sel.Frame.X+sel.Frame.W || btn.Frame.X != field.Frame.X+field.Frame.W {
		t.Fatalf("parts have gaps: %v %v %v", sel.Frame, field.Frame, btn.Frame)
	}
	if host.Frame.W != 600 {
		t.Fatalf("group width %v, want it to grow to 600", host.Frame.W)
	}

	// Nothing focused: the frame strokes the plain border.
	frame := host.Children[3]
	var dl render.DrawList
	frame.Paint(&dl, c.Text())
	th := c.Theme()
	last := dl.Vertices[len(dl.Vertices)-1].Col
	if last != [4]float32{th.Border.R, th.Border.G, th.Border.B, th.Border.A} {
		t.Fatalf("unfocused border %v, want Border", last)
	}
	// Focus the field: the group, not the field, draws the ring.
	tf := c.Widget("url", nil).(*TextInput)
	c.Focus().Focus(tf)
	dl.Reset()
	frame.Paint(&dl, c.Text())
	last = dl.Vertices[len(dl.Vertices)-1].Col
	if last != [4]float32{th.FocusRing.R, th.FocusRing.G, th.FocusRing.B, th.FocusRing.A} {
		t.Fatalf("focused border %v, want FocusRing", last)
	}
	dl.Reset()
	tf.paint(&dl, c.Text())
	for _, v := range dl.Vertices {
		if v.Col == [4]float32{th.FocusRing.R, th.FocusRing.G, th.FocusRing.B, th.FocusRing.A} {
			t.Fatal("a grouped field drew its own focus ring")
		}
	}
}

func TestSelectFitContentFollowsLabel(t *testing.T) {
	c := newTestCtx(t)
	root := BuildFrame(c, groupBody(0), 600, 200, &input.Mouse{X: -1, Y: -1}, &input.Keyboard{})
	get := findByKids(root, 4).Children[0].Frame.W
	c.EndFrame()
	root = BuildFrame(c, groupBody(1), 600, 200, &input.Mouse{X: -1, Y: -1}, &input.Keyboard{})
	del := findByKids(root, 4).Children[0].Frame.W
	if get >= del || get >= 160 {
		t.Fatalf("fit widths GET=%v DELETE=%v, want GET narrower and both content-sized", get, del)
	}
}

func TestSelectOptionColorsTintMenu(t *testing.T) {
	c := newTestCtx(t)
	green := render.RGBA8(0, 200, 0, 255)
	BuildFrame(c, func(*Ctx) View {
		return Select("m", groupMethods).OptionColors(func(o SelectOption) render.Color {
			if o.Value == "GET" {
				return green
			}
			return render.Color{}
		})
	}, 600, 200, &input.Mouse{X: -1, Y: -1}, &input.Keyboard{})
	st := c.Widget("m", nil).(*selectState)
	if got := st.menu.items[0].Color; got != green {
		t.Fatalf("GET item color %v, want %v", got, green)
	}
	if got := st.menu.items[1].Color; got.A != 0 {
		t.Fatalf("DELETE item colored %v, want none", got)
	}
}
