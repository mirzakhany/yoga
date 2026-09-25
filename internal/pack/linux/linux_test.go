package linux_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/mirzakhany/yoga/internal/pack/linux"
)

func TestPackageTarGz(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "bin")
	if err := os.WriteFile(bin, []byte("#!/bin/sh\necho hi\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "out")
	err := linux.Package(linux.Options{
		Name:    "Demo",
		Version: "0.1.0",
		Binary:  bin,
		Arch:    "amd64",
		OutDir:  out,
	})
	if err != nil {
		t.Fatal(err)
	}
	matches, _ := filepath.Glob(filepath.Join(out, "*.tar.gz"))
	if len(matches) != 1 {
		t.Fatalf("expected one tarball, got %v", matches)
	}
}

func TestPackageTarXzWithFiles(t *testing.T) {
	if _, err := exec.LookPath("xz"); err != nil {
		t.Skip("xz not on PATH")
	}
	dir := t.TempDir()
	bin := filepath.Join(dir, "bin")
	if err := os.WriteFile(bin, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "assets"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "assets", "demo.desktop"), []byte("[Desktop Entry]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "LICENSE"), []byte("MIT\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "out")
	err := linux.Package(linux.Options{
		Name:     "Demo",
		Binary:   bin,
		BinName:  "demo",
		OutDir:   out,
		Artifact: "demo-linux-v1.0.0-amd64",
		Format:   "tar.xz",
		Files: map[string]string{
			"LICENSE":        filepath.Join(dir, "LICENSE"),
			"desktop-assets": filepath.Join(dir, "assets"),
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	list, err := exec.Command("tar", "-tJf", filepath.Join(out, "demo-linux-v1.0.0-amd64.tar.xz")).Output()
	if err != nil {
		t.Fatal(err)
	}
	got := strings.Fields(string(list))
	sort.Strings(got)
	want := []string{"LICENSE", "demo", "desktop-assets/", "desktop-assets/demo.desktop"}
	if strings.Join(got, " ") != strings.Join(want, " ") {
		t.Fatalf("archive = %q, want %q", got, want)
	}
}
