package ui

import (
	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/layout"
)

// Focusable is a widget that can receive keyboard focus and join tab traversal.
// The method set is identical to components.Focusable, so existing components
// satisfy this structurally — ui need not import components (keeping the
// dependency direction downward).
type Focusable interface {
	Focus()
	Blur()
	Focused() bool
	HandleText(runes []rune)
	HandleKeys(keys []input.KeyEvent)
	CapturesTab() bool
	FocusOnClick() bool
	FocusEl() *layout.Element
}

// FocusScope routes Tab traversal and keyboard input among focusables. Unlike
// the old components.FocusManager (built once, app-owned), a FocusScope is
// rebuilt every frame: components call Add during Layout(c), so beginFrame
// clears the per-frame item list while preserving which widget is focused.
type FocusScope struct {
	items   []Focusable
	focused Focusable // identity preserved across rebuilds
}

// NewFocusScope creates an empty scope. The runtime owns one for the lifetime
// of the app and resets its item list each frame.
func NewFocusScope() *FocusScope { return &FocusScope{} }

// beginFrame clears the per-frame registration list. The focused widget is
// kept; it is re-matched against the new item list as components re-Add.
func (f *FocusScope) beginFrame() { f.items = f.items[:0] }

// Add registers a focusable in tab order for this frame. Idempotent per frame
// in practice because each component Adds itself once during its Layout(c).
func (f *FocusScope) Add(items ...Focusable) {
	for _, it := range items {
		if it != nil {
			f.items = append(f.items, it)
		}
	}
}

// Current returns the focused widget, or nil.
func (f *FocusScope) Current() Focusable { return f.focused }

// Focus moves keyboard focus to w (blurring the previous holder).
func (f *FocusScope) Focus(w Focusable) { f.focusTo(w) }

// EnsureFocus focuses fallback when nothing valid is focused — i.e. no widget is
// focused, or the focused widget did not re-register this frame (e.g. its page
// swapped out). Lets a page declare a default focus target without overriding an
// explicit click-to-focus on another widget.
func (f *FocusScope) EnsureFocus(fallback Focusable) {
	if f.focused == nil || f.indexOf(f.focused) < 0 {
		f.focusTo(fallback)
	}
}

// focusTo blurs the previous widget and focuses w (which may be nil).
func (f *FocusScope) focusTo(w Focusable) {
	if f.focused == w {
		return
	}
	if f.focused != nil {
		f.focused.Blur()
	}
	f.focused = w
	if w != nil {
		w.Focus()
	}
}

// indexOf returns the position of the focused widget in this frame's item list,
// or -1 if it is no longer present.
func (f *FocusScope) indexOf(w Focusable) int {
	for i, it := range f.items {
		if it == w {
			return i
		}
	}
	return -1
}

// Next advances focus to the next registered widget, wrapping around.
func (f *FocusScope) Next() {
	if len(f.items) == 0 {
		return
	}
	i := f.indexOf(f.focused)
	f.focusTo(f.items[(i+1+len(f.items))%len(f.items)])
}

// Prev moves focus to the previous registered widget, wrapping around.
func (f *FocusScope) Prev() {
	if len(f.items) == 0 {
		return
	}
	i := f.indexOf(f.focused)
	if i < 0 {
		i = 0
	}
	f.focusTo(f.items[(i-1+len(f.items))%len(f.items)])
}

// HandleMouse grants focus on primary click when the pointer is inside a
// FocusOnClick widget's FocusEl. Later items win over earlier ones (topmost).
func (f *FocusScope) HandleMouse(m *input.Mouse) {
	if m == nil || !m.Pressed {
		return
	}
	for i := len(f.items) - 1; i >= 0; i-- {
		it := f.items[i]
		if !it.FocusOnClick() {
			continue
		}
		if el := it.FocusEl(); el != nil && el.Frame.Contains(m.X, m.Y) {
			f.focusTo(it)
			return
		}
	}
}

// Route delivers keyboard input to the focused widget. Plain Tab moves focus
// unless the current widget CapturesTab(); Ctrl+Tab always moves focus.
func (f *FocusScope) Route(kb *input.Keyboard) {
	cur := f.focused
	// Don't deliver to a widget that did not re-register this frame (stale focus
	// after a page/tab swap); wait for EnsureFocus or a click to re-establish.
	if cur == nil || kb == nil || f.indexOf(cur) < 0 {
		return
	}
	var keys []input.KeyEvent
	for _, ev := range kb.Keys {
		if ev.Key == input.KeyTab {
			if ev.Mods.Has(input.ModCtrl) || !cur.CapturesTab() {
				if ev.Mods.Has(input.ModShift) {
					f.Prev()
				} else {
					f.Next()
				}
				cur = f.focused
				continue
			}
		}
		keys = append(keys, ev)
	}
	if cur == nil {
		return
	}
	// Text first, then keys: within a single frame, typed characters should be
	// inserted before control keys act on them (e.g. Enter submitting a field
	// must see the text typed the same frame).
	if len(kb.Chars) > 0 {
		cur.HandleText(kb.Chars)
	}
	if len(keys) > 0 {
		cur.HandleKeys(keys)
	}
}
