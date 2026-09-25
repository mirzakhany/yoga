package pack

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/BurntSushi/toml"
)

// Config is the optional yoga.toml at the app root.
type Config struct {
	Name    string `toml:"name"`
	ID      string `toml:"id"`
	Version string `toml:"version"`
	Main    string `toml:"main"`
	Icon    string `toml:"icon"`
	// Artifact names packaged files, without extension. Placeholders:
	// {name}, {version}, {os} (GOOS), {platform} (macos|linux|windows|web)
	// and {arch}. Default "{name}-{version}-{os}-{arch}"; darwin keeps
	// "{name}-{version}" unless set.
	Artifact string        `toml:"artifact"`
	Build    BuildConfig   `toml:"build"`
	Window   WindowConfig  `toml:"window"`
	Darwin   DarwinConfig  `toml:"darwin"`
	Linux    LinuxConfig   `toml:"linux"`
	Windows  WindowsConfig `toml:"windows"`
}

// BuildConfig holds extra `go build` inputs shared by every target.
type BuildConfig struct {
	// Ldflags is passed to -ldflags; {version} expands to the app version.
	Ldflags string `toml:"ldflags"`
	// Tags is passed to -tags.
	Tags []string `toml:"tags"`
	// Flags are extra `go build` arguments, e.g. ["-trimpath"].
	Flags []string `toml:"flags"`
}

// LinuxConfig holds Linux packaging options.
type LinuxConfig struct {
	// Binary is the executable name inside the archive (defaults to name).
	Binary string `toml:"binary"`
	// Format is the archive format: "tar.gz" (default) or "tar.xz".
	Format string `toml:"format"`
	// Files maps an archive path to a source file or directory (relative to
	// the app dir). When set, the archive holds the binary plus exactly
	// these files, and no generated .desktop entry or icon.
	Files map[string]string `toml:"files"`
}

// WindowsConfig holds Windows build and packaging options.
type WindowsConfig struct {
	// Console keeps the console subsystem; by default the exe is linked
	// with -H windowsgui so no terminal window opens next to the app.
	Console bool `toml:"console"`
	// Company and Description fill the exe's version resource.
	Company     string `toml:"company"`
	Description string `toml:"description"`
	Copyright   string `toml:"copyright"`
	// Files maps a zip path to a source file or directory (relative to the
	// app dir), added next to the exe.
	Files map[string]string `toml:"files"`
}

// WindowConfig is optional window metadata (used by web shell title, docs).
type WindowConfig struct {
	Title  string `toml:"title"`
	Width  int    `toml:"width"`
	Height int    `toml:"height"`
}

// DarwinConfig holds macOS packaging metadata and artifact options.
type DarwinConfig struct {
	// ID overrides top-level id for CFBundleIdentifier when set.
	ID string `toml:"id"`
	// Icon overrides the top-level icon for the app bundle, e.g. a
	// hand-made .icns.
	Icon string `toml:"icon"`
	// DisplayName is CFBundleDisplayName (defaults to name).
	DisplayName string `toml:"display_name"`
	// Copyright is NSHumanReadableCopyright.
	Copyright string `toml:"copyright"`
	// Category is LSApplicationCategoryType, e.g. public.app-category.productivity.
	Category string `toml:"category"`
	// MinSystem is LSMinimumSystemVersion (default 11.0).
	MinSystem string `toml:"min_system"`
	// BundleVersion is CFBundleVersion build number (defaults to version).
	BundleVersion string `toml:"bundle_version"`
	// Formats lists artifacts to produce: "app", "dmg", "pkg" (default dmg; app always built).
	Formats []string         `toml:"formats"`
	DMG     DarwinDMGConfig  `toml:"dmg"`
	Sign    DarwinSignConfig `toml:"sign"`
	// Notarize submits the signed app to Apple's notary service and staples
	// the ticket before the DMG is made. Credentials come from the
	// environment: APPLE_ID, APPLE_TEAM_ID and APPLE_APP_SPECIFIC_PASSWORD,
	// or NOTARY_KEYCHAIN_PROFILE for a stored notarytool profile.
	Notarize bool `toml:"notarize"`
}

// DarwinDMGConfig customizes the Finder DMG window layout.
type DarwinDMGConfig struct {
	Background      string `toml:"background"` // PNG/JPEG path
	VolumeName      string `toml:"volume_name"`
	VolumeIcon      string `toml:"volume_icon"` // .icns shown for the mounted volume
	WindowPos       []int  `toml:"window_pos"`  // [x, y] of the Finder window
	WindowWidth     int    `toml:"window_width"`
	WindowHeight    int    `toml:"window_height"`
	IconSize        int    `toml:"icon_size"`
	AppPos          []int  `toml:"app_pos"`          // [x, y]
	ApplicationsPos []int  `toml:"applications_pos"` // [x, y]
}

// DarwinSignConfig holds codesign / productbuild identities for distribution.
type DarwinSignConfig struct {
	// Identity codesigns the .app (Developer ID Application or 3rd Party Mac Developer Application).
	Identity string `toml:"identity"`
	// InstallerIdentity signs the .pkg (Developer ID Installer or 3rd Party Mac Developer Installer).
	InstallerIdentity string `toml:"installer_identity"`
	// Entitlements is a path to an entitlements plist (optional).
	Entitlements string `toml:"entitlements"`
}

// Defaults fills missing fields from the working directory / module.
func (c *Config) Defaults(workDir string) {
	if c.Name == "" {
		c.Name = filepath.Base(workDir)
		if c.Name == "." || c.Name == "/" || c.Name == "" {
			c.Name = "App"
		}
	}
	if c.ID == "" {
		c.ID = "com.example." + sanitizeID(c.Name)
	}
	if c.Version == "" {
		c.Version = "0.1.0"
	}
	if c.Main == "" {
		c.Main = "."
	}
	if c.Window.Title == "" {
		c.Window.Title = c.Name
	}
	if c.Linux.Binary == "" {
		c.Linux.Binary = c.Name
	}
	if c.Linux.Format == "" {
		c.Linux.Format = "tar.gz"
	}
	c.Darwin.defaults(c)
}

func (d *DarwinConfig) defaults(c *Config) {
	if d.ID == "" {
		d.ID = c.ID
	}
	if d.Icon == "" {
		d.Icon = c.Icon
	}
	if d.DisplayName == "" {
		d.DisplayName = c.Name
	}
	if d.MinSystem == "" {
		d.MinSystem = "11.0"
	}
	if d.BundleVersion == "" {
		d.BundleVersion = c.Version
	}
	if len(d.Formats) == 0 {
		d.Formats = []string{"dmg"}
	}
	if d.DMG.VolumeName == "" {
		d.DMG.VolumeName = c.Name
	}
	if d.DMG.WindowWidth <= 0 {
		d.DMG.WindowWidth = 660
	}
	if d.DMG.WindowHeight <= 0 {
		d.DMG.WindowHeight = 400
	}
	if d.DMG.IconSize <= 0 {
		d.DMG.IconSize = 128
	}
	if len(d.DMG.AppPos) < 2 {
		d.DMG.AppPos = []int{180, 200}
	}
	if len(d.DMG.ApplicationsPos) < 2 {
		d.DMG.ApplicationsPos = []int{480, 200}
	}
	if len(d.DMG.WindowPos) < 2 {
		d.DMG.WindowPos = []int{100, 100}
	}
}

// BundleID returns the effective CFBundleIdentifier.
func (c Config) BundleID() string {
	if c.Darwin.ID != "" {
		return c.Darwin.ID
	}
	return c.ID
}

// WantDarwinFormat reports whether formats includes name (case-insensitive).
func (c Config) WantDarwinFormat(name string) bool {
	name = strings.ToLower(name)
	for _, f := range c.Darwin.Formats {
		if strings.ToLower(strings.TrimSpace(f)) == name {
			return true
		}
	}
	// "app" is always produced as an intermediate; treat as wanted if only pkg/dmg listed.
	if name == "app" {
		return true
	}
	return false
}

func sanitizeID(s string) string {
	s = strings.ToLower(s)
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	out := b.String()
	if out == "" {
		return "app"
	}
	return out
}

// LoadConfig reads yoga.toml from dir if present.
func LoadConfig(dir string) (Config, error) {
	var cfg Config
	path := filepath.Join(dir, "yoga.toml")
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			cfg.Defaults(dir)
			return cfg, nil
		}
		return cfg, err
	}
	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		return cfg, fmt.Errorf("yoga.toml: %w", err)
	}
	cfg.Defaults(dir)
	return cfg, nil
}

// Overrides are CLI flag values; empty fields leave the config alone.
type Overrides struct {
	ID            string
	Formats       string // comma-separated darwin formats
	Version       string // a leading "v" is dropped, so a git tag works
	BuildNumber   string // CFBundleVersion
	Ldflags       string // appended to [build] ldflags
	Sign          string
	InstallerSign string
	Entitlements  string
	Notarize      bool
}

// ApplyCLIOverrides merges non-empty CLI flag values into cfg.
func (c *Config) ApplyCLIOverrides(o Overrides) {
	if o.ID != "" {
		c.ID = o.ID
		c.Darwin.ID = o.ID
	}
	if o.Formats != "" {
		parts := strings.Split(o.Formats, ",")
		out := make([]string, 0, len(parts))
		for _, p := range parts {
			p = strings.TrimSpace(strings.ToLower(p))
			if p != "" {
				out = append(out, p)
			}
		}
		if len(out) > 0 {
			c.Darwin.Formats = out
		}
	}
	if v := strings.TrimPrefix(o.Version, "v"); v != "" {
		// bundle_version followed version unless it was set on its own.
		if c.Darwin.BundleVersion == c.Version {
			c.Darwin.BundleVersion = v
		}
		c.Version = v
	}
	if o.BuildNumber != "" {
		c.Darwin.BundleVersion = o.BuildNumber
	}
	if o.Ldflags != "" {
		c.Build.Ldflags = strings.TrimSpace(c.Build.Ldflags + " " + o.Ldflags)
	}
	if o.Sign != "" {
		c.Darwin.Sign.Identity = o.Sign
	}
	if o.InstallerSign != "" {
		c.Darwin.Sign.InstallerIdentity = o.InstallerSign
	}
	if o.Entitlements != "" {
		c.Darwin.Sign.Entitlements = o.Entitlements
	}
	if o.Notarize {
		c.Darwin.Notarize = true
	}
}

// ArtifactName expands the artifact template for a target and arch.
func (c Config) ArtifactName(target, arch string) string {
	tmpl := c.Artifact
	if tmpl == "" {
		tmpl = "{name}-{version}-{os}-{arch}"
		if target == "darwin" {
			tmpl = "{name}-{version}"
		}
	}
	platform := target
	if target == "darwin" {
		platform = "macos"
	}
	name := strings.ReplaceAll(c.Name, " ", "-")
	return strings.NewReplacer(
		"{name}", name,
		"{version}", c.Version,
		"{os}", target,
		"{platform}", platform,
		"{arch}", arch,
	).Replace(tmpl)
}

// TargetOS normalizes -os values.
func TargetOS(osFlag string) string {
	if osFlag == "" || osFlag == "host" {
		return runtime.GOOS
	}
	switch osFlag {
	case "web", "js", "wasm":
		return "web"
	case "darwin", "macos", "mac":
		return "darwin"
	case "linux":
		return "linux"
	case "windows", "win":
		return "windows"
	default:
		return osFlag
	}
}
