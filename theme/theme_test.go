package theme

import (
	"math"
	"testing"

	"github.com/mirzakhany/yoga/render"
)

func TestNormalizeFillsYogaTokensFromLegacy(t *testing.T) {
	t0 := Theme{
		Name:       "legacy-test",
		Dark:       true,
		Background: render.RGBA8(24, 24, 29, 255),
		Panel:      render.RGBA8(33, 33, 40, 255),
		PanelAlt:   render.RGBA8(43, 43, 52, 255),
		Text:       render.RGBA8(213, 217, 227, 255),
		TextDim:    render.RGBA8(126, 131, 145, 255),
		Accent:     render.RGBA8(94, 129, 232, 255),
		AccentText: render.RGBA8(255, 255, 255, 255),
		Hover:      render.RGBA8(54, 56, 68, 255),
		Active:     render.RGBA8(70, 74, 92, 255),
		Border:     render.RGBA8(54, 56, 68, 255),
	}
	normalize(&t0)
	if colorUnset(t0.Surface) {
		t.Fatal("expected Surface from legacy Background")
	}
	if t0.Spacing.M != DefaultSpacing().M {
		t.Fatalf("spacing M = %v, want %v", t0.Spacing.M, DefaultSpacing().M)
	}
}

func TestYogaDarkIsDefault(t *testing.T) {
	t.Cleanup(func() { Use("yoga-dark") })
	Use("yoga-dark")
	cur := Current()
	if cur.Name != "yoga-dark" {
		t.Fatalf("active theme = %q, want yoga-dark", cur.Name)
	}
	if !cur.Dark {
		t.Fatal("expected dark yoga theme")
	}
	if colorUnset(cur.Accent) {
		t.Fatal("Accent should be set")
	}
	if cur.Background != cur.Surface {
		t.Fatal("legacy Background should match Surface")
	}
}

func TestThemeClone(t *testing.T) {
	src, ok := Get("yoga-dark")
	if !ok {
		t.Fatal("yoga-dark missing")
	}
	c := src.Clone()
	c.Name = "clone-test"
	c.Accent = render.RGBA8(1, 2, 3, 255)
	if src.Accent == c.Accent && src.Name == "clone-test" {
		t.Fatal("clone should not mutate source name")
	}
	if src.Name != "yoga-dark" {
		t.Fatal("source name mutated")
	}
	if src.Syntax != nil && c.Syntax != nil {
		c.Syntax[0] = render.RGBA8(9, 9, 9, 255)
		if len(c.Syntax) != len(src.Syntax) {
			t.Fatal("syntax map size mismatch")
		}
	}
}

func TestRegisterNormalizesBuiltinThemes(t *testing.T) {
	t0, ok := Get("nord")
	if !ok {
		t.Fatal("nord theme missing")
	}
	if t0.Metrics.ControlHeight != DefaultComponentMetrics().ControlHeight {
		t.Fatalf("nord metrics not normalized: %+v", t0.Metrics)
	}
	if colorUnset(t0.Surface) {
		t.Fatal("nord should have Yoga color tokens after normalize")
	}
}

func TestYogaPresetsRegistered(t *testing.T) {
	for _, name := range []string{"yoga-dark", "yoga-light", "yoga-high-contrast", "yoga-midnight"} {
		if _, ok := Get(name); !ok {
			t.Fatalf("missing preset %q", name)
		}
	}
	if _, ok := Get("fluent-dark"); ok {
		t.Fatal("fluent-dark should be removed")
	}
}

func TestRemovedThemes(t *testing.T) {
	for _, name := range []string{"dark", "light", "one-dark", "tokyo-night", "rose-pine"} {
		if _, ok := Get(name); ok {
			t.Fatalf("theme %q should be removed", name)
		}
	}
}

func TestAddedThemes(t *testing.T) {
	for _, name := range []string{
		"solarized-light", "catppuccin-latte", "everforest-dark", "everforest-light",
	} {
		if _, ok := Get(name); !ok {
			t.Fatalf("missing theme %q", name)
		}
	}
}

func TestBuiltinThemesHaveColorLadder(t *testing.T) {
	for _, name := range Names() {
		th, ok := Get(name)
		if !ok {
			t.Fatalf("theme %q missing", name)
		}
		assertColorDistinct(t, name, "Surface", th.Surface, "Chrome", th.Chrome)
		assertColorDistinct(t, name, "Chrome", th.Chrome, "ChromeMuted", th.ChromeMuted)
		assertColorDistinct(t, name, "Foreground", th.Foreground, "ForegroundMuted", th.ForegroundMuted)
		assertColorDistinct(t, name, "ForegroundMuted", th.ForegroundMuted, "ForegroundSubtle", th.ForegroundSubtle)
		assertColorDistinct(t, name, "Border", th.Border, "BorderStrong", th.BorderStrong)
		assertColorDistinct(t, name, "ListHover", th.ListHover, "ListActive", th.ListActive)
		assertColorDistinct(t, name, "Accent", th.Accent, "AccentHover", th.AccentHover)
		assertColorDistinct(t, name, "AccentHover", th.AccentHover, "AccentPressed", th.AccentPressed)
		if th.ListHover == th.Chrome {
			t.Errorf("%s: ListHover should differ from Chrome", name)
		}
	}
}

func TestHighContrastUsesAccessibilityMetrics(t *testing.T) {
	th, ok := Get("yoga-high-contrast")
	if !ok {
		t.Fatal("yoga-high-contrast missing")
	}
	def := DefaultStroke()
	if th.Stroke.Thin != def.Thick {
		t.Fatalf("high-contrast thin stroke = %v, want %v", th.Stroke.Thin, def.Thick)
	}
	if th.Radius.Small != 0 {
		t.Fatalf("high-contrast small radius = %v, want 0", th.Radius.Small)
	}
}

func TestBuiltinThemesAreDistinct(t *testing.T) {
	names := Names()
	for i := 0; i < len(names); i++ {
		if names[i] == SystemName {
			continue
		}
		a, _ := Get(names[i])
		for j := i + 1; j < len(names); j++ {
			if names[j] == SystemName {
				continue
			}
			b, _ := Get(names[j])
			if themesTooSimilar(a, b) {
				t.Errorf("themes %q and %q are too similar (surface+accent fingerprint)", a.Name, b.Name)
			}
		}
	}
}

func assertColorDistinct(t *testing.T, themeName, aLabel string, a render.Color, bLabel string, b render.Color) {
	t.Helper()
	if a == b {
		t.Errorf("%s: %s and %s are identical (%v)", themeName, aLabel, bLabel, a)
	}
}

func themesTooSimilar(a, b Theme) bool {
	// Two themes are clones if surface, chrome, and accent all match.
	return a.Surface == b.Surface && a.Chrome == b.Chrome && a.Accent == b.Accent
}

func TestSurfaceLadderDelta(t *testing.T) {
	for _, name := range Names() {
		th, _ := Get(name)
		if !surfaceLadderOK(th.Surface, th.Chrome, th.ChromeMuted) {
			t.Errorf("%s: Surface/Chrome/ChromeMuted lack visible separation", name)
		}
	}
}

func surfaceLadderOK(a, b, c render.Color) bool {
	return colorDelta(a, b) >= 8 || colorDelta(b, c) >= 8 || colorDelta(a, c) >= 8
}

func colorDelta(a, b render.Color) int {
	dr := int(math.Abs(float64(a.R*255 - b.R*255)))
	dg := int(math.Abs(float64(a.G*255 - b.G*255)))
	db := int(math.Abs(float64(a.B*255 - b.B*255)))
	return max(dr, dg, db)
}
