package darwin

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// createDMG builds a compressed DMG with Applications symlink and optional
// Finder background / icon layout.
func createDMG(appRoot string, opts Options) (string, error) {
	appName := filepath.Base(appRoot)
	dmgPath := filepath.Join(opts.OutDir, opts.Artifact+".dmg")
	_ = os.Remove(dmgPath)

	staging := filepath.Join(opts.OutDir, ".dmg-staging")
	_ = os.RemoveAll(staging)
	if err := os.MkdirAll(staging, 0o755); err != nil {
		return "", err
	}
	defer os.RemoveAll(staging)

	if err := copyDir(appRoot, filepath.Join(staging, appName)); err != nil {
		return "", err
	}
	_ = os.Symlink("/Applications", filepath.Join(staging, "Applications"))

	if opts.DMG.Background != "" {
		bgDir := filepath.Join(staging, ".background")
		if err := os.MkdirAll(bgDir, 0o755); err != nil {
			return "", err
		}
		ext := strings.ToLower(filepath.Ext(opts.DMG.Background))
		if ext == "" {
			ext = ".png"
		}
		bgDest := filepath.Join(bgDir, "background"+ext)
		if err := copyFile(opts.DMG.Background, bgDest, 0o644); err != nil {
			return "", fmt.Errorf("darwin: dmg background: %w", err)
		}
	}

	return createStyledDMG(staging, dmgPath, appName, opts)
}

func createStyledDMG(staging, dmgPath, appName string, opts Options) (string, error) {
	rwPath := filepath.Join(opts.OutDir, ".dmg-rw.dmg")
	_ = os.Remove(rwPath)
	defer os.Remove(rwPath)

	// Leave headroom for Finder metadata / resource forks.
	cmd := exec.Command("hdiutil", "create",
		"-volname", opts.DMG.VolumeName,
		"-srcfolder", staging,
		"-ov", "-format", "UDRW",
		"-fs", "HFS+",
		"-quiet",
		rwPath,
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("darwin: hdiutil create rw: %w", err)
	}

	// Mount under /Volumes: Finder only scripts disks it lists, and it does
	// not list a volume mounted at a custom -mountpoint.
	out, err := exec.Command("hdiutil", "attach", rwPath, "-readwrite", "-noverify", "-noautoopen").Output()
	if err != nil {
		return "", fmt.Errorf("darwin: hdiutil attach: %w", err)
	}
	mountRoot := mountPoint(string(out))
	if mountRoot == "" {
		return "", fmt.Errorf("darwin: hdiutil attach: no mount point in %q", out)
	}
	defer func() {
		_ = exec.Command("hdiutil", "detach", mountRoot, "-quiet", "-force").Run()
	}()

	// Show "Chapar", not "Chapar.app", next to the Applications link.
	_ = exec.Command("SetFile", "-a", "E", filepath.Join(mountRoot, appName)).Run()
	if err := applyDMGFinderLayout(mountRoot, appName, opts); err != nil {
		fmt.Fprintf(os.Stderr, "darwin: dmg layout: %v (dmg will still be created)\n", err)
	}
	if opts.DMG.VolumeIcon != "" {
		// After the Finder pass, which drops a volume icon set before it.
		// The custom-icon attribute on the root makes Finder use the file.
		icon := filepath.Join(mountRoot, ".VolumeIcon.icns")
		if err := copyFile(opts.DMG.VolumeIcon, icon, 0o644); err != nil {
			return "", fmt.Errorf("darwin: dmg volume icon: %w", err)
		}
		_ = exec.Command("SetFile", "-c", "icnC", icon).Run()
		if err := exec.Command("SetFile", "-a", "C", mountRoot).Run(); err != nil {
			fmt.Fprintf(os.Stderr, "darwin: dmg volume icon: SetFile: %v\n", err)
		}
	}

	// Bless / sync before detach
	time.Sleep(500 * time.Millisecond)
	_ = exec.Command("sync").Run()
	if err := exec.Command("hdiutil", "detach", mountRoot, "-quiet").Run(); err != nil {
		// retry force
		_ = exec.Command("hdiutil", "detach", mountRoot, "-force").Run()
	}

	_ = os.Remove(dmgPath)
	conv := exec.Command("hdiutil", "convert", rwPath, "-format", "UDZO", "-imagekey", "zlib-level=9", "-quiet", "-o", dmgPath)
	conv.Stdout = os.Stdout
	conv.Stderr = os.Stderr
	if err := conv.Run(); err != nil {
		return "", fmt.Errorf("darwin: hdiutil convert: %w", err)
	}
	return dmgPath, nil
}

// mountPoint picks the /Volumes path out of `hdiutil attach` output, whose
// lines end in a tab-separated mount point when the slice has one.
func mountPoint(out string) string {
	for _, line := range strings.Split(out, "\n") {
		if i := strings.Index(line, "\t/Volumes/"); i >= 0 {
			return strings.TrimSpace(line[i+1:])
		}
	}
	return ""
}

func applyDMGFinderLayout(mountRoot, appName string, opts Options) error {
	// A second mounted volume with the same name gets a " 1" suffix, so ask
	// Finder for the disk this image actually mounted as.
	vol := filepath.Base(mountRoot)
	w, h := opts.DMG.WindowWidth, opts.DMG.WindowHeight
	wx, wy := opts.DMG.WindowPos[0], opts.DMG.WindowPos[1]
	ax, ay := opts.DMG.AppPos[0], opts.DMG.AppPos[1]
	px, py := opts.DMG.ApplicationsPos[0], opts.DMG.ApplicationsPos[1]
	iconSize := opts.DMG.IconSize

	bgClause := ""
	if opts.DMG.Background != "" {
		// Find the background file we copied into .background/
		entries, _ := os.ReadDir(filepath.Join(mountRoot, ".background"))
		bgFile := "background.png"
		for _, e := range entries {
			if !e.IsDir() {
				bgFile = e.Name()
				break
			}
		}
		// Hide background folder from Finder icon view
		_ = exec.Command("SetFile", "-a", "V", filepath.Join(mountRoot, ".background")).Run()
		bgClause = fmt.Sprintf(`
    set background picture of theViewOptions to file ".background:%s"
`, bgFile)
	}

	script := fmt.Sprintf(`
tell application "Finder"
  tell disk "%s"
    open
    set current view of container window to icon view
    set toolbar visible of container window to false
    set statusbar visible of container window to false
    set the bounds of container window to {%d, %d, %d, %d}
    set theViewOptions to the icon view options of container window
    set arrangement of theViewOptions to not arranged
    set icon size of theViewOptions to %d
    %s
    set position of item "%s" of container window to {%d, %d}
    set position of item "Applications" of container window to {%d, %d}
    close
    open
    update without registering applications
    delay 1
    close
  end tell
end tell
`, escapeAS(vol), wx, wy, wx+w, wy+h, iconSize, bgClause, escapeAS(appName), ax, ay, px, py)

	cmd := exec.Command("osascript", "-e", script)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("osascript: %w (%s)", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func escapeAS(s string) string {
	return strings.ReplaceAll(s, `"`, `\"`)
}
