package ui

import (
	"runtime"
	"testing"
	"weak"

	"github.com/yogaui/yoga/input"
	"github.com/yogaui/yoga/layout"
	"github.com/yogaui/yoga/render"
	"github.com/yogaui/yoga/shape"
)

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
