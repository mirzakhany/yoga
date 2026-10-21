package ui

import (
	"testing"

	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/theme"
)

func TestBadgeHueUsesCategoricalPair(t *testing.T) {
	c := newTestCtx(t)
	c.BeginFrame(400, 300, nil, nil)
	th := c.Theme()
	for _, v := range []*Node{Badge("GET").Hue(HueGreen).Mono(), Tag("Production").Hue(HueGreen)} {
		el := v.Layout(c)
		el.Frame = render.Rect{W: el.Style.Width, H: el.Style.Height}
		var dl render.DrawList
		el.Paint(&dl, c.Text())
		if len(dl.Vertices) == 0 {
			t.Fatal("badge painted nothing")
		}
		fill := dl.Vertices[0].Col
		want := th.CategoricalSurface[theme.HueGreen]
		if fill != [4]float32{want.R, want.G, want.B, want.A} {
			t.Fatalf("fill %v, want CategoricalSurface[green] %v", fill, want)
		}
	}
	// Tone after Hue switches back to status coloring.
	if d := Badge("x").Hue(HueRed).Tone(BadgeError).extra.(*badgeData); d.hasHue {
		t.Fatal("Tone should clear Hue")
	}
}
