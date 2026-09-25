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
	Name     string
	ID       string
	Version  string
	Binary   string
	BinName  string // executable name in the archive (defaults to Name)
	Icon     string
	Arch     string
	OutDir   string
	Artifact string // archive name without extension
	Format   string // "tar.gz" (default) or "tar.xz"
	// Files maps an archive path to a source file or directory. When set,
	// the archive holds the binary plus exactly these files.
	Files map[string]string
}

// Package writes a tar.gz or tar.xz and, when appimagetool is available,
// an AppImage.
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
	if opts.BinName == "" {
		opts.BinName = opts.Name
	}
	if opts.Artifact == "" {
		opts.Artifact = fmt.Sprintf("%s-%s-linux-%s", sanitize(opts.Name), opts.Version, opts.Arch)
	}
	if opts.Format == "" {
		opts.Format = "tar.gz"
	}
	if opts.Format != "tar.gz" && opts.Format != "tar.xz" {
		return fmt.Errorf("linux: unknown format %q (want tar.gz or tar.xz)", opts.Format)
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

	binDest := filepath.Join(stage, opts.BinName)
	if err := copyFile(opts.Binary, binDest, 0o755); err != nil {
		return err
	}

	if len(opts.Files) > 0 {
		for dst, src := range opts.Files {
			if err := copyTree(src, filepath.Join(stage, dst)); err != nil {
				return fmt.Errorf("linux: %s: %w", dst, err)
			}
		}
	} else {
		if err := os.WriteFile(filepath.Join(stage, opts.Name+".desktop"), []byte(desktopEntry(opts)), 0o644); err != nil {
			return err
		}
		if opts.Icon != "" {
			ext := filepath.Ext(opts.Icon)
			if ext == "" {
				ext = ".png"
			}
			_ = copyFile(opts.Icon, filepath.Join(stage, opts.Name+ext), 0o644)
		}
	}

	tarball := filepath.Join(opts.OutDir, opts.Artifact+"."+opts.Format)
	if err := writeTar(tarball, stage, opts.Format); err != nil {
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
	if err := copyFile(opts.Binary, filepath.Join(appDir, "usr", "bin", opts.BinName), 0o755); err != nil {
		return err
	}
	// AppRun
	appRun := fmt.Sprintf("#!/bin/sh\nexec \"$(dirname \"$0\")/usr/bin/%s\" \"$@\"\n", opts.BinName)
	if err := os.WriteFile(filepath.Join(appDir, "AppRun"), []byte(appRun), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(appDir, opts.Name+".desktop"), []byte(desktopEntry(opts)), 0o644); err != nil {
		return err
	}
	if opts.Icon != "" {
		_ = copyFile(opts.Icon, filepath.Join(appDir, opts.Name+filepath.Ext(opts.Icon)), 0o644)
	}

	outApp := filepath.Join(opts.OutDir, opts.Artifact+".AppImage")
	cmd := exec.Command("appimagetool", appDir, outApp)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("linux: appimagetool: %w", err)
	}
	fmt.Printf("linux: wrote %s\n", outApp)
	return nil
}

func desktopEntry(opts Options) string {
	return fmt.Sprintf(`[Desktop Entry]
Type=Application
Name=%s
Exec=%s
Icon=%s
StartupWMClass=%s
Categories=Utility;
Terminal=false
`, opts.Name, opts.BinName, opts.Name, opts.ID)
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

// copyTree copies src (a file or a directory) to dst.
func copyTree(src, dst string) error {
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
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		return copyFile(path, target, info.Mode().Perm())
	})
}

// writeTar archives dir into outPath. tar.xz shells out to xz, which Go's
// standard library cannot write.
func writeTar(outPath, dir, format string) error {
	if format == "tar.gz" {
		return writeTarGz(outPath, dir)
	}
	if _, err := exec.LookPath("xz"); err != nil {
		return fmt.Errorf("linux: tar.xz needs xz on PATH: %w", err)
	}
	f, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer f.Close()
	xz := exec.Command("xz", "-9", "-c")
	xz.Stdout = f
	xz.Stderr = os.Stderr
	in, err := xz.StdinPipe()
	if err != nil {
		return err
	}
	if err := xz.Start(); err != nil {
		return err
	}
	werr := writeTarStream(in, dir)
	in.Close()
	if err := xz.Wait(); err != nil {
		return fmt.Errorf("linux: xz: %w", err)
	}
	return werr
}

func writeTarGz(outPath, dir string) error {
	f, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer f.Close()
	gz := gzip.NewWriter(f)
	if err := writeTarStream(gz, dir); err != nil {
		return err
	}
	return gz.Close()
}

func writeTarStream(w io.Writer, dir string) error {
	tw := tar.NewWriter(w)
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil || rel == "." {
			return err
		}
		hdr, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		hdr.Name = filepath.ToSlash(rel)
		if info.IsDir() {
			hdr.Name += "/"
		}
		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		r, err := os.Open(path)
		if err != nil {
			return err
		}
		defer r.Close()
		_, err = io.Copy(tw, r)
		return err
	})
	if err != nil {
		return err
	}
	return tw.Close()
}
