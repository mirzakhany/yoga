package ui

import (
	"testing"

	"github.com/mirzakhany/yoga/highlight"
	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
)

// TestEditorSearchPaste checks that Cmd/Ctrl+V with the find bar focused fills
// the query instead of editing the document.
func TestEditorSearchPaste(t *testing.T) {
	text, err := shape.NewEngine(1, false)
	if err != nil {
		t.Skip(err)
	}
	clip := &input.MemClipboard{}
	SetFrameResources(text, render.NewSpriteSheet(text.Atlas), clip)

	ed := NewEditor([]byte("alpha beta\ngamma beta\n"), highlight.Noop{})
	defer ed.Close()
	ed.openSearch(false)

	clip.Set("beta\nignored")
	ed.HandleKeys([]input.KeyEvent{{Key: input.KeyV, Mods: input.ModSuper}})
	if ed.search.query != "beta" {
		t.Fatalf("query = %q, want %q", ed.search.query, "beta")
	}
	if got := len(ed.search.matches); got != 2 {
		t.Fatalf("matches = %d, want 2", got)
	}
	if got := string(ed.pt.Bytes()); got != "alpha beta\ngamma beta\n" {
		t.Fatalf("document changed: %q", got)
	}

	// The menu path (Editor.Paste) lands in the query too.
	ed.Paste()
	if ed.search.query != "betabeta" {
		t.Fatalf("query after Paste = %q", ed.search.query)
	}
}
