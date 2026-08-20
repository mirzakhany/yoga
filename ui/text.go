package ui

import (
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
	"github.com/mirzakhany/yoga/theme"
)

// Text renders a string. .Size(n) sets the font size in logical pixels.
// Color and size inherit from the parent environment (e.g. a Button's TextColor)
// unless overridden with Style.
func Text(s string) *Node {
	return &Node{kind: kindText, text: s}
}

func (n *Node) layoutText(c *Ctx) *layout.Element {
	th := c.Theme()
	size := c.env.fontSize
	if size <= 0 {
		size = th.Typography.Body.Size
	}
	if n.spec.hasFontSize {
		size = n.spec.fontSize
	}
	col := c.env.textColor
	if !c.env.hasColor {
		col = th.Foreground
	}
	r := n.spec.resolve(th, interactState{})
	if r.hasFg {
		col = r.fg
	}
	if r.hasFontSize {
		size = r.fontSize
	}

	eng := c.Text()
	var tw, lh float32
	if eng != nil {
		tw, lh = eng.MeasureAt(n.text, size)
	} else {
		tw, lh = size*0.5*float32(len(n.text)), size
	}
	// Width/Height are border-box. Include padding so glyphs keep their
	// measured size inside the content box; paint insets by Style.Padding.
	var padL, padR, padT, padB float32
	if n.spec.hasPad {
		padL, padR = n.spec.pad.Left, n.spec.pad.Right
		padT, padB = n.spec.pad.Top, n.spec.pad.Bottom
	}
	st := applyLayoutSpec(layout.Box().Size(tw+padL+padR, lh+padT+padB).FlexShrink(0), n.spec)
	el := layout.New(st)
	content := n.text
	el.Paint = func(dl *render.DrawList, text *shape.Engine) {
		pad := el.Style.Padding
		x := el.Frame.X + pad.Left
		contentH := el.Frame.H - pad.Top - pad.Bottom
		y := el.Frame.Y + pad.Top
		if contentH > lh {
			y += (contentH - lh) / 2
		}
		text.DrawStringTopAt(dl, content, x, y, col, size)
	}
	_ = render.Color{}
	return el
}

// Title is body text using the theme title ramp.
func Title(s string) *Node {
	th := theme.Current()
	return Text(s).Size(th.Typography.Title.Size)
}

// Subtitle is semibold subtitle text.
func Subtitle(s string) *Node {
	th := theme.Current()
	return Text(s).Size(th.Typography.Subtitle.Size)
}

// Caption is small muted text.
func Caption(s string) *Node {
	th := theme.Current()
	return Text(s).Size(th.Typography.Caption.Size).Style(Spec{}.TextColor(TokenForegroundMuted))
}

// Strong is semibold body text.
func Strong(s string) *Node {
	th := theme.Current()
	return Text(s).Size(th.Typography.BodyStrong.Size)
}

// Muted is body text in ForegroundMuted.
func Muted(s string) *Node {
	th := theme.Current()
	return Text(s).Size(th.Typography.Body.Size).Style(Spec{}.TextColor(TokenForegroundMuted))
}
