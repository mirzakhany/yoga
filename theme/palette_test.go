package theme

import (
	"testing"

	"github.com/yogaui/yoga/render"
)

// Palettes a theme author might write, one per side.
func testPalettes() []Palette {
	return []Palette{
		{
			Name: "palette-test-dark", Dark: true,
			Neutral: Ramp{
				hex(0x0F1013), hex(0x131418), hex(0x1B1D23), hex(0x23252C),
				hex(0x2B2E36), hex(0x363943), hex(0x434651), hex(0x50545F),
				hex(0x7A7F8C), hex(0x9EA3AF), hex(0xC3C7D0), hex(0xE1E4EA),
			},
			Accent:  hex(0x3567D6),
			Error:   hex(0xE5484D),
			Warning: hex(0xF5B94A),
			Success: hex(0x46B864),
		},
		{
			Name: "palette-test-light",
			Neutral: Ramp{
				hex(0xE2E4E9), hex(0xFFFFFF), hex(0xF0F1F4), hex(0xE7E9ED),
				hex(0xFFFFFF), hex(0xD1D4DB), hex(0xDFE1E6), hex(0xC9CDD6),
				hex(0x6F7480), hex(0x5D6370), hex(0x3F4450), hex(0x16181D),
			},
			Accent:  hex(0x2F62D9),
			Error:   hex(0xC62A2F),
			Warning: hex(0x8A5A00),
			Success: hex(0x1E7A3C),
		},
	}
}

func hex(v uint32) render.Color { return rgb(uint8(v>>16), uint8(v>>8), uint8(v)) }

// A theme built from a palette meets the same contrast targets as the
// builtin themes, without the author picking any derived color.
func TestFromPaletteMeetsContrastTargets(t *testing.T) {
	for _, p := range testPalettes() {
		th := FromPalette(p)
		normalize(&th)
		t.Run(p.Name, func(t *testing.T) {
			for _, c := range contrastChecks(th) {
				if got := ContrastRatio(c.fg, c.bg); got < c.want {
					t.Errorf("%s: %.2f:1, want %.2f:1 (fg %s, bg %s)",
						c.label, got, c.want, hexOf(c.fg), hexOf(c.bg))
				}
			}
		})
	}
}

// Ramp steps land on their documented roles.
func TestFromPaletteMapsRampSteps(t *testing.T) {
	p := testPalettes()[0]
	th := FromPalette(p)
	n := p.Neutral
	for _, r := range []struct {
		role      string
		got, want render.Color
	}{
		{"Surface", th.Surface, n[1]},
		{"Chrome", th.Chrome, n[2]},
		{"Overlay", th.Overlay, n[4]},
		{"Border", th.Border, n[5]},
		{"ListHover", th.ListHover, n[6]},
		{"ListActive", th.ListActive, n[7]},
		{"ForegroundMuted", th.ForegroundMuted, n[9]},
		{"Foreground", th.Foreground, n[11]},
	} {
		if r.got != r.want {
			t.Errorf("%s = %s, want ramp step %s", r.role, hexOf(r.got), hexOf(r.want))
		}
	}
	if ContrastRatio(th.AccentForeground, th.AccentHover) < ContrastText ||
		ContrastRatio(th.AccentForeground, th.AccentPressed) < ContrastText {
		t.Error("derived accent states cost the label its contrast")
	}
}
