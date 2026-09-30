package ui

import (
	"testing"

	"github.com/mirzakhany/yoga/highlight"
	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/layout"
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

// TestEditorSearchClickPlacesCaret drives real frames: after the text area
// takes focus, a click in the find input must focus it again and put the
// caret under the pointer, so typing lands there. In replace mode a click on
// the second row focuses the replace input.
func TestEditorSearchClickPlacesCaret(t *testing.T) {
	c := newTestCtx(t)
	ed := NewEditor([]byte("alpha beta\ngamma beta\n"), highlight.Noop{})
	defer ed.Close()
	body := func(cc *Ctx) View { return Column(ViewOf(ed).Grow(1)).Grow(1) }
	frame := func(m *input.Mouse, kb *input.Keyboard) {
		if m == nil {
			m = &input.Mouse{X: -1, Y: -1}
		}
		if kb == nil {
			kb = &input.Keyboard{}
		}
		root := BuildFrame(c, body, 1000, 600, m, kb)
		// Same order as the app loop.
		layout.Dispatch(root, m)
		c.Focus().HandleMouse(m)
		c.Focus().Route(kb)
		layout.Paint(root, &render.DrawList{}, c.Text())
		c.EndFrame()
	}
	click := func(x, y float32) {
		frame(&input.Mouse{X: x, Y: y}, nil)
		frame(&input.Mouse{X: x, Y: y, Down: true, Pressed: true}, nil)
		frame(&input.Mouse{X: x, Y: y, Released: true}, nil)
	}
	frame(nil, nil)
	ed.openSearch(true)
	ed.search.query = "abcdef"
	ed.search.queryCaret = len(ed.search.query)
	frame(nil, nil)

	// Click in the text area: the bar gives up focus.
	f := ed.viewport.Frame
	click(f.X+80, f.Y+f.H-20)
	if ed.search.focused {
		t.Fatal("text-area click left the find bar focused")
	}

	// Click between "ab" and "cdef" in the find input.
	in := ed.searchInputRect(0)
	cellW, _ := c.Text().MeasureMono("a")
	click(in.X+searchTextPadX+cellW*2+1, in.Y+in.H/2)
	if !ed.search.focused || ed.search.focusField != 0 {
		t.Fatalf("find input not focused: focused=%v field=%d", ed.search.focused, ed.search.focusField)
	}
	if ed.search.queryCaret != 2 {
		t.Fatalf("queryCaret = %d, want 2", ed.search.queryCaret)
	}
	frame(nil, &input.Keyboard{Chars: []rune{'X'}})
	if ed.search.query != "abXcdef" {
		t.Fatalf("typed into %q, want abXcdef", ed.search.query)
	}

	// Click in the replace input.
	rin := ed.searchInputRect(1)
	click(rin.X+rin.W/2, rin.Y+rin.H/2)
	if ed.search.focusField != 1 {
		t.Fatalf("replace input not focused: field=%d", ed.search.focusField)
	}
	frame(nil, &input.Keyboard{Chars: []rune{'r'}})
	if ed.search.replace != "r" {
		t.Fatalf("replace = %q, want r", ed.search.replace)
	}
}
