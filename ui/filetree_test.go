package ui

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFileTreeReloadKeepsExpandedFolders(t *testing.T) {
	newTestCtx(t)
	root := t.TempDir()
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	must(os.MkdirAll(filepath.Join(root, "a", "b"), 0o755))
	must(os.WriteFile(filepath.Join(root, "a", "b", "x.go"), nil, 0o644))

	ft := NewFileTree(root)
	if !ft.Reveal(filepath.Join(root, "a", "b", "x.go")) {
		t.Fatal("reveal x.go failed")
	}
	a := ft.find(filepath.Join(root, "a"))
	if a == nil || !a.expanded {
		t.Fatal("a should be loaded and expanded")
	}

	must(os.WriteFile(filepath.Join(root, "new.txt"), nil, 0o644))
	ft.Reload(root)
	if ft.find(filepath.Join(root, "new.txt")) == nil {
		t.Fatal("reload did not pick up new.txt")
	}
	if got := ft.find(filepath.Join(root, "a")); got != a || !got.expanded {
		t.Fatal("reload replaced or collapsed the expanded folder")
	}
}

func TestFileTreeHidden(t *testing.T) {
	newTestCtx(t)
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, ".env"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	ft := NewFileTree(root)
	if ft.find(filepath.Join(root, ".env")) != nil {
		t.Fatal("dotfiles are hidden by default")
	}
	ft.Hidden = func(string, bool) bool { return false }
	ft.Reload(root)
	if ft.find(filepath.Join(root, ".env")) == nil {
		t.Fatal("Hidden=false should show .env")
	}
	if ft.Reveal(filepath.Join(filepath.Dir(root), "elsewhere")) {
		t.Fatal("reveal outside root should fail")
	}
}
