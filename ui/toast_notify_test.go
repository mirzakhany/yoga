package ui

import (
	"testing"

	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/shape"
	"github.com/mirzakhany/yoga/theme"
)

func buildNoticeFrame(t *testing.T, c *Ctx) *layout.Element {
	t.Helper()
	return BuildFrame(c, func(_ *Ctx) View { return Text("page") }, 900, 700, nil, nil)
}

func newNoticeCtx(t *testing.T) (*Ctx, *shape.Engine) {
	t.Helper()
	text, err := shape.NewEngine(1, false)
	if err != nil {
		t.Fatal(err)
	}
	SetFrameResources(text, nil, nil)
	return New(text, NewFocusScope(), nil), text
}

// lowest returns the largest bottom edge in the subtree under el.
func lowest(el *layout.Element) float32 {
	b := el.Frame.Y + el.Frame.H
	for _, k := range el.Children {
		b = f32max(b, lowest(k))
	}
	return b
}

func TestNotifyActionClickRunsAndDismisses(t *testing.T) {
	c, text := newNoticeCtx(t)
	installed := false
	c.Toasts().Notify(ToastOpts{
		ID:      "lsp-missing",
		Title:   "Python language server not installed",
		Message: "pyright-langserver provides completion and diagnostics for scripts. Install it now?",
		Variant: ToastWarning,
		Actions: []ToastAction{
			{Label: "Later"},
			{Label: "Install", Primary: true, OnClick: func() { installed = true }},
		},
	})
	root := buildNoticeFrame(t, c)

	n := c.Toasts().notices[0]
	if n.el == nil {
		t.Fatal("notice card was not laid out")
	}
	f := n.el.Frame
	if f.X+f.W > 900 || f.Y < 0 || f.Y+f.H > 700 {
		t.Fatalf("card %+v is off screen", f)
	}
	for _, k := range n.el.Children {
		if b := lowest(k); b > f.Y+f.H+0.5 {
			t.Fatalf("card content reaches %v, below the card bottom %v", b, f.Y+f.H)
		}
	}

	th := theme.Current()
	pad := th.Spacing.M
	tw, _ := text.MeasureAt("Install", th.Typography.Body.Size)
	bw := tw + 2*th.Spacing.M
	layout.Dispatch(root, &input.Mouse{
		X:        f.X + f.W - pad - bw/2,
		Y:        f.Y + f.H - pad - th.Metrics.ControlHeight/2,
		Pressed:  true,
		Released: true,
	})
	if !installed {
		t.Fatal("clicking Install should run its action")
	}
	if c.Toasts().Showing("lsp-missing") {
		t.Fatal("an action click should dismiss the toast")
	}
}

func TestNotifyReplacesByIDAndDismisses(t *testing.T) {
	c, _ := newNoticeCtx(t)
	h := c.Toasts()
	h.Notify(ToastOpts{ID: "a", Title: "one"})
	h.Notify(ToastOpts{ID: "a", Title: "two"})
	h.Notify(ToastOpts{Title: "anonymous"})
	if len(h.notices) != 2 || h.notices[0].opts.Title != "two" {
		t.Fatalf("notices = %+v, want the replaced one plus the anonymous one", h.notices)
	}
	if _, ok := h.AnimationWait(); ok {
		t.Fatal("sticky notices should not schedule an expiry repaint")
	}
	buildNoticeFrame(t, c)
	if a, b := h.notices[0].el.Frame, h.notices[1].el.Frame; a.Y+a.H > b.Y {
		t.Fatalf("stacked cards overlap: %+v above %+v", a, b)
	}
	h.Dismiss("a")
	if h.Showing("a") || len(h.notices) != 1 {
		t.Fatalf("Dismiss left %+v", h.notices)
	}
}
