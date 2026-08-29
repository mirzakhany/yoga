package theme

import "testing"

func TestSystemThemeRegistered(t *testing.T) {
	if _, ok := Get(SystemName); !ok {
		t.Fatal("system theme missing from registry")
	}
}

func TestUseSystemAppliesYogaVariant(t *testing.T) {
	restore := swapPrefersDark(func() bool { return true })
	defer restore()
	t.Cleanup(func() { Use("yoga-dark") })

	if !Use(SystemName) {
		t.Fatal("Use(system) failed")
	}
	if Selected() != SystemName {
		t.Fatalf("Selected() = %q, want system", Selected())
	}
	cur := Current()
	if cur.Name != "yoga-dark" {
		t.Fatalf("resolved theme = %q, want yoga-dark", cur.Name)
	}
	if !cur.Dark {
		t.Fatal("expected dark palette")
	}
}

func TestUseSystemLightVariant(t *testing.T) {
	restore := swapPrefersDark(func() bool { return false })
	defer restore()
	t.Cleanup(func() { Use("yoga-dark") })

	if !Use(SystemName) {
		t.Fatal("Use(system) failed")
	}
	cur := Current()
	if cur.Name != "yoga-light" {
		t.Fatalf("resolved theme = %q, want yoga-light", cur.Name)
	}
	if cur.Dark {
		t.Fatal("expected light palette")
	}
}

func TestSyncSystemFollowsOSChange(t *testing.T) {
	dark := true
	restore := swapPrefersDark(func() bool { return dark })
	defer restore()
	t.Cleanup(func() { Use("yoga-dark") })

	Use(SystemName)
	if Current().Name != "yoga-dark" {
		t.Fatalf("initial = %q, want yoga-dark", Current().Name)
	}

	dark = false
	if !SyncSystem() {
		t.Fatal("SyncSystem should apply light after OS change")
	}
	if Current().Name != "yoga-light" {
		t.Fatalf("after sync = %q, want yoga-light", Current().Name)
	}
	if Selected() != SystemName {
		t.Fatalf("Selected() = %q, want system", Selected())
	}

	dark = true
	if !SyncSystem() {
		t.Fatal("SyncSystem should apply dark after OS change")
	}
	if Current().Name != "yoga-dark" {
		t.Fatalf("after second sync = %q, want yoga-dark", Current().Name)
	}
}

func TestSyncSystemNoOpWhenNotSystem(t *testing.T) {
	t.Cleanup(func() { Use("yoga-dark") })
	Use("nord")
	if SyncSystem() {
		t.Fatal("SyncSystem should not run when a fixed theme is selected")
	}
}

func swapPrefersDark(fn func() bool) func() {
	old := prefersDarkFn
	prefersDarkFn = fn
	return func() { prefersDarkFn = old }
}
