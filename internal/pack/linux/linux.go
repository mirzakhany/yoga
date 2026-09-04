package linux

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// Options configures Linux tar.gz (+ optional AppImage) packaging.
type Options struct {
	Name    string
	ID      string
	Version string
	Binary  string
	Icon    string
	Arch    string
	OutDir  string
}

// Package writes a tar.gz and, when appimagetool is available, an AppImage.
func Package(opts Options) error {
	if opts.Name == "" || opts.Binary == "" {
		return fmt.Errorf("linux: name and binary required")
	}
	if opts.ID == "" {
		opts.ID = "com.example." + sanitize(opts.Name)
	}
	if opts.OutDir == "" {
		opts.OutDir = filepath.Join("dist", "linux")
	}
	if opts.Arch == "" {
		opts.Arch = runtime.GOARCH
	}
	if err := os.MkdirAll(opts.OutDir, 0o755); err != nil {
		return err
	}

	stage := filepath.Join(opts.OutDir, ".stage")
	_ = os.RemoveAll(stage)
	if err := os.MkdirAll(stage, 0o755); err != nil {
		return err
	}
	defer os.RemoveAll(stage)

	binDest := filepath.Join(stage, opts.Name)
	if err := copyFile(opts.Binary, binDest, 0o755); err != nil {
		return err
	}

	desktop := fmt.Sprintf(`[Desktop Entry]
Type=Application
Name=%s
Exec=%s
Icon=%s
StartupWMClass=%s
Categories=Utility;
Terminal=false
`, opts.Name, opts.Name, opts.Name, opts.ID)
	if err := os.WriteFile(filepath.Join(stage, opts.Name+".desktop"), []byte(desktop), 0o644); err != nil {
		return err
	}

	if opts.Icon != "" {
		ext := filepath.Ext(opts.Icon)
		if ext == "" {
			ext = ".png"
		}
		_ = copyFile(opts.Icon, filepath.Join(stage, opts.Name+ext), 0o644)
	}

	tarball := filepath.Join(opts.OutDir, fmt.Sprintf("%s-%s-linux-%s.tar.gz", sanitize(opts.Name), opts.Version, opts.Arch))
	if err := writeTarGz(tarball, stage); err != nil {
		return err
	}
	fmt.Printf("linux: wrote %s\n", tarball)

	if _, err := exec.LookPath("appimagetool"); err != nil {
		fmt.Fprintln(os.Stderr, "linux: appimagetool not found; skipped AppImage (install from https://appimage.github.io/appimagetool/)")
		return nil
	}

	appDir := filepath.Join(opts.OutDir, opts.Name+".AppDir")
	_ = os.RemoveAll(appDir)
	if err := os.MkdirAll(filepath.Join(appDir, "usr", "bin"), 0o755); err != nil {
		return err
	}
	if err := copyFile(opts.Binary, filepath.Join(appDir, "usr", "bin", opts.Name), 0o755); err != nil {
		return err
	}
	// AppRun
	appRun := fmt.Sprintf("#!/bin/sh\nexec \"$(dirname \"$0\")/usr/bin/%s\" \"$@\"\n", opts.Name)
	if err := os.WriteFile(filepath.Join(appDir, "AppRun"), []byte(appRun), 0o755); err != nil {
		return err
	}
	desk := fmt.Sprintf(`[Desktop Entry]
Type=Application
Name=%s
Exec=%s
Icon=%s
StartupWMClass=%s
Categories=Utility;
Terminal=false
`, opts.Name, opts.Name, opts.Name, opts.ID)
	if err := os.WriteFile(filepath.Join(appDir, opts.Name+".desktop"), []byte(desk), 0o644); err != nil {
		return err
	}
	if opts.Icon != "" {
		_ = copyFile(opts.Icon, filepath.Join(appDir, opts.Name+filepath.Ext(opts.Icon)), 0o644)
	}

	outApp := filepath.Join(opts.OutDir, fmt.Sprintf("%s-%s-%s.AppImage", sanitize(opts.Name), opts.Version, opts.Arch))
	cmd := exec.Command("appimagetool", appDir, outApp)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("linux: appimagetool: %w", err)
	}
	fmt.Printf("linux: wrote %s\n", outApp)
	return nil
}

func sanitize(s string) string {
	return strings.ReplaceAll(s, " ", "-")
}

func copyFile(src, dst string, mode os.FileMode) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, mode)
}

func writeTarGz(outPath, dir string) error {
	f, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer f.Close()
	gz := gzip.NewWriter(f)
	defer gz.Close()
	tw := tar.NewWriter(gz)
	defer tw.Close()

	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		hdr, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		hdr.Name = rel
		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}
		r, err := os.Open(path)
		if err != nil {
			return err
		}
		defer r.Close()
		_, err = io.Copy(tw, r)
		return err
	})
}
