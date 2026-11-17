package ui

import (
	"testing"
	"time"

	"github.com/mirzakhany/yoga/highlight"
	"github.com/mirzakhany/yoga/input"
)

func TestEditorRepathFollowsTheLanguage(t *testing.T) {
	e := newTestEditor(t, []byte("package main\n\nfunc main() {}\n"))
	e.Path = "untitled"
	e.Repath("/tmp/x/main.go")
	if e.Path != "/tmp/x/main.go" {
		t.Fatalf("path %q", e.Path)
	}
	if _, noop := e.hl.(highlight.Noop); noop {
		t.Fatal("a .go path should get the Go highlighter")
	}
	deadline := time.Now().Add(5 * time.Second)
	for len(e.tokens) == 0 {
		if time.Now().After(deadline) {
			t.Fatal("no tokens after Repath")
		}
		time.Sleep(10 * time.Millisecond)
		e.Update(&input.Mouse{})
	}
	e.Repath("/tmp/x/notes.zzz")
	if _, noop := e.hl.(highlight.Noop); !noop || e.tokens != nil {
		t.Fatalf("unknown extension: %T %d tokens", e.hl, len(e.tokens))
	}
}
