package windows

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
)

// Options configures a portable Windows zip package.
type Options struct {
	Name     string
	Binary   string // path to .exe
	OutDir   string
	Artifact string // zip name without extension
	// Files maps a zip path to a source file or directory.
	Files map[string]string
}

// Package creates <Artifact>.zip holding the exe and any extra files.
func Package(opts Options) error {
	if opts.Name == "" || opts.Binary == "" {
		return fmt.Errorf("windows: name and binary required")
	}
	if opts.OutDir == "" {
		opts.OutDir = filepath.Join("dist", "windows")
	}
	if opts.Artifact == "" {
		opts.Artifact = opts.Name
	}
	if err := os.MkdirAll(opts.OutDir, 0o755); err != nil {
		return err
	}

	zipPath := filepath.Join(opts.OutDir, opts.Artifact+".zip")
	_ = os.Remove(zipPath)

	f, err := os.Create(zipPath)
	if err != nil {
		return err
	}
	defer f.Close()
	zw := zip.NewWriter(f)

	if err := addFile(zw, opts.Binary, opts.Name+".exe"); err != nil {
		return err
	}
	dsts := make([]string, 0, len(opts.Files))
	for dst := range opts.Files {
		dsts = append(dsts, dst)
	}
	sort.Strings(dsts)
	for _, dst := range dsts {
		if err := addTree(zw, opts.Files[dst], dst); err != nil {
			return fmt.Errorf("windows: %s: %w", dst, err)
		}
	}
	if err := zw.Close(); err != nil {
		return err
	}

	fmt.Printf("windows: wrote %s\n", zipPath)
	return nil
}

// addTree adds src (a file, or a directory walked recursively) under name.
func addTree(zw *zip.Writer, src, name string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		return addFile(zw, path, filepath.ToSlash(filepath.Join(name, rel)))
	})
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
