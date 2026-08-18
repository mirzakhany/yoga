package ui

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMatchFileExt(t *testing.T) {
	if !matchFileExt("main.go", []string{".go"}) {
		t.Fatal("expected .go to match")
	}
	if !matchFileExt("main.go", []string{"go"}) {
		t.Fatal("expected go without dot to match")
	}
	if matchFileExt("main.go", []string{".md"}) {
		t.Fatal("did not expect .md to match .go")
	}
	if matchFileExt("README", []string{".go"}) {
		t.Fatal("extensionless should not match")
	}
}

func TestFilterFileEntriesKeepsDirs(t *testing.T) {
	in := []fileEntry{
		{Name: "src", IsDir: true},
		{Name: "a.go", IsDir: false},
		{Name: "b.md", IsDir: false},
	}
	out := filterFileEntries(in, FileFilter{Exts: []string{".go"}})
	if len(out) != 2 || out[0].Name != "src" || out[1].Name != "a.go" {
		t.Fatalf("filter: %+v", out)
	}
}

func TestReadFileEntriesHidesDotfiles(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "visible.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".hidden"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}

	got := readFileEntries(dir, false)
	names := map[string]fileEntry{}
	for _, e := range got {
		names[e.Name] = e
	}
	if _, ok := names["visible.txt"]; !ok {
		t.Fatalf("missing file: %+v", names)
	}
	if e, ok := names["sub"]; !ok || !e.IsDir {
		t.Fatalf("missing dir: %+v", names)
	}
	if _, ok := names[".hidden"]; ok {
		t.Fatal("dotfile should be hidden")
	}

	got = readFileEntries(dir, true)
	names = map[string]fileEntry{}
	for _, e := range got {
		names[e.Name] = e
	}
	if _, ok := names[".hidden"]; !ok {
		t.Fatal("ShowHidden should list dotfiles")
	}
	if !got[0].IsDir {
		t.Fatal("directories should sort first")
	}
}

func TestRememberRecent(t *testing.T) {
	got := rememberRecent([]string{"/a", "/b"}, []string{"/c", "/a"})
	if len(got) < 2 || got[0] != "/c" || got[1] != "/a" {
		t.Fatalf("recent order: %v", got)
	}
}
