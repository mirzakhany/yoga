package pack

import "testing"

func TestArtifactName(t *testing.T) {
	var cfg Config
	cfg.Name = "My App"
	cfg.Defaults(t.TempDir())
	cfg.ApplyCLIOverrides(Overrides{Version: "v1.2.3"})

	if got := cfg.ArtifactName("linux", "amd64"); got != "My-App-1.2.3-linux-amd64" {
		t.Errorf("default linux = %q", got)
	}
	if got := cfg.ArtifactName("darwin", "arm64"); got != "My-App-1.2.3" {
		t.Errorf("default darwin = %q", got)
	}
	cfg.Artifact = "app-{platform}-v{version}-{arch}"
	if got := cfg.ArtifactName("darwin", "arm64"); got != "app-macos-v1.2.3-arm64" {
		t.Errorf("template darwin = %q", got)
	}
}

func TestVersionOverrideFollowsBundleVersion(t *testing.T) {
	var cfg Config
	cfg.Defaults(t.TempDir())
	cfg.ApplyCLIOverrides(Overrides{Version: "v2.0.0"})
	if cfg.Version != "2.0.0" || cfg.Darwin.BundleVersion != "2.0.0" {
		t.Fatalf("version %q, bundle %q", cfg.Version, cfg.Darwin.BundleVersion)
	}

	cfg.Darwin.BundleVersion = "42"
	cfg.ApplyCLIOverrides(Overrides{Version: "2.1.0"})
	if cfg.Darwin.BundleVersion != "42" {
		t.Fatalf("explicit bundle_version replaced: %q", cfg.Darwin.BundleVersion)
	}
	cfg.ApplyCLIOverrides(Overrides{BuildNumber: "202609251200"})
	if cfg.Darwin.BundleVersion != "202609251200" {
		t.Fatalf("build number: %q", cfg.Darwin.BundleVersion)
	}
}

func TestGoBuildArgs(t *testing.T) {
	var cfg Config
	cfg.Defaults(t.TempDir())
	cfg.Version = "1.0.0"
	cfg.Build.Ldflags = "-s -X main.v={version}"
	cfg.Build.Tags = []string{"a", "b"}
	cfg.Build.Flags = []string{"-trimpath"}

	got := goBuildArgs(cfg, "windows", "out.exe")
	want := []string{"build", "-trimpath", "-tags", "a,b", "-ldflags", "-s -X main.v=1.0.0 -H=windowsgui", "-o", "out.exe", "."}
	if len(got) != len(want) {
		t.Fatalf("args = %q", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("args = %q, want %q", got, want)
		}
	}

	cfg.Windows.Console = true
	if got := goBuildArgs(cfg, "windows", "out.exe"); got[5] != "-s -X main.v=1.0.0" {
		t.Fatalf("console ldflags = %q", got[5])
	}
}
