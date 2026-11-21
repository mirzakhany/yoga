package theme

import "testing"

// The type ramp must step down in size from Display to Overline in every
// builtin theme, so each role reads as its own level.
func TestTypographyRampDescends(t *testing.T) {
	for _, name := range Names() {
		th, _ := Get(name)
		ty := th.Typography
		ramp := []struct {
			role string
			s    TypographyStyle
		}{
			{"Display", ty.Display},
			{"Headline", ty.Headline},
			{"Title", ty.Title},
			{"Subtitle", ty.Subtitle},
			{"Body", ty.Body},
			{"BodySmall", ty.BodySmall},
			{"Caption", ty.Caption},
			{"Overline", ty.Overline},
		}
		for i, r := range ramp {
			if r.s.Size <= 0 || r.s.LineHeight < r.s.Size {
				t.Errorf("%s: %s is %v/%v, want a size and a line height at least as tall", name, r.role, r.s.Size, r.s.LineHeight)
			}
			if i > 0 && r.s.Size >= ramp[i-1].s.Size {
				t.Errorf("%s: %s (%v) is not smaller than %s (%v)", name, r.role, r.s.Size, ramp[i-1].role, ramp[i-1].s.Size)
			}
		}
	}
}

// A theme that sets only the original five roles still gets the new ones.
func TestNormalizeFillsMissingTypeRoles(t *testing.T) {
	th := Theme{Name: "partial-type", Typography: Typography{
		Caption:    TypographyStyle{Size: 11, LineHeight: 15, Weight: 400},
		Body:       TypographyStyle{Size: 13, LineHeight: 18, Weight: 400},
		BodyStrong: TypographyStyle{Size: 13, LineHeight: 18, Weight: 600},
		Subtitle:   TypographyStyle{Size: 15, LineHeight: 21, Weight: 600},
		Title:      TypographyStyle{Size: 18, LineHeight: 26, Weight: 600},
	}}
	normalize(&th)
	def := DefaultTypography()
	if th.Typography.Body.Size != 13 {
		t.Fatalf("Body = %v, want the theme's own 13", th.Typography.Body.Size)
	}
	if th.Typography.Headline != def.Headline || th.Typography.Overline != def.Overline {
		t.Fatalf("new roles = %+v / %+v, want defaults", th.Typography.Headline, th.Typography.Overline)
	}
}

// Every builtin theme has transition durations, from fastest to slowest.
func TestMotionDurationsAscend(t *testing.T) {
	for _, name := range Names() {
		th, _ := Get(name)
		m := th.Motion
		if m.Fast <= 0 || m.Fast >= m.Normal || m.Normal >= m.Slow {
			t.Errorf("%s: Motion = %v / %v / %v, want 0 < Fast < Normal < Slow", name, m.Fast, m.Normal, m.Slow)
		}
	}
}

// The easings start at 0, end at 1, and stay inside that range.
func TestEasingsSpanZeroToOne(t *testing.T) {
	for name, ease := range map[string]func(float32) float32{
		"EaseEnter": EaseEnter, "EaseExit": EaseExit, "EaseStandard": EaseStandard,
	} {
		if ease(0) != 0 || ease(1) != 1 {
			t.Errorf("%s(0), (1) = %v, %v; want 0, 1", name, ease(0), ease(1))
		}
		for i := 0; i <= 20; i++ {
			if v := ease(float32(i) / 20); v < 0 || v > 1 {
				t.Errorf("%s(%v) = %v, outside [0, 1]", name, float32(i)/20, v)
			}
		}
		if ease(-1) != 0 || ease(2) != 1 {
			t.Errorf("%s does not clamp its input", name)
		}
	}
}
