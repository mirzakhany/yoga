package windows

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Options configures a portable Windows zip package.
type Options struct {
	Name    string
	Version string
	Binary  string // path to .exe
	Icon    string // optional; copied as-is if present
	Arch    string
	OutDir  string
}

// Package creates Name-version-windows-arch.zip containing the executable.
func Package(opts Options) error {
	if opts.Name == "" || opts.Binary == "" {
		return fmt.Errorf("windows: name and binary required")
	}
	if opts.OutDir == "" {
		opts.OutDir = filepath.Join("dist", "windows")
	}
	if opts.Arch == "" {
		opts.Arch = "amd64"
	}
	if err := os.MkdirAll(opts.OutDir, 0o755); err != nil {
		return err
	}

	zipName := fmt.Sprintf("%s-%s-windows-%s.zip", sanitize(opts.Name), opts.Version, opts.Arch)
	zipPath := filepath.Join(opts.OutDir, zipName)
	_ = os.Remove(zipPath)

	f, err := os.Create(zipPath)
	if err != nil {
		return err
	}
	defer f.Close()
	zw := zip.NewWriter(f)
	defer zw.Close()

	exeName := opts.Name + ".exe"
	if err := addFile(zw, opts.Binary, exeName); err != nil {
		return err
	}
	if opts.Icon != "" {
		base := filepath.Base(opts.Icon)
		_ = addFile(zw, opts.Icon, base)
	}

	fmt.Printf("windows: wrote %s\n", zipPath)
	return nil
}

func sanitize(s string) string {
	return strings.ReplaceAll(s, " ", "-")
}

func addFile(zw *zip.Writer, src, name string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	st, err := in.Stat()
	if err != nil {
		return err
	}
	hdr, err := zip.FileInfoHeader(st)
	if err != nil {
		return err
	}
	hdr.Name = name
	hdr.Method = zip.Deflate
	w, err := zw.CreateHeader(hdr)
	if err != nil {
		return err
	}
	_, err = io.Copy(w, in)
	return err
}
