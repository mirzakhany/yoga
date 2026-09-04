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
	dmgName := fmt.Sprintf("%s-%s.dmg", sanitizeFile(opts.Name), opts.Version)
	dmgPath := filepath.Join(opts.OutDir, dmgName)
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
		rwPath,
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("darwin: hdiutil create rw: %w", err)
	}

	mountRoot := filepath.Join(opts.OutDir, ".dmg-mount")
	_ = os.RemoveAll(mountRoot)
	if err := os.MkdirAll(mountRoot, 0o755); err != nil {
		return "", err
	}
	defer func() {
		_ = exec.Command("hdiutil", "detach", mountRoot, "-quiet", "-force").Run()
		_ = os.RemoveAll(mountRoot)
	}()

	attach := exec.Command("hdiutil", "attach", rwPath, "-readwrite", "-noverify", "-noautoopen", "-mountpoint", mountRoot)
	attach.Stdout = os.Stdout
	attach.Stderr = os.Stderr
	if err := attach.Run(); err != nil {
		return "", fmt.Errorf("darwin: hdiutil attach: %w", err)
	}

	if err := applyDMGFinderLayout(mountRoot, appName, opts); err != nil {
		fmt.Fprintf(os.Stderr, "darwin: dmg layout: %v (dmg will still be created)\n", err)
	}

	// Bless / sync before detach
	time.Sleep(500 * time.Millisecond)
	_ = exec.Command("sync").Run()
	if err := exec.Command("hdiutil", "detach", mountRoot, "-quiet").Run(); err != nil {
		// retry force
		_ = exec.Command("hdiutil", "detach", mountRoot, "-force").Run()
	}

	_ = os.Remove(dmgPath)
	conv := exec.Command("hdiutil", "convert", rwPath, "-format", "UDZO", "-imagekey", "zlib-level=9", "-o", dmgPath)
	conv.Stdout = os.Stdout
	conv.Stderr = os.Stderr
	if err := conv.Run(); err != nil {
		return "", fmt.Errorf("darwin: hdiutil convert: %w", err)
	}
	return dmgPath, nil
}

func applyDMGFinderLayout(mountRoot, appName string, opts Options) error {
	vol := opts.DMG.VolumeName
	w, h := opts.DMG.WindowWidth, opts.DMG.WindowHeight
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
    set the bounds of container window to {100, 100, %d, %d}
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
`, escapeAS(vol), 100+w, 100+h, iconSize, bgClause, escapeAS(appName), ax, ay, px, py)

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
