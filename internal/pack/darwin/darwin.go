package darwin

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// Options configures macOS .app / DMG / PKG packaging.
type Options struct {
	Name          string
	DisplayName   string
	ID            string
	Version       string // CFBundleShortVersionString
	BundleVersion string // CFBundleVersion
	Copyright     string
	Category      string // LSApplicationCategoryType
	MinSystem     string
	Binary        string
	Icon          string
	OutDir        string

	// Formats: subset of "dmg", "pkg" (app bundle is always written).
	Formats []string

	DMG  DMGOptions
	Sign SignOptions
}

// DMGOptions customizes the Finder window for the distributable DMG.
type DMGOptions struct {
	Background      string
	VolumeName      string
	WindowWidth     int
	WindowHeight    int
	IconSize        int
	AppPos          [2]int
	ApplicationsPos [2]int
}

// SignOptions holds codesign / productbuild identities.
type SignOptions struct {
	Identity          string
	InstallerIdentity string
	Entitlements      string
}

// Package creates the .app bundle and optional DMG / PKG artifacts.
func Package(opts Options) error {
	if runtime.GOOS != "darwin" {
		return fmt.Errorf("darwin: packaging requires macOS (hdiutil/iconutil/productbuild)")
	}
	if opts.Name == "" {
		return fmt.Errorf("darwin: name required")
	}
	if opts.Binary == "" {
		return fmt.Errorf("darwin: binary required")
	}
	if opts.ID == "" {
		return fmt.Errorf("darwin: bundle id required (yoga.toml id / [darwin].id or -id)")
	}
	if opts.OutDir == "" {
		opts.OutDir = filepath.Join("dist", "darwin")
	}
	if opts.DisplayName == "" {
		opts.DisplayName = opts.Name
	}
	if opts.Version == "" {
		opts.Version = "0.1.0"
	}
	if opts.BundleVersion == "" {
		opts.BundleVersion = opts.Version
	}
	if opts.MinSystem == "" {
		opts.MinSystem = "11.0"
	}
	if opts.DMG.VolumeName == "" {
		opts.DMG.VolumeName = opts.Name
	}
	if opts.DMG.WindowWidth <= 0 {
		opts.DMG.WindowWidth = 660
	}
	if opts.DMG.WindowHeight <= 0 {
		opts.DMG.WindowHeight = 400
	}
	if opts.DMG.IconSize <= 0 {
		opts.DMG.IconSize = 128
	}
	if opts.DMG.AppPos == [2]int{} {
		opts.DMG.AppPos = [2]int{180, 200}
	}
	if opts.DMG.ApplicationsPos == [2]int{} {
		opts.DMG.ApplicationsPos = [2]int{480, 200}
	}
	if len(opts.Formats) == 0 {
		opts.Formats = []string{"dmg"}
	}

	if err := os.MkdirAll(opts.OutDir, 0o755); err != nil {
		return err
	}

	appRoot, err := buildAppBundle(opts)
	if err != nil {
		return err
	}
	fmt.Printf("darwin: wrote %s\n", appRoot)

	if opts.Sign.Identity != "" {
		if err := codesignApp(appRoot, opts.Sign); err != nil {
			return err
		}
		fmt.Printf("darwin: signed %s\n", appRoot)
	}

	wantDMG, wantPKG := false, false
	for _, f := range opts.Formats {
		switch strings.ToLower(strings.TrimSpace(f)) {
		case "dmg":
			wantDMG = true
		case "pkg":
			wantPKG = true
		case "app":
			// already written
		default:
			fmt.Fprintf(os.Stderr, "darwin: ignoring unknown format %q\n", f)
		}
	}

	if wantDMG {
		dmgPath, err := createDMG(appRoot, opts)
		if err != nil {
			return err
		}
		fmt.Printf("darwin: wrote %s\n", dmgPath)
	}
	if wantPKG {
		pkgPath, err := createPKG(appRoot, opts)
		if err != nil {
			return err
		}
		fmt.Printf("darwin: wrote %s\n", pkgPath)
		if opts.Sign.InstallerIdentity == "" {
			fmt.Fprintln(os.Stderr, "darwin: pkg is unsigned — set [darwin.sign] installer_identity for App Store / notarized distribution")
		}
	}
	return nil
}

func buildAppBundle(opts Options) (string, error) {
	appName := opts.Name + ".app"
	appRoot := filepath.Join(opts.OutDir, appName)
	_ = os.RemoveAll(appRoot)

	macOSDir := filepath.Join(appRoot, "Contents", "MacOS")
	resDir := filepath.Join(appRoot, "Contents", "Resources")
	if err := os.MkdirAll(macOSDir, 0o755); err != nil {
		return "", err
	}
	if err := os.MkdirAll(resDir, 0o755); err != nil {
		return "", err
	}

	destBin := filepath.Join(macOSDir, opts.Name)
	if err := copyFile(opts.Binary, destBin, 0o755); err != nil {
		return "", err
	}

	iconName := ""
	if opts.Icon != "" {
		icnsPath := filepath.Join(resDir, "AppIcon.icns")
		if err := ensureICNS(opts.Icon, icnsPath); err != nil {
			fmt.Fprintf(os.Stderr, "darwin: icon: %v (continuing without icon)\n", err)
		} else {
			iconName = "AppIcon"
		}
	}

	plist := buildInfoPlist(opts, iconName)
	if err := os.WriteFile(filepath.Join(appRoot, "Contents", "Info.plist"), []byte(plist), 0o644); err != nil {
		return "", err
	}
	// PkgInfo required by some tooling
	_ = os.WriteFile(filepath.Join(appRoot, "Contents", "PkgInfo"), []byte("APPL????"), 0o644)
	return appRoot, nil
}

func buildInfoPlist(opts Options, iconName string) string {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
`)
	writePlistString(&b, "CFBundleDevelopmentRegion", "en")
	writePlistString(&b, "CFBundleExecutable", opts.Name)
	writePlistString(&b, "CFBundleIdentifier", opts.ID)
	writePlistString(&b, "CFBundleInfoDictionaryVersion", "6.0")
	writePlistString(&b, "CFBundleName", opts.Name)
	writePlistString(&b, "CFBundleDisplayName", opts.DisplayName)
	writePlistString(&b, "CFBundlePackageType", "APPL")
	writePlistString(&b, "CFBundleShortVersionString", opts.Version)
	writePlistString(&b, "CFBundleVersion", opts.BundleVersion)
	writePlistString(&b, "LSMinimumSystemVersion", opts.MinSystem)
	b.WriteString("\t<key>NSHighResolutionCapable</key>\n\t<true/>\n")
	b.WriteString("\t<key>LSApplicationSupportsAutomaticGraphicsSwitching</key>\n\t<true/>\n")
	if iconName != "" {
		writePlistString(&b, "CFBundleIconFile", iconName)
	}
	if opts.Copyright != "" {
		writePlistString(&b, "NSHumanReadableCopyright", opts.Copyright)
	}
	if opts.Category != "" {
		writePlistString(&b, "LSApplicationCategoryType", opts.Category)
	}
	b.WriteString(`</dict>
</plist>
`)
	return b.String()
}

func writePlistString(b *strings.Builder, key, val string) {
	b.WriteString("\t<key>")
	b.WriteString(key)
	b.WriteString("</key>\n\t<string>")
	b.WriteString(xmlEscape(val))
	b.WriteString("</string>\n")
}

func xmlEscape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, `"`, "&quot;")
	return s
}

func codesignApp(appRoot string, sign SignOptions) error {
	args := []string{"--force", "--deep", "--options", "runtime", "--sign", sign.Identity}
	if sign.Entitlements != "" {
		args = append(args, "--entitlements", sign.Entitlements)
	}
	args = append(args, appRoot)
	cmd := exec.Command("codesign", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("darwin: codesign: %w", err)
	}
	return nil
}

func sanitizeFile(s string) string {
	return strings.ReplaceAll(s, " ", "-")
}

func ensureICNS(src, dest string) error {
	if strings.HasSuffix(strings.ToLower(src), ".icns") {
		return copyFile(src, dest, 0o644)
	}
	tmp := dest + ".iconset"
	_ = os.RemoveAll(tmp)
	if err := os.MkdirAll(tmp, 0o755); err != nil {
		return err
	}
	defer os.RemoveAll(tmp)

	sizes := []int{16, 32, 128, 256, 512}
	for _, sz := range sizes {
		out := filepath.Join(tmp, fmt.Sprintf("icon_%dx%d.png", sz, sz))
		if err := exec.Command("sips", "-z", fmt.Sprint(sz), fmt.Sprint(sz), src, "--out", out).Run(); err != nil {
			return fmt.Errorf("sips: %w", err)
		}
		out2 := filepath.Join(tmp, fmt.Sprintf("icon_%dx%d@2x.png", sz, sz))
		_ = exec.Command("sips", "-z", fmt.Sprint(sz*2), fmt.Sprint(sz*2), src, "--out", out2).Run()
	}
	if err := exec.Command("iconutil", "-c", "icns", tmp, "-o", dest).Run(); err != nil {
		return fmt.Errorf("iconutil: %w", err)
	}
	return nil
}

func copyFile(src, dst string, mode os.FileMode) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, mode)
}

func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, info.Mode())
	})
}
