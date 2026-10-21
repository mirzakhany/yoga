package ui

import (
	"strings"
	"testing"

	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
)

func fitDialogHeight(t *testing.T, lines int, maxH float32) float32 {
	t.Helper()
	c := newTestCtx(t)
	c.Dialogs().Show(DialogOpts{
		Title: "Fit", Width: 360, FitContent: true, MaxHeight: maxH,
		Body: func(*Ctx) View {
			return Column(Paragraph(strings.TrimSuffix(strings.Repeat("line\n", lines), "\n"))).Padding(16)
		},
		Actions: []DialogAction{{Label: "OK", Primary: true}},
	})
	for range 3 {
		BuildFrame(c, func(*Ctx) View { return Column() }, 900, 800, &input.Mouse{X: -1, Y: -1}, &input.Keyboard{})
		c.EndFrame()
	}
	return c.Dialogs().panel.Frame.H
}

func TestDialogFitContent(t *testing.T) {
	short := fitDialogHeight(t, 2, 0)
	tall := fitDialogHeight(t, 8, 0)
	if short >= tall {
		t.Fatalf("2-line dialog %v not shorter than 8-line %v", short, tall)
	}
	if short >= 200 {
		t.Fatalf("short dialog %v should hug its content", short)
	}
	capped := fitDialogHeight(t, 200, 300)
	if capped != 300 {
		t.Fatalf("capped dialog %v, want MaxHeight 300", capped)
	}
}

func TestFormGroupedDrawsOneList(t *testing.T) {
	c := newTestCtx(t)
	items := []FormItem{
		FormSwitch("a", "A", "first", true, nil),
		FormSwitch("b", "B", "second", false, nil),
		FormHeading("More", ""),
		FormSwitch("c", "C", "third", false, nil),
	}
	count := func(grouped bool) int {
		root := BuildFrame(c, func(*Ctx) View {
			f := Form("f", items...)
			if grouped {
				f = f.Grouped()
			}
			return f
		}, 600, 600, &input.Mouse{X: -1, Y: -1}, &input.Keyboard{})
		c.EndFrame()
		return len(root.Children[0].Children)
	}
	if n := count(false); n != 4 {
		t.Fatalf("plain form: %d children, want 4 cards", n)
	}
	if n := count(true); n != 3 {
		t.Fatalf("grouped form: %d children, want list, heading, list", n)
	}
}

func TestSwitchUsesSwitchTokens(t *testing.T) {
	c := newTestCtx(t)
	th := c.Theme()
	for _, on := range []bool{false, true} {
		root := BuildFrame(c, func(*Ctx) View { return Switch("s").Check(on) }, 200, 100, &input.Mouse{X: -1, Y: -1}, &input.Keyboard{})
		var dl render.DrawList
		layout.Paint(root, &dl, c.Text())
		knob, track := th.SwitchKnobOff, th.SwitchTrackOff
		if on {
			knob, track = th.SwitchKnobOn, th.SwitchTrackOn
		}
		if !drewColor(&dl, knob) || !drewColor(&dl, track) {
			t.Fatalf("on=%v: switch did not paint its track/knob tokens", on)
		}
		c.EndFrame()
	}
}
