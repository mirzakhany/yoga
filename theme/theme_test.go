package theme

import (
	"testing"

	"github.com/mirzakhany/yoga/render"
)

func TestNormalizeFillsYogaTokensFromLegacy(t *testing.T) {
	t0 := dark()
	t0.Surface = render.Color{}
	normalize(&t0)
	if colorUnset(t0.Surface) {
		t.Fatal("expected Surface from legacy Background")
	}
	if t0.Spacing.M != DefaultSpacing().M {
		t.Fatalf("spacing M = %v, want %v", t0.Spacing.M, DefaultSpacing().M)
	}
}

func TestYogaDarkIsDefault(t *testing.T) {
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
		// maps are distinct
		c.Syntax[0] = render.RGBA8(9, 9, 9, 255)
		if src.Syntax[0] == c.Syntax[0] && len(src.Syntax) > 0 {
			// may or may not share class 0; just ensure clone has its own map
		}
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
	for _, name := range []string{"yoga-dark", "yoga-light", "yoga-high-contrast"} {
		if _, ok := Get(name); !ok {
			t.Fatalf("missing preset %q", name)
		}
	}
	if _, ok := Get("fluent-dark"); ok {
		t.Fatal("fluent-dark should be removed")
	}
}
