package windows_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mirzakhany/yoga/internal/pack/windows"
)

func TestPackageZip(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "app.exe")
	if err := os.WriteFile(bin, []byte("MZ fake"), 0o644); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "out")
	err := windows.Package(windows.Options{
		Name:     "Demo",
		Binary:   bin,
		OutDir:   out,
		Artifact: "demo-windows-v0.1.0-amd64",
	})
	if err != nil {
		t.Fatal(err)
	}
	matches, _ := filepath.Glob(filepath.Join(out, "*.zip"))
	if len(matches) != 1 {
		t.Fatalf("expected one zip, got %v", matches)
	}
}
