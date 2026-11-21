package theme

import "testing"

func TestUseDensity(t *testing.T) {
	defer func() {
		UseDensity(DensityDefault)
		Use("yoga-dark")
	}()
	Use("yoga-dark")
	want, _ := Get("yoga-dark")
	if Current().Spacing != want.Spacing || Current().Metrics != want.Metrics {
		t.Fatal("default density changed the theme")
	}

	UseDensity(DensityCompact)
	th := Current()
	if th.Metrics.ControlHeight != 26 || th.Metrics.MenuItemHeight != 26 {
		t.Fatalf("compact heights %v/%v, want 26", th.Metrics.ControlHeight, th.Metrics.MenuItemHeight)
	}
	if th.Spacing.M >= want.Spacing.M || th.Spacing.XXS < 1 {
		t.Fatalf("compact spacing %+v not tightened sensibly", th.Spacing)
	}
	if th.Metrics.TreeRowHeight <= 0 || th.Metrics.TreeRowHeight >= want.Typography.Body.LineHeight+want.Spacing.S {
		t.Fatalf("compact tree row %v", th.Metrics.TreeRowHeight)
	}
	// Applying it twice scales from the theme, not from scaled values.
	UseDensity(DensityCompact)
	if Current().Metrics.ControlHeight != 26 || Current().Spacing != th.Spacing {
		t.Fatal("UseDensity compounded")
	}

	UseDensity(DensityComfortable)
	if Current().Metrics.ControlHeight != 36 || Current().Spacing.M <= want.Spacing.M {
		t.Fatalf("comfortable: %v / %+v", Current().Metrics.ControlHeight, Current().Spacing)
	}

	// It survives a theme switch, and Default restores the exact metrics.
	Use("yoga-light")
	if Current().Metrics.ControlHeight != 36 {
		t.Fatal("theme switch dropped the density")
	}
	UseDensity(DensityDefault)
	light, _ := Get("yoga-light")
	if Current().Spacing != light.Spacing || Current().Metrics != light.Metrics {
		t.Fatal("DensityDefault did not restore the theme's own values")
	}
}

// Touch density reaches 44px controls and rows, while spacing grows by at
// most a quarter so layouts still fit a phone.
func TestTouchDensity(t *testing.T) {
	defer func() {
		UseDensity(DensityDefault)
		Use("yoga-dark")
	}()
	Use("yoga-dark")
	want, _ := Get("yoga-dark")
	UseDensity(DensityTouch)
	th := Current()
	if th.Metrics.ControlHeight != 44 {
		t.Fatalf("touch ControlHeight %v, want 44", th.Metrics.ControlHeight)
	}
	if th.Metrics.MenuItemHeight < 44 || th.Metrics.TreeRowHeight < 32 {
		t.Fatalf("touch rows %v / %v, want menu rows >= 44 and tree rows >= 32",
			th.Metrics.MenuItemHeight, th.Metrics.TreeRowHeight)
	}
	if th.Spacing.M <= want.Spacing.M || th.Spacing.M > want.Spacing.M*1.25+0.5 {
		t.Fatalf("touch Spacing.M %v from %v, want more but at most 1.25x", th.Spacing.M, want.Spacing.M)
	}
}

// Native builds use a mouse, so the device suggests the default density.
func TestPreferredDensityOnDesktop(t *testing.T) {
	if got := PreferredDensity(); got != DensityDefault {
		t.Fatalf("PreferredDensity() = %v, want DensityDefault", got)
	}
}
