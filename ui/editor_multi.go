package ui

import (
	"bytes"
	"sort"
	"strings"

	"github.com/mirzakhany/yoga/input"
)

// Multiple selections. The editor's caret and selAnchor stay the primary
// selection, which everything single-caret (search, completion, hover,
// scrolling) follows; extra holds the others. An action with several
// selections runs once per selection, from the last in the document to the
// first, so an edit never moves a selection not yet visited, and the
// visited ones shift by the edit's size.

// cursor is one extra selection: a caret and its anchor, -1 for none.
type cursor struct{ caret, anchor int }

func (c cursor) lo() int {
	if c.anchor >= 0 && c.anchor < c.caret {
		return c.anchor
	}
	return c.caret
}

func (c cursor) hi() int {
	if c.anchor > c.caret {
		return c.anchor
	}
	return c.caret
}

// Cursors is how many selections there are, 1 or more.
func (e *Editor) Cursors() int { return 1 + len(e.extra) }

// Selections returns every selection as [start, end) ranges in document
// order. An empty range is a bare caret.
func (e *Editor) Selections() [][2]int {
	out := make([][2]int, 0, e.Cursors())
	for _, c := range e.allCursors() {
		out = append(out, [2]int{c.lo(), c.hi()})
	}
	sort.Slice(out, func(i, j int) bool { return out[i][0] < out[j][0] })
	return out
}

// ClearCursors drops every selection but the primary one.
func (e *Editor) ClearCursors() { e.extra = nil }

// AddCursorAt adds a caret at byte offset off, keeping the others.
func (e *Editor) AddCursorAt(off int) {
	off = clampInt(off, 0, e.pt.Len())
	e.extra = append(e.extra, cursor{e.caret, e.selAnchor})
	e.caret, e.selAnchor = off, -1
	e.normalizeCursors()
	e.ensureCaretVisible()
}

// AddCursorLine adds a caret on the line above (dir -1) or below (dir 1)
// the outermost caret that way, at the same column where the line allows.
func (e *Editor) AddCursorLine(dir int) {
	all := e.allCursors()
	edge := all[0]
	for _, c := range all {
		if (dir < 0 && c.caret < edge.caret) || (dir > 0 && c.caret > edge.caret) {
			edge = c
		}
	}
	ln := e.lineOf(edge.caret) + dir
	if ln < 0 || ln >= e.pt.LineCount() {
		return
	}
	col := edge.caret - e.pt.LineStart(e.lineOf(edge.caret))
	off := e.pt.LineStart(ln) + min(col, len(e.pt.Line(ln)))
	e.AddCursorAt(off)
}

// AddNextOccurrence selects the word at the caret, or, with text selected,
// adds the next occurrence of it as another selection (⌘D).
func (e *Editor) AddNextOccurrence() {
	if !e.hasSelection() {
		lo, hi := e.wordRangeAt(e.caret)
		if lo < hi {
			e.selAnchor, e.caret = lo, hi
		}
		return
	}
	lo, hi := e.selRange()
	needle := e.pt.Bytes()[lo:hi]
	doc := e.pt.Bytes()
	// Search after the last selection of it, wrapping to the start.
	from := hi
	for _, c := range e.extra {
		if c.hi() > from {
			from = c.hi()
		}
	}
	taken := map[int]bool{lo: true}
	for _, c := range e.extra {
		taken[c.lo()] = true
	}
	for _, start := range []int{from, 0} {
		for at := start; at <= len(doc); {
			i := bytes.Index(doc[at:], needle)
			if i < 0 {
				break
			}
			pos := at + i
			if !taken[pos] {
				e.extra = append(e.extra, cursor{e.caret, e.selAnchor})
				e.selAnchor, e.caret = pos, pos+len(needle)
				e.normalizeCursors()
				e.ensureCaretVisible()
				return
			}
			at = pos + max(1, len(needle))
		}
	}
}

// SelectAllOccurrences selects every occurrence of the selection, or of the
// word at the caret (⌘⇧L).
func (e *Editor) SelectAllOccurrences() {
	if !e.hasSelection() {
		e.AddNextOccurrence()
		if !e.hasSelection() {
			return
		}
	}
	lo, hi := e.selRange()
	needle := e.pt.Bytes()[lo:hi]
	doc := e.pt.Bytes()
	e.extra = e.extra[:0]
	for at := 0; at <= len(doc); {
		i := bytes.Index(doc[at:], needle)
		if i < 0 {
			break
		}
		pos := at + i
		if pos != lo {
			e.extra = append(e.extra, cursor{pos + len(needle), pos})
		}
		at = pos + len(needle)
	}
	e.normalizeCursors()
}

func (e *Editor) allCursors() []cursor {
	return append([]cursor{{e.caret, e.selAnchor}}, e.extra...)
}

// normalizeCursors drops extra selections that repeat or overlap another,
// merging overlapping ranges into the one that comes first.
func (e *Editor) normalizeCursors() {
	if len(e.extra) == 0 {
		return
	}
	n := e.pt.Len()
	all := e.allCursors()
	for i := range all {
		all[i].caret = clampInt(all[i].caret, 0, n)
		if all[i].anchor >= 0 {
			all[i].anchor = clampInt(all[i].anchor, 0, n)
		}
	}
	primary := all[0]
	idx := make([]int, len(all))
	for i := range idx {
		idx[i] = i
	}
	sort.SliceStable(idx, func(a, b int) bool { return all[idx[a]].lo() < all[idx[b]].lo() })
	kept := []cursor{}
	keptPrimary := -1
	for _, i := range idx {
		c := all[i]
		if k := len(kept) - 1; k >= 0 && (c.lo() < kept[k].hi() || c.lo() == kept[k].lo()) {
			// Overlaps the previous: grow it to cover both.
			p := kept[k]
			lo, hi := min(p.lo(), c.lo()), max(p.hi(), c.hi())
			if lo == hi {
				kept[k] = cursor{lo, -1}
			} else {
				kept[k] = cursor{hi, lo}
			}
			if i == 0 {
				keptPrimary = k
			}
			continue
		}
		if i == 0 {
			keptPrimary = len(kept)
		}
		kept = append(kept, c)
	}
	if keptPrimary < 0 {
		keptPrimary = 0
		kept[0] = primary
	}
	e.caret, e.selAnchor = kept[keptPrimary].caret, kept[keptPrimary].anchor
	e.extra = e.extra[:0]
	for i, c := range kept {
		if i != keptPrimary {
			e.extra = append(e.extra, c)
		}
	}
}

// multi runs fn once per selection, with that selection as the primary one.
// edits says fn may change the text: the runs then form one undo step.
func (e *Editor) multi(edits bool, fn func()) {
	all := e.allCursors()
	order := make([]int, len(all))
	for i := range order {
		order[i] = i
	}
	sort.SliceStable(order, func(a, b int) bool { return all[order[a]].lo() > all[order[b]].lo() })
	if edits {
		e.groupSeq++
		e.undoGroup = e.groupSeq
		defer func() { e.undoGroup = 0 }()
	}
	e.inMulti = true
	defer func() { e.inMulti = false }()
	out := make([]cursor, len(all))
	var done []int
	for _, i := range order {
		e.caret, e.selAnchor = all[i].caret, all[i].anchor
		before := e.pt.Len()
		fn()
		out[i] = cursor{e.caret, e.selAnchor}
		if d := e.pt.Len() - before; d != 0 {
			for _, j := range done {
				out[j].caret += d
				if out[j].anchor >= 0 {
					out[j].anchor += d
				}
			}
		}
		done = append(done, i)
	}
	e.caret, e.selAnchor = out[0].caret, out[0].anchor
	e.extra = append(e.extra[:0], out[1:]...)
	e.normalizeCursors()
	e.ensureCaretVisible()
	if len(e.extra) > 0 {
		e.closeCompletion() // completion is for a single caret
	}
}

// multiKey handles a key with several selections and reports whether it did.
func (e *Editor) multiKey(ev input.KeyEvent) bool {
	clip := frameClipboard()
	if ev.Mods.Primary() {
		switch ev.Key {
		case input.KeyC:
			if clip != nil {
				clip.Set(e.selectionsText())
			}
			return true
		case input.KeyX:
			if clip != nil {
				clip.Set(e.selectionsText())
			}
			e.multi(true, func() { e.deleteSelection() })
			return true
		case input.KeyV:
			if clip == nil {
				return true
			}
			text := clip.Get()
			// As many lines as selections: one line each, in document order.
			lines := strings.Split(strings.TrimSuffix(text, "\n"), "\n")
			if len(lines) != e.Cursors() {
				e.multi(true, func() { e.replaceSelection(text, mergeNone) })
				return true
			}
			sels := e.Selections()
			e.multi(true, func() {
				lo, _ := e.selRange()
				for k, s := range sels {
					if s[0] == lo {
						e.replaceSelection(lines[k], mergeNone)
						return
					}
				}
			})
			return true
		case input.KeyA, input.KeyZ, input.KeyY:
			e.extra = nil
			return false // select all, undo and redo work on the primary
		}
		return false
	}
	switch ev.Key {
	case input.KeyEscape:
		e.extra = nil
		return true
	case input.KeyEnter, input.KeyTab, input.KeyBackspace, input.KeyDelete:
		e.multi(true, func() { e.HandleKeys([]input.KeyEvent{ev}) })
		return true
	case input.KeyLeft, input.KeyRight, input.KeyUp, input.KeyDown, input.KeyHome, input.KeyEnd:
		e.multi(false, func() { e.HandleKeys([]input.KeyEvent{ev}) })
		return true
	}
	return false
}

// selectionsText is the selected text of every selection, one per line.
func (e *Editor) selectionsText() string {
	doc := e.pt.Bytes()
	var parts []string
	for _, s := range e.Selections() {
		if s[0] < s[1] {
			parts = append(parts, string(doc[s[0]:s[1]]))
		}
	}
	return strings.Join(parts, "\n")
}

func clampInt(v, lo, hi int) int {
	return max(lo, min(v, hi))
}

func (e *Editor) extraCarets() []int {
	out := make([]int, len(e.extra))
	for i, c := range e.extra {
		out[i] = c.caret
	}
	return out
}

// CaretStyle is the shape the caret is drawn in.
type CaretStyle int

const (
	CaretBar       CaretStyle = iota // a thin bar before the character
	CaretBlock                       // a block over the character
	CaretUnderline                   // a line under the character
)

// Replace replaces bytes [lo, hi) with text, as one undoable edit, and puts
// the caret after the new text. It does nothing in a read-only editor.
func (e *Editor) Replace(lo, hi int, text string) {
	lo, hi = e.clampOffset(lo), e.clampOffset(hi)
	if hi < lo {
		lo, hi = hi, lo
	}
	e.extra = nil
	e.applyEdit(lo, hi-lo, text, mergeNone)
	e.ensureCaretVisible()
}

// UndoGroup runs fn, and the edits it makes undo and redo as one step.
func (e *Editor) UndoGroup(fn func()) {
	if e.undoGroup != 0 {
		fn() // already inside a group
		return
	}
	e.groupSeq++
	e.undoGroup = e.groupSeq
	defer func() { e.undoGroup = 0 }()
	fn()
}
