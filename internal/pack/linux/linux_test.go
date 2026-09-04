package linux_test

import (
	"os"
	"path/filepath"
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
