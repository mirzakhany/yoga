package components

import (
	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/layout"
)

// Focusable is a widget that can receive keyboard focus and participate in tab
// traversal. Implementations declare whether they capture plain Tab (e.g. the
// editor for indent) and whether a click inside FocusEl should grant focus.
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

// FocusManager owns an ordered ring of focusable widgets and routes keyboard
// input to the currently focused one.
type FocusManager struct {
	items []Focusable
	cur   int // index into items, -1 when nothing is focused
}

// NewFocusManager creates an empty focus manager.
func NewFocusManager() *FocusManager {
	return &FocusManager{cur: -1}
}

// Add appends focusables in tab order.
func (fm *FocusManager) Add(items ...Focusable) {
	fm.items = append(fm.items, items...)
}

// Current returns the focused widget, or nil.
func (fm *FocusManager) Current() Focusable {
	if fm.cur < 0 || fm.cur >= len(fm.items) {
		return nil
	}
	return fm.items[fm.cur]
}

// Focus moves keyboard focus to f, blurring the previous widget.
func (fm *FocusManager) Focus(f Focusable) {
	if f == nil {
		return
	}
	for i, item := range fm.items {
		if item == f {
			fm.focusIndex(i)
			return
		}
	}
}

func (fm *FocusManager) focusIndex(i int) {
	if i < 0 || i >= len(fm.items) {
		return
	}
	if fm.cur >= 0 && fm.cur < len(fm.items) && fm.cur != i {
		fm.items[fm.cur].Blur()
	}
	fm.cur = i
	fm.items[i].Focus()
}

// Next advances focus to the next widget, wrapping around.
func (fm *FocusManager) Next() {
	if len(fm.items) == 0 {
		return
	}
	next := 0
	if fm.cur >= 0 {
		next = (fm.cur + 1) % len(fm.items)
	}
	fm.focusIndex(next)
}

// Prev moves focus to the previous widget, wrapping around.
func (fm *FocusManager) Prev() {
	if len(fm.items) == 0 {
		return
	}
	prev := len(fm.items) - 1
	if fm.cur >= 0 {
		prev = (fm.cur - 1 + len(fm.items)) % len(fm.items)
	}
	fm.focusIndex(prev)
}

// HandleMouse grants focus on primary click when the pointer is inside a
// FocusOnClick widget's FocusEl. Later items in the ring win over earlier ones
// so nested/overlapping regions behave predictably.
func (fm *FocusManager) HandleMouse(m *input.Mouse) {
	if m == nil || !m.Pressed {
		return
	}
	for i := len(fm.items) - 1; i >= 0; i-- {
		item := fm.items[i]
		if !item.FocusOnClick() {
			continue
		}
		el := item.FocusEl()
		if el != nil && el.Frame.Contains(m.X, m.Y) {
			fm.focusIndex(i)
			return
		}
	}
}

// Route delivers keyboard input to the focused widget. Tab traversal is handled
// here: plain Tab moves focus unless the current widget CapturesTab(); Ctrl+Tab
// always moves focus.
func (fm *FocusManager) Route(kb *input.Keyboard) {
	cur := fm.Current()
	if cur == nil || kb == nil {
		return
	}

	var keys []input.KeyEvent
	for _, ev := range kb.Keys {
		if ev.Key == input.KeyTab {
			force := ev.Mods.Has(input.ModCtrl)
			if force || !cur.CapturesTab() {
				if ev.Mods.Has(input.ModShift) {
					fm.Prev()
				} else {
					fm.Next()
				}
				cur = fm.Current()
				continue
			}
		}
		keys = append(keys, ev)
	}

	if cur == nil {
		return
	}
	if len(keys) > 0 {
		cur.HandleKeys(keys)
	}
	if len(kb.Chars) > 0 {
		cur.HandleText(kb.Chars)
	}
}
