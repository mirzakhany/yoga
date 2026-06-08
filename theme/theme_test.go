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
