package components

import (
	"time"

	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/ui"
)

// This file gives the remaining components their ui.View entry point — a single
// Layout(c) method that returns the component's element and, where relevant,
// self-registers focus/animation/overlays through the frame context. Animated
// and overlay-owning components (Editor, Spinner, Toast, Select, Tree, Dropdown,
// Dialog) define Layout in their own files.

// ── Decorative / non-focusable: just expose the element ──────────────────────

func (l *Label) Layout(_ *ui.Ctx) *layout.Element      { return l.El }
func (n *Navigation) Layout(_ *ui.Ctx) *layout.Element { return n.El }
func (a *Alert) Layout(_ *ui.Ctx) *layout.Element      { return a.El }
func (b *Breadcrumb) Layout(_ *ui.Ctx) *layout.Element { return b.El }
func (c *Card) Layout(_ *ui.Ctx) *layout.Element       { return c.El }
func (s *ScrollView) Layout(_ *ui.Ctx) *layout.Element { return s.El }
func (t *Table) Layout(_ *ui.Ctx) *layout.Element      { return t.El }
func (t *TagEdit) Layout(_ *ui.Ctx) *layout.Element    { return t.El }
func (s *Splitter) Layout(_ *ui.Ctx) *layout.Element   { return s.El }
func (l *ListView) Layout(_ *ui.Ctx) *layout.Element   { return l.El }
func (s *Segmented) Layout(_ *ui.Ctx) *layout.Element  { return s.El }

// IconButton is interactive but not Focusable (no keyboard handling); it
// receives clicks through its OnMouse hook during dispatch.
func (b *IconButton) Layout(_ *ui.Ctx) *layout.Element { return b.El }

// ── Focusable: register with the frame's focus scope ─────────────────────────

func (b *Button) Layout(c *ui.Ctx) *layout.Element {
	c.Focus().Add(b)
	return b.El
}

func (t *TabBar) Layout(c *ui.Ctx) *layout.Element {
	c.Focus().Add(t)
	return t.El
}

func (c *Checkbox) Layout(cx *ui.Ctx) *layout.Element {
	cx.Focus().Add(c)
	return c.El
}

// TextField self-registers focus, advances its caret blink, and schedules the
// next repaint while focused — folding the old per-frame Update + app-level
// AnimationWait into one call.
func (tf *TextField) Layout(c *ui.Ctx) *layout.Element {
	c.Focus().Add(tf)
	tf.Update(c.Mouse())
	if tf.focused {
		since := time.Since(tf.blinkStart) % (2 * textFieldBlink)
		wait := textFieldBlink - (since % textFieldBlink)
		c.Animate(wait)
	}
	return tf.El
}

// FileTree registers focus and, while its context menu is open, self-registers
// the menu as an overlay.
func (ft *FileTree) Layout(c *ui.Ctx) *layout.Element {
	c.Focus().Add(ft)
	if ft.tree.menu.Open {
		c.Overlay(ft.tree.menu.El)
	}
	return ft.El()
}
