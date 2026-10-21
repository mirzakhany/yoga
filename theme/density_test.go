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
