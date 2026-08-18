package ui

import (
	"time"

	"github.com/mirzakhany/yoga/layout"
)

func (n *Navigation) Layout(_ *Ctx) *layout.Element { return n.El }
func (a *Alert) Layout(_ *Ctx) *layout.Element      { return a.El }
func (b *Breadcrumb) Layout(_ *Ctx) *layout.Element { return b.El }
func (s *ScrollView) Layout(_ *Ctx) *layout.Element { return s.El }
func (t *Table) Layout(c *Ctx) *layout.Element {
	t.Update(c.Mouse())
	if e := t.EditEl(); e != nil {
		c.Overlay(e)
	}
	return t.El
}
func (t *TagEdit) Layout(c *Ctx) *layout.Element {
	t.Update(c.Mouse())
	return t.El
}
func (l *ListView) Layout(_ *Ctx) *layout.Element  { return l.El }
func (s *Segmented) Layout(_ *Ctx) *layout.Element { return s.El }

func (t *TabBar) Layout(c *Ctx) *layout.Element {
	c.Focus().Add(t)
	return t.El
}

func (tf *TextInput) Layout(c *Ctx) *layout.Element {
	c.Focus().Add(tf)
	tf.Update(c.Mouse())
	if tf.focused {
		since := time.Since(tf.blinkStart) % (2 * textFieldBlink)
		wait := textFieldBlink - (since % textFieldBlink)
		c.Animate(wait)
	}
	return tf.El
}

func (ft *FileTree) Layout(c *Ctx) *layout.Element {
	c.Focus().Add(ft)
	ft.Update(c.Mouse())
	if ft.tree.menu.Open {
		c.Overlay(ft.tree.menu.El)
	}
	return ft.El()
}

func (r *Radio) Layout(c *Ctx) *layout.Element {
	c.Focus().Add(r)
	return r.El
}
