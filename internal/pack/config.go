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
	Name    string       `toml:"name"`
	ID      string       `toml:"id"`
	Version string       `toml:"version"`
	Main    string       `toml:"main"`
	Icon    string       `toml:"icon"`
	Window  WindowConfig `toml:"window"`
	Darwin  DarwinConfig `toml:"darwin"`
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
	Formats []string `toml:"formats"`
	DMG     DarwinDMGConfig  `toml:"dmg"`
	Sign    DarwinSignConfig `toml:"sign"`
}

// DarwinDMGConfig customizes the Finder DMG window layout.
type DarwinDMGConfig struct {
	Background       string `toml:"background"` // PNG/JPEG path
	VolumeName       string `toml:"volume_name"`
	WindowWidth      int    `toml:"window_width"`
	WindowHeight     int    `toml:"window_height"`
	IconSize         int    `toml:"icon_size"`
	AppPos           []int  `toml:"app_pos"`           // [x, y]
	ApplicationsPos  []int  `toml:"applications_pos"` // [x, y]
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
	c.Darwin.defaults(c)
}

func (d *DarwinConfig) defaults(c *Config) {
	if d.ID == "" {
		d.ID = c.ID
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

// ApplyCLIOverrides merges non-empty CLI flag values into cfg.
func (c *Config) ApplyCLIOverrides(id, formats string) {
	if id != "" {
		c.ID = id
		c.Darwin.ID = id
	}
	if formats != "" {
		parts := strings.Split(formats, ",")
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
