package ui

import (
	"runtime"
	"testing"
	"weak"

	"github.com/mirzakhany/yoga/highlight"
	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
)

// TestClosedEditorReleasedAfterFrame reproduces the shape of a closed tab: an
// editor holding a large document is laid out (so the runtime registers it for
// focus, overlays and commands), then removed from the tree. Once a later frame
// no longer includes it, nothing the runtime owns may still reach it.
//
// This is the case per-frame slices get wrong: they reset their length but keep
// the backing array, and the frame after a close registers *fewer* widgets — so
// the vacated tail slots are never overwritten.
func TestClosedEditorReleasedAfterFrame(t *testing.T) {
	text, err := shape.NewEngine(1, false)
	if err != nil {
		t.Skip(err)
	}
	SetFrameResources(text, render.NewSpriteSheet(text.Atlas), &input.MemClipboard{})
	c := New(text, NewFocusScope(), nil)
	c.SetIcons(render.NewSpriteSheet(text.Atlas))
	c.SetClipboard(&input.MemClipboard{})

	const docSize = 8 << 20
	payload := make([]byte, 0, docSize)
	for len(payload) < docSize {
		payload = append(payload, `{"id":1,"name":"item"}`+"\n"...)
	}

	ed := NewEditor(payload, highlight.Noop{})
	// A second focusable so the "open" frame registers more widgets than the
	// "closed" one, which is what strands the editor in the tail.
	var dl render.DrawList
	frame := func(withEditor bool) {
		body := func(cc *Ctx) View {
			if withEditor {
				return Column(
					TextField("f1", "a"),
					TextField("f2", "b"),
					ViewOf(ed).Grow(1),
				).Grow(1)
			}
			return Column(TextField("f1", "a")).Grow(1)
		}
		root := BuildFrame(c, body, 900, 600, &input.Mouse{}, &input.Keyboard{})
		dl.Reset()
		layout.Paint(root, &dl, text)
		c.EndFrame()
	}

	// Open: lay the editor out and give it focus, as a user clicking in would.
	frame(true)
	c.Focus().Focus(ed)
	frame(true)

	withEditor := heapMB()

	// Close the tab: drop the app's own reference and stop including it.
	ed.Close()
	ed = nil
	for i := 0; i < 3; i++ {
		frame(false)
	}

	afterClose := heapMB()
	runtime.KeepAlive(c)
	runtime.KeepAlive(payload)

	freed := withEditor - afterClose
	want := float64(docSize) / 1e6 * 0.5
	t.Logf("editor with a %d MB document: heap %.1f -> %.1f MB (freed %.1f MB)",
		docSize>>20, withEditor, afterClose, freed)
	if freed < want {
		t.Errorf("closing the editor freed only %.1f MB of ~%.1f MB — the runtime still reaches it",
			freed, float64(docSize)/1e6)
	}
}

// TestClosedInInputPhaseReleasedSameFrame covers a tab closed by a click. The
// runtime builds the body for input, dispatches — the click handler drops the
// tab — then rebuilds for paint. The input build still contained the tab, so
// its widget-store entries (whose handlers capture the tab) must not survive
// this frame's EndFrame: an idle app may not run another frame for minutes.
func TestClosedInInputPhaseReleasedSameFrame(t *testing.T) {
	text, err := shape.NewEngine(1, false)
	if err != nil {
		t.Skip(err)
	}
	SetFrameResources(text, render.NewSpriteSheet(text.Atlas), &input.MemClipboard{})
	c := New(text, NewFocusScope(), nil)
	c.SetIcons(render.NewSpriteSheet(text.Atlas))
	c.SetClipboard(&input.MemClipboard{})

	type tab struct{ body []byte }
	open := &tab{body: make([]byte, payloadSize)}
	ref := weak.Make(open)

	body := func(cc *Ctx) View {
		if open == nil {
			return Column(Text("empty")).Grow(1)
		}
		tb := open
		return Column(
			Button("tab-send", Text("Send")).OnClick(func() { _ = tb.body }),
			TextField("tab-url", "https://example.com").OnChange(func(string) { _ = tb.body }),
		).Grow(1)
	}

	// One iteration of the runtime loop; onInput stands in for dispatch.
	var dl render.DrawList
	iter := func(onInput func()) {
		mouse, kb := &input.Mouse{}, &input.Keyboard{}
		BuildFrame(c, body, 900, 600, mouse, kb)
		if onInput != nil {
			onInput()
		}
		root := BuildFrame(c, body, 900, 600, mouse, kb)
		dl.Reset()
		layout.Paint(root, &dl, text)
		c.EndFrame()
	}

	iter(nil)
	iter(nil)
	iter(func() { open = nil }) // the close click

	runtime.GC()
	runtime.GC()
	runtime.KeepAlive(c)
	if ref.Value() != nil {
		t.Error("a tab closed during the input phase is still reachable after its frame ended")
	}
}

// TestFocusScopeReleasesUnregisteredWidgets isolates the focus item list.
func TestFocusScopeReleasesUnregisteredWidgets(t *testing.T) {
	f := NewFocusScope()

	held := make([]Focusable, 0, 4)
	for i := 0; i < 4; i++ {
		held = append(held, &payloadFocusable{buf: make([]byte, payloadSize)})
	}

	f.beginFrame()
	f.Add(held...)
	f.finishBuild()

	before := heapMB()
	held = nil

	// A later frame registers nothing, as happens when the widgets are gone.
	f.beginFrame()
	f.finishBuild()

	after := heapMB()
	runtime.KeepAlive(f)

	freed := before - after
	want := float64(4*payloadSize) / 1e6 * 0.9
	t.Logf("4 x %d MB focusables: heap %.1f -> %.1f MB (freed %.1f MB)",
		payloadSize>>20, before, after, freed)
	if freed < want {
		t.Errorf("focus scope freed only %.1f MB of %.1f MB", freed, float64(4*payloadSize)/1e6)
	}
}

// payloadFocusable is a Focusable carrying a large buffer, standing in for a
// widget that owns a document.
type payloadFocusable struct {
	buf     []byte
	focused bool
}

func (p *payloadFocusable) Focus()                      { p.focused = true }
func (p *payloadFocusable) Blur()                       { p.focused = false }
func (p *payloadFocusable) Focused() bool               { return p.focused }
func (p *payloadFocusable) CapturesTab() bool           { return false }
func (p *payloadFocusable) FocusOnClick() bool          { return false }
func (p *payloadFocusable) FocusEl() *layout.Element    { return nil }
func (p *payloadFocusable) HandleText([]rune)           {}
func (p *payloadFocusable) HandleKeys([]input.KeyEvent) {}
