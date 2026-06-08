package components

import (
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/mirzakhany/yoga/highlight"
	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
	"github.com/mirzakhany/yoga/text"
	"github.com/mirzakhany/yoga/theme"
)

// Editor is a virtualized, syntax-highlighted, *editable* code view designed for
// large files. Three ideas keep it fast regardless of document size:
//
//  1. Storage: text lives in a text.PieceTable, so edits never copy the whole
//     file.
//  2. Virtualization: paint() only generates quads for the lines actually
//     visible inside the editor's Frame — a million-line file still emits a few
//     dozen rows of geometry per frame.
//  3. Async highlighting: a highlight.Highlighter parses on a worker goroutine;
//     the editor polls finished token sets and folds them into a per-byte color
//     table that the painter samples per glyph.
//
// Editing state is a caret (a byte offset into the document) plus an optional
// selection anchor. Every mutation funnels through applyEdit, which keeps the
// piece table, the undo stack, the modified flag, and the highlighter in sync.
type Editor struct {
	El    *layout.Element
	theme *theme.Theme
	engine *shape.Engine
	pt    *text.PieceTable
	hl    highlight.Highlighter
	clip  input.Clipboard

	Path string // file path, empty for an unsaved scratch buffer

	// ScrollPx / ScrollX are scroll positions in pixels (updated by scrollbars).
	// ContentHeight / ContentWidth are total document extents for scrollbar thumbs.
	ScrollPx, ScrollX       float32
	ContentHeight           float32
	ContentWidth            float32

	viewport *layout.Element // clipped text area (paint + mouse)
	vbar     *Scrollbar
	hbar     *Scrollbar

	// Editing state.
	caret      int // byte offset of the caret
	selAnchor  int // byte offset where a selection began, or -1 for none
	dragging   bool
	blinkStart time.Time
	modified   bool
	focused    bool

	// parseUntil bounds a short window after an edit during which the runtime
	// should poll for highlight results quickly (so colors appear promptly).
	// Zero when no parse is expected (idle, or a Noop highlighter).
	parseUntil time.Time

	// Undo/redo. Each op describes a single replacement so it can be inverted.
	undo        []editOp
	redo        []editOp
	canCoalesce bool // last op was contiguous typing and may absorb the next rune

	// tokens is the latest syntax highlight result (sorted by Start). Queried at
	// paint time instead of a per-byte table so large files stay lightweight.
	tokens []highlight.Token

	contentSizeDirty bool

	gutterW float32
	lineH   float32
	textPad float32
	goalX   float32 // pixel x for up/down caret movement
}

// editOp records a replacement of [pos, pos+len(deleted)) with inserted, plus
// the caret before/after, which is enough to undo or redo it.
type editOp struct {
	pos         int
	deleted     string
	inserted    string
	caretBefore int
	caretAfter  int
}

// tabWidth is the number of columns a tab advances to (next tab stop).
const tabWidth = 4

const editorBarSize = 14 // scrollbar thickness (vertical and horizontal)

// NewEditor creates a scratch editor over initial content with the given
// highlighter. NewEditorFor is preferred for real files.
func NewEditor(engine *shape.Engine, theme *theme.Theme, initial []byte, hl highlight.Highlighter) *Editor {
	return newEditor(engine, theme, "", initial, hl, nil)
}

func NewEditorFor(engine *shape.Engine, theme *theme.Theme, path string, content []byte, clip input.Clipboard) *Editor {
	return newEditor(engine, theme, path, content, highlight.ForPath(path), clip)
}

func newEditor(engine *shape.Engine, theme *theme.Theme, path string, content []byte, hl highlight.Highlighter, clip input.Clipboard) *Editor {
	m := engine.MetricsMono()
	e := &Editor{
		theme:      theme,
		engine:     engine,
		pt:         text.New(content),
		hl:         hl,
		clip:       clip,
		Path:       path,
		selAnchor:  -1,
		blinkStart: time.Now(),
		gutterW:    52,
		lineH:      m.LineHeight,
		textPad:           8,
		contentSizeDirty:  true,
	}
	e.viewport = layout.New(layout.Box().FlexGrow(1))
	e.viewport.Clip = true
	e.viewport.Paint = e.paint
	e.viewport.OnMouse = e.onMouse

	e.vbar = NewScrollbar(theme, &e.ScrollPx, &e.ContentHeight, editorBarSize)
	e.hbar = NewScrollbarAxis(theme, Horizontal, &e.ScrollX, &e.ContentWidth, editorBarSize)

	e.El = layout.New(layout.Box().FlexGrow(1), e.viewport, e.vbar.El, e.hbar.El)
	e.hl.Update(e.pt.Bytes())
	e.markParsePending()
	return e
}

// markParsePending opens a brief fast-poll window so a freshly requested parse's
// result is picked up within a frame or two. Skipped for the Noop highlighter,
// which never delivers tokens (so plain-text buffers stay fully idle).
func (e *Editor) markParsePending() {
	if _, noop := e.hl.(highlight.Noop); noop {
		return
	}
	e.parseUntil = time.Now().Add(1500 * time.Millisecond)
}

// Close releases the highlighter's worker and native resources.
func (e *Editor) Close() { e.hl.Close() }

// Bytes returns the current document content.
func (e *Editor) Bytes() []byte { return e.pt.Bytes() }

// Modified reports whether there are unsaved changes.
func (e *Editor) Modified() bool { return e.modified }

// MarkSaved clears the modified flag (call after writing to disk).
func (e *Editor) MarkSaved() { e.modified = false }

// Update polls the highlighter, recomputes scroll extents, and drives scrollbars.
// Call once per frame after layout (pass the frame mouse state for wheel/drag).
func (e *Editor) Update(m *input.Mouse) {
	if toks, ok := e.hl.Poll(); ok {
		e.tokens = toks
		e.parseUntil = time.Time{} // result consumed; stop fast-polling
	}
	if e.contentSizeDirty {
		e.recomputeContentSize()
	}
	if m != nil {
		_, _, vShow, hShow := e.scrollMetrics()
		e.syncScrollbarLayout(vShow, hShow)
		vp := e.contentViewport()
		e.vbar.Update(m, vp)
		e.hbar.Update(m, vp)
		e.clampScroll()
	}
}

const editorLargeDocLines = 500

func (e *Editor) recomputeContentSize() {
	e.ContentHeight = float32(e.pt.LineCount()) * e.lineH
	lc := e.pt.LineCount()
	var maxTextW float32
	if lc > editorLargeDocLines {
		// Shaping every line blocks the UI on multi-thousand-line files; approximate
		// from the longest line byte length and a single glyph width sample.
		cellW, _ := e.engine.MeasureMono("n")
		maxTextW = float32(e.pt.MaxLineByteLen()) * cellW
	} else {
		for ln := 0; ln < lc; ln++ {
			if w := e.lineTextWidth(ln); w > maxTextW {
				maxTextW = w
			}
		}
	}
	e.ContentWidth = e.gutterW + e.textPad + maxTextW
	e.contentSizeDirty = false
}

func (e *Editor) shapedLine(ln int) shape.Line {
	return e.engine.LineMono(e.pt.Line(ln))
}

func (e *Editor) lineTextWidth(line int) float32 {
	return e.shapedLine(line).Width
}

func (e *Editor) scrollMetrics() (clientW, clientH float32, vShow, hShow bool) {
	f := e.viewport.Frame
	clientW, clientH = f.W, f.H
	vShow = e.ContentHeight > clientH
	if vShow {
		clientW = f.W - editorBarSize
	}
	hShow = e.ContentWidth > clientW
	if hShow {
		clientH = f.H - editorBarSize
	}
	if e.ContentHeight > clientH {
		vShow = true
		clientW = f.W - editorBarSize
		hShow = e.ContentWidth > clientW
		if hShow {
			clientH = f.H - editorBarSize
		}
	}
	return clientW, clientH, vShow, hShow
}

func (e *Editor) contentViewport() render.Rect {
	f := e.viewport.Frame
	cw, ch, _, _ := e.scrollMetrics()
	return render.Rect{X: f.X, Y: f.Y, W: cw, H: ch}
}

func (e *Editor) syncScrollbarLayout(vShow, hShow bool) {
	if vShow {
		bottom := float32(0)
		if hShow {
			bottom = editorBarSize
		}
		e.vbar.El.Style = layout.Box().W(editorBarSize).AbsTop(0).AbsRight(0).AbsBottom(bottom)
		e.vbar.El.ReapplyStyle()
	}
	if hShow {
		right := float32(0)
		if vShow {
			right = editorBarSize
		}
		e.hbar.El.Style = layout.Box().H(editorBarSize).AbsLeft(0).AbsRight(right).AbsBottom(0)
		e.hbar.El.ReapplyStyle()
	}
}

func (e *Editor) textClientW(clientW float32) float32 {
	w := clientW - e.gutterW - e.textPad
	if w < 0 {
		return 0
	}
	return w
}

func (e *Editor) maxScrollX(clientW float32) float32 {
	textExtent := e.ContentWidth - e.gutterW - e.textPad
	return f32max(0, textExtent-e.textClientW(clientW))
}

func (e *Editor) clampScroll() {
	clientW, clientH, _, _ := e.scrollMetrics()
	e.ScrollPx = clampf(e.ScrollPx, 0, f32max(0, e.ContentHeight-clientH))
	e.ScrollX = clampf(e.ScrollX, 0, e.maxScrollX(clientW))
}

func (e *Editor) overScrollbar(m *input.Mouse) bool {
	_, _, vShow, hShow := e.scrollMetrics()
	if vShow && e.vbar.El.Frame.Contains(m.X, m.Y) {
		return true
	}
	if hShow && e.hbar.El.Frame.Contains(m.X, m.Y) {
		return true
	}
	return false
}

// Focus grants keyboard focus to the editor.
func (e *Editor) Focus() {
	e.focused = true
	e.blinkStart = time.Now()
}

// Blur removes keyboard focus from the editor.
func (e *Editor) Blur() { e.focused = false }

// Focused reports whether the editor has keyboard focus.
func (e *Editor) Focused() bool { return e.focused }

// CapturesTab reports that plain Tab should indent rather than move focus.
func (e *Editor) CapturesTab() bool { return true }

// FocusOnClick reports that clicking the editor should grant focus.
func (e *Editor) FocusOnClick() bool { return true }

// FocusEl returns the element used for click-to-focus hit testing.
func (e *Editor) FocusEl() *layout.Element { return e.El }

// AnimationWait implements the runtime's optional animation hook. When focused,
// the caret blinks; the returned delay is the time until the next blink toggle,
// shortened to ~16ms while dragging a selection or while a highlight parse is
// still expected, so those stay responsive.
func (e *Editor) AnimationWait() (time.Duration, bool) {
	if !e.focused && !time.Now().Before(e.parseUntil) {
		return 0, false
	}
	const half = 500 * time.Millisecond // caretVisible toggles every half period
	since := time.Since(e.blinkStart) % (2 * half)
	wait := half - (since % half)
	if e.dragging || time.Now().Before(e.parseUntil) {
		if fast := 16 * time.Millisecond; wait > fast {
			wait = fast
		}
	}
	return wait, true
}

// ---------------------------------------------------------------------------
// Editing core
// ---------------------------------------------------------------------------

// applyEdit replaces [pos, pos+delLen) with ins, records an undo op (coalescing
// contiguous single-rune typing), reparses, and marks the buffer modified. It
// returns the caret position just after the inserted text.
func (e *Editor) applyEdit(pos, delLen int, ins string, coalesceTyping bool) int {
	b := e.pt.Bytes()
	if pos < 0 {
		pos = 0
	}
	if pos+delLen > len(b) {
		delLen = len(b) - pos
	}
	deleted := string(b[pos : pos+delLen])
	startPt := e.pointOf(pos)
	oldEndPt := e.pointOf(pos + delLen)

	op := editOp{
		pos:         pos,
		deleted:     deleted,
		inserted:    ins,
		caretBefore: e.caret,
		caretAfter:  pos + len(ins),
	}

	if delLen > 0 {
		e.pt.Delete(pos, delLen)
	}
	if len(ins) > 0 {
		e.pt.Insert(pos, []byte(ins))
	}

	// Coalesce: append this typed text onto the previous op when it is a pure,
	// contiguous insertion (so a run of keystrokes is one undo step).
	merged := false
	if coalesceTyping && e.canCoalesce && delLen == 0 && len(e.undo) > 0 && !strings.Contains(ins, "\n") {
		last := &e.undo[len(e.undo)-1]
		if last.deleted == "" && last.pos+len(last.inserted) == pos {
			last.inserted += ins
			last.caretAfter = op.caretAfter
			merged = true
		}
	}
	if !merged {
		e.undo = append(e.undo, op)
	}
	e.canCoalesce = coalesceTyping && delLen == 0 && !strings.Contains(ins, "\n")

	e.redo = e.redo[:0]
	e.modified = true
	e.contentSizeDirty = true
	e.selAnchor = -1
	e.caret = op.caretAfter
	e.afterMutation(highlight.Edit{
		StartByte: pos, OldEndByte: pos + delLen, NewEndByte: pos + len(ins),
		Start: startPt, OldEnd: oldEndPt, NewEnd: e.pointOf(pos + len(ins)),
	})
	return e.caret
}

// afterMutation requests an incremental reparse, resets the caret blink, and
// keeps the caret on screen.
func (e *Editor) afterMutation(edit highlight.Edit) {
	e.hl.UpdateEdit(e.pt.Bytes(), edit)
	e.markParsePending()
	e.blinkStart = time.Now()
	e.ensureCaretVisible()
}

// selRange returns the ordered [lo, hi) selection, or (caret, caret) if none.
func (e *Editor) selRange() (int, int) {
	if e.selAnchor < 0 || e.selAnchor == e.caret {
		return e.caret, e.caret
	}
	if e.selAnchor < e.caret {
		return e.selAnchor, e.caret
	}
	return e.caret, e.selAnchor
}

func (e *Editor) hasSelection() bool {
	return e.selAnchor >= 0 && e.selAnchor != e.caret
}

// replaceSelection replaces the current selection (or nothing) with s.
func (e *Editor) replaceSelection(s string, coalesceTyping bool) {
	lo, hi := e.selRange()
	e.applyEdit(lo, hi-lo, s, coalesceTyping)
}

func (e *Editor) deleteSelection() bool {
	if !e.hasSelection() {
		return false
	}
	lo, hi := e.selRange()
	e.applyEdit(lo, hi-lo, "", false)
	return true
}

// Undo reverts the most recent edit.
func (e *Editor) Undo() {
	if len(e.undo) == 0 {
		return
	}
	op := e.undo[len(e.undo)-1]
	e.undo = e.undo[:len(e.undo)-1]
	startPt := e.pointOf(op.pos)
	oldEndPt := e.pointOf(op.pos + len(op.inserted))
	if len(op.inserted) > 0 {
		e.pt.Delete(op.pos, len(op.inserted))
	}
	if len(op.deleted) > 0 {
		e.pt.Insert(op.pos, []byte(op.deleted))
	}
	e.redo = append(e.redo, op)
	e.caret = op.caretBefore
	e.selAnchor = -1
	e.canCoalesce = false
	e.modified = true
	e.afterMutation(highlight.Edit{
		StartByte: op.pos, OldEndByte: op.pos + len(op.inserted), NewEndByte: op.pos + len(op.deleted),
		Start: startPt, OldEnd: oldEndPt, NewEnd: e.pointOf(op.pos + len(op.deleted)),
	})
}

// Redo re-applies the most recently undone edit.
func (e *Editor) Redo() {
	if len(e.redo) == 0 {
		return
	}
	op := e.redo[len(e.redo)-1]
	e.redo = e.redo[:len(e.redo)-1]
	startPt := e.pointOf(op.pos)
	oldEndPt := e.pointOf(op.pos + len(op.deleted))
	if len(op.deleted) > 0 {
		e.pt.Delete(op.pos, len(op.deleted))
	}
	if len(op.inserted) > 0 {
		e.pt.Insert(op.pos, []byte(op.inserted))
	}
	e.undo = append(e.undo, op)
	e.caret = op.caretAfter
	e.selAnchor = -1
	e.canCoalesce = false
	e.modified = true
	e.afterMutation(highlight.Edit{
		StartByte: op.pos, OldEndByte: op.pos + len(op.deleted), NewEndByte: op.pos + len(op.inserted),
		Start: startPt, OldEnd: oldEndPt, NewEnd: e.pointOf(op.pos + len(op.inserted)),
	})
}

// ---------------------------------------------------------------------------
// Input handling
// ---------------------------------------------------------------------------

// HandleText inserts text-producing characters (from the OS text input path).
func (e *Editor) HandleText(runes []rune) {
	if len(runes) == 0 {
		return
	}
	e.replaceSelection(string(runes), true)
}

// HandleKeys processes navigation, editing, and shortcut keys for one frame.
func (e *Editor) HandleKeys(keys []input.KeyEvent) {
	for _, ev := range keys {
		shift := ev.Mods.Has(input.ModShift)
		if ev.Mods.Primary() {
			switch ev.Key {
			case input.KeyA:
				e.selAnchor = 0
				e.caret = e.pt.Len()
			case input.KeyC:
				e.copy()
			case input.KeyX:
				e.cut()
			case input.KeyV:
				e.paste()
			case input.KeyZ:
				if shift {
					e.Redo()
				} else {
					e.Undo()
				}
			}
			continue
		}
		switch ev.Key {
		case input.KeyEnter:
			e.replaceSelection("\n", false)
		case input.KeyTab:
			e.replaceSelection("\t", false)
		case input.KeyBackspace:
			e.backspace()
		case input.KeyDelete:
			e.deleteForward()
		case input.KeyLeft:
			e.moveCluster(-1, shift)
		case input.KeyRight:
			e.moveCluster(1, shift)
		case input.KeyUp:
			e.moveVertical(-1, shift)
		case input.KeyDown:
			e.moveVertical(1, shift)
		case input.KeyHome:
			ln := e.lineOf(e.caret)
			e.moveTo(e.pt.LineStart(ln), shift)
		case input.KeyEnd:
			ln := e.lineOf(e.caret)
			e.moveTo(e.pt.LineStart(ln)+len(e.pt.Line(ln)), shift)
		}
	}
}

func (e *Editor) backspace() {
	if e.deleteSelection() {
		return
	}
	if e.caret == 0 {
		return
	}
	prev := e.prevRune(e.caret)
	e.applyEdit(prev, e.caret-prev, "", false)
}

func (e *Editor) deleteForward() {
	if e.deleteSelection() {
		return
	}
	next := e.nextRune(e.caret)
	if next == e.caret {
		return
	}
	e.applyEdit(e.caret, next-e.caret, "", false)
}

func (e *Editor) copy() {
	if e.clip == nil || !e.hasSelection() {
		return
	}
	lo, hi := e.selRange()
	e.clip.Set(string(e.pt.Bytes()[lo:hi]))
}

func (e *Editor) cut() {
	if e.clip == nil || !e.hasSelection() {
		return
	}
	lo, hi := e.selRange()
	e.clip.Set(string(e.pt.Bytes()[lo:hi]))
	e.applyEdit(lo, hi-lo, "", false)
}

func (e *Editor) paste() {
	if e.clip == nil {
		return
	}
	s := e.clip.Get()
	if s == "" {
		return
	}
	e.replaceSelection(s, false)
}

// moveTo moves the caret to off, extending the selection when extend is set and
// collapsing it otherwise.
func (e *Editor) moveTo(off int, extend bool) {
	if off < 0 {
		off = 0
	}
	if n := e.pt.Len(); off > n {
		off = n
	}
	if extend {
		if e.selAnchor < 0 {
			e.selAnchor = e.caret
		}
	} else {
		e.selAnchor = -1
	}
	e.caret = off
	e.canCoalesce = false
	e.blinkStart = time.Now()
	e.ensureCaretVisible()
}

func (e *Editor) moveCluster(dir int, extend bool) {
	ln := e.lineOf(e.caret)
	ls := e.pt.LineStart(ln)
	line := e.shapedLine(ln)
	off := e.caret - ls
	var next int
	if dir < 0 {
		next = line.PrevCluster(off)
	} else {
		next = line.NextCluster(off, len(e.pt.Line(ln)))
	}
	e.goalX = line.XForByte(next)
	e.moveTo(ls+next, extend)
}

func (e *Editor) moveVertical(dir int, extend bool) {
	line, _ := e.lineColOf(e.caret)
	if e.goalX == 0 {
		e.goalX = e.caretXInLine(e.lineOf(e.caret), e.caret)
	}
	line += dir
	if line < 0 {
		line = 0
	}
	if lc := e.pt.LineCount(); line >= lc {
		line = lc - 1
	}
	ls := e.pt.LineStart(line)
	ln := e.shapedLine(line)
	off := ln.ByteForX(e.goalX)
	e.moveTo(ls+off, extend)
}

// ---------------------------------------------------------------------------
// Offset / line-column geometry
// ---------------------------------------------------------------------------

func (e *Editor) prevRune(off int) int {
	if off <= 0 {
		return 0
	}
	b := e.pt.Bytes()
	_, sz := utf8.DecodeLastRune(b[:off])
	return off - sz
}

func (e *Editor) nextRune(off int) int {
	b := e.pt.Bytes()
	if off >= len(b) {
		return len(b)
	}
	_, sz := utf8.DecodeRune(b[off:])
	return off + sz
}

// pointOf maps a byte offset to a Tree-sitter row/column (byte column within row).
func (e *Editor) pointOf(off int) highlight.Pt {
	line := e.lineOf(off)
	return highlight.Pt{Row: line, Col: off - e.pt.LineStart(line)}
}

// lineOf returns the line index containing byte offset off (binary search over
// the piece table's line starts).
func (e *Editor) lineOf(off int) int {
	lo, hi := 0, e.pt.LineCount()-1
	for lo < hi {
		mid := (lo + hi + 1) / 2
		if e.pt.LineStart(mid) <= off {
			lo = mid
		} else {
			hi = mid - 1
		}
	}
	return lo
}

func (e *Editor) lineColOf(off int) (line, col int) {
	line = e.lineOf(off)
	ls := e.pt.LineStart(line)
	if off < ls {
		off = ls
	}
	return line, utf8.RuneCount(e.pt.Bytes()[ls:off])
}

func (e *Editor) offsetOf(line, col int) int {
	if line < 0 {
		line = 0
	}
	if lc := e.pt.LineCount(); line >= lc {
		line = lc - 1
	}
	ls := e.pt.LineStart(line)
	txt := e.pt.Line(line)
	n := 0
	for i := range txt { // ranges over rune start byte indices
		if n == col {
			return ls + i
		}
		n++
	}
	return ls + len(txt)
}

// ---------------------------------------------------------------------------
// Mouse: click to place caret, drag to select
// ---------------------------------------------------------------------------

func (e *Editor) onMouse(el *layout.Element, m *input.Mouse) {
	if e.overScrollbar(m) {
		return
	}
	f := el.Frame
	if m.Pressed && f.Contains(m.X, m.Y) {
		off := e.offsetAtPoint(m.X, m.Y)
		if m.Consumed { // a higher widget already took it
			return
		}
		e.caret = off
		e.selAnchor = off
		e.dragging = true
		e.canCoalesce = false
		e.blinkStart = time.Now()
		m.Consumed = true
	}
	if e.dragging && m.Down {
		e.caret = e.offsetAtPoint(m.X, m.Y)
	}
	if !m.Down {
		e.dragging = false
	}
}

func (e *Editor) caretXInLine(ln, caret int) float32 {
	ls := e.pt.LineStart(ln)
	return e.shapedLine(ln).XForByte(caret - ls)
}

func (e *Editor) offsetAtPoint(px, py float32) int {
	f := e.viewport.Frame
	line := int((py - f.Y + e.ScrollPx) / e.lineH)
	if line < 0 {
		line = 0
	}
	if lc := e.pt.LineCount(); line >= lc {
		line = lc - 1
	}
	ls := e.pt.LineStart(line)
	x0 := f.X + e.gutterW + e.textPad - e.ScrollX
	ln := e.shapedLine(line)
	off := ln.ByteForX(px - x0)
	e.goalX = ln.XForByte(off)
	return ls + off
}

func (e *Editor) ensureCaretVisible() {
	_, clientH, _, _ := e.scrollMetrics()
	if clientH <= 0 {
		return
	}
	line := e.lineOf(e.caret)
	top := float32(line) * e.lineH
	if top < e.ScrollPx {
		e.ScrollPx = top
	} else if top+e.lineH > e.ScrollPx+clientH {
		e.ScrollPx = top + e.lineH - clientH
	}
	if e.ScrollPx < 0 {
		e.ScrollPx = 0
	}

	clientW, _, _, _ := e.scrollMetrics()
	textW := e.textClientW(clientW)
	if textW <= 0 {
		return
	}
	relX := e.caretXInLine(line, e.caret)
	if relX < e.ScrollX {
		e.ScrollX = relX
	} else if relX+2 > e.ScrollX+textW {
		e.ScrollX = relX + 2 - textW
	}
	if e.ScrollX < 0 {
		e.ScrollX = 0
	}
}

// ---------------------------------------------------------------------------
// Painting
// ---------------------------------------------------------------------------

func (e *Editor) colorAt(byteOffset int) highlight.ColorClass {
	tok := e.tokens
	lo, hi := 0, len(tok)
	for lo < hi {
		mid := (lo + hi) / 2
		if byteOffset < tok[mid].Start {
			hi = mid
		} else if byteOffset >= tok[mid].End {
			lo = mid + 1
		} else {
			return tok[mid].Class
		}
	}
	return highlight.ClassDefault
}

func (e *Editor) paint(dl *render.DrawList, _ *shape.Engine) {
	f := e.viewport.Frame
	vp := e.contentViewport()
	gutter := render.Rect{X: f.X, Y: f.Y, W: e.gutterW, H: f.H}
	textArea := render.Rect{X: f.X + e.gutterW, Y: vp.Y, W: vp.W - e.gutterW, H: vp.H}
	m := e.engine.MetricsMono()

	dl.AddRect(f, e.theme.Background)

	first := int(e.ScrollPx / e.lineH)
	if first < 0 {
		first = 0
	}
	visibleCount := int(vp.H/e.lineH) + 2
	last := first + visibleCount
	if lc := e.pt.LineCount(); last > lc {
		last = lc
	}

	selLo, selHi := e.selRange()
	hasSel := selLo != selHi
	x0 := f.X + e.gutterW + e.textPad - e.ScrollX

	dl.PushClip(vp)
	dl.PushClip(textArea)
	for ln := first; ln < last; ln++ {
		y := f.Y + float32(ln)*e.lineH - e.ScrollPx
		if y >= vp.Y+vp.H {
			break
		}
		ls := e.pt.LineStart(ln)
		txt := e.pt.Line(ln)
		lineEnd := ls + len(txt)
		sline := e.shapedLine(ln)

		if hasSel && selLo <= lineEnd && selHi >= ls {
			a := maxInt(selLo, ls)
			b := minInt(selHi, lineEnd)
			for _, rect := range sline.SelectionRects(a-ls, b-ls, y, e.lineH) {
				dl.AddRect(render.Rect{X: x0 + rect[0], Y: rect[1], W: rect[2], H: rect[3]}, e.theme.Selection)
			}
		}

		topY := y + (e.lineH-m.LineHeight)/2
		e.engine.DrawLineGlyphs(dl, sline, x0, topY, func(byteOff int) render.Color {
			return e.theme.SyntaxColor(e.colorAt(ls + byteOff))
		})
	}
	if e.focused && e.caretVisible() {
		cl := e.lineOf(e.caret)
		cx := x0 + e.caretXInLine(cl, e.caret)
		cy := f.Y + float32(cl)*e.lineH - e.ScrollPx
		dl.AddRect(render.Rect{X: cx, Y: cy + (e.lineH-m.LineHeight)/2, W: 2, H: m.LineHeight}, e.theme.Accent)
	}
	dl.PopClip()
	dl.PopClip()

	dl.AddRect(gutter, e.theme.PanelAlt)
	dl.PushClip(gutter)
	for ln := first; ln < last; ln++ {
		y := f.Y + float32(ln)*e.lineH - e.ScrollPx
		if y >= vp.Y+vp.H {
			break
		}
		num := strconv.Itoa(ln + 1)
		numW, _ := e.engine.MeasureMono(num)
		e.engine.DrawStringTopMono(dl, num, f.X+e.gutterW-numW-8, y+(e.lineH-m.LineHeight)/2, e.theme.TextDim)
	}
	dl.PopClip()
}

// caretVisible returns the blink phase (on for the first half of each cycle).
func (e *Editor) caretVisible() bool {
	if !e.focused {
		return false
	}
	const period = 1000 * time.Millisecond
	since := time.Since(e.blinkStart) % period
	return since < period/2
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
