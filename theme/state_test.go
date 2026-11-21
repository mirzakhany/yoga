package theme

import (
	"testing"

	"github.com/mirzakhany/yoga/render"
)

// On Chrome the layers are the theme's own hover and press colors.
func TestStateLayersKeepThemeColorsOnChrome(t *testing.T) {
	for _, name := range Names() {
		th, _ := Get(name)
		if got := th.HoverOn(th.Chrome); got != th.ListHover {
			t.Errorf("%s: HoverOn(Chrome) = %s, want ListHover %s", name, hexOf(got), hexOf(th.ListHover))
		}
		if got := th.PressedOn(th.Chrome); got != th.ListActive {
			t.Errorf("%s: PressedOn(Chrome) = %s, want ListActive %s", name, hexOf(got), hexOf(th.ListActive))
		}
	}
}

// On the other surfaces rows are painted on, hover must be visible, press must
// go at least as far, and Foreground text must stay readable on both.
func TestStateLayersOnOtherSurfaces(t *testing.T) {
	const visible = 1.10
	for _, name := range Names() {
		th, _ := Get(name)
		for _, s := range []struct {
			label string
			bg    render.Color
		}{
			{"Surface", th.Surface},
			{"Overlay", th.Overlay},
		} {
			h, p := th.HoverOn(s.bg), th.PressedOn(s.bg)
			if got := ContrastRatio(h, s.bg); got < visible {
				t.Errorf("%s: HoverOn(%s) is %.3f:1 from it, want %.2f", name, s.label, got, visible)
			}
			if ContrastRatio(p, s.bg) < ContrastRatio(h, s.bg) {
				t.Errorf("%s: PressedOn(%s) is weaker than HoverOn(%s)", name, s.label, s.label)
			}
			for _, c := range []render.Color{h, p} {
				if got := ContrastRatio(th.Foreground, c); got < ContrastText {
					t.Errorf("%s: Foreground on a %s layer %s is %.2f:1", name, s.label, hexOf(c), got)
				}
			}
		}
	}
}
