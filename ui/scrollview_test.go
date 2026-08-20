package ui

import (
	"testing"

	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/layout"
)

func TestScrollViewContentHeight(t *testing.T) {
	content := layout.New(layout.Box().Direction(layout.Column).Gap(8))
	for i := 0; i < 10; i++ {
		content.Children = append(content.Children, layout.New(layout.Box().H(40)))
	}
	sv := NewScrollView(content)
	root := layout.New(layout.Box().FlexGrow(1), sv.host)
	root.Calculate(400, 200)

	if content.Frame.H < 470 {
		t.Fatalf("scroll content height: got %v want >= 470", content.Frame.H)
	}
	if content.Children[0].Frame.H < 39 {
		t.Fatalf("block height shrunk: got %v", content.Children[0].Frame.H)
	}
}

func TestScrollDSLDoesNotShrinkChildren(t *testing.T) {
	c := New(nil, NewFocusScope(), nil)
	blocks := make([]View, 0, 10)
	for i := 0; i < 10; i++ {
		blocks = append(blocks, Raw(layout.New(layout.Box().H(40).FlexShrink(0))))
	}
	root := BuildFrame(c, func(_ *Ctx) View {
		return Scroll("s", Column(blocks...).Gap(8)).Grow(1)
	}, 400, 200, nil, nil)

	var col *layout.Element
	var find func(*layout.Element)
	find = func(e *layout.Element) {
		if e == nil || col != nil {
			return
		}
		if len(e.Children) == 10 {
			col = e
			return
		}
		for _, ch := range e.Children {
			find(ch)
		}
	}
	find(root)
	if col == nil {
		t.Fatal("scroll content column not found")
	}
	if col.Frame.H < 470 {
		t.Fatalf("content height: got %v want >= 470", col.Frame.H)
	}
	for i := 1; i < len(col.Children); i++ {
		prev := col.Children[i-1]
		ch := col.Children[i]
		if ch.Frame.Y < prev.Frame.Y+prev.Frame.H-0.5 {
			t.Fatalf("children overlap at %d: prev=%v child=%v", i, prev.Frame, ch.Frame)
		}
	}
}

func TestScrollDSLWheel(t *testing.T) {
	c := New(nil, NewFocusScope(), nil)
	body := func(_ *Ctx) View {
		blocks := make([]View, 0, 10)
		for i := 0; i < 10; i++ {
			blocks = append(blocks, Raw(layout.New(layout.Box().H(40).FlexShrink(0))))
		}
		return Scroll("s", Column(blocks...).Gap(8)).Grow(1)
	}
	mouse := &input.Mouse{X: 40, Y: 40}
	root := BuildFrame(c, body, 400, 200, mouse, nil)
	layout.Dispatch(root, mouse)

	mouse.ScrollY = -2
	layout.Dispatch(root, mouse)

	sv := c.Widget("s", func() any { return NewScrollView(nil) }).(*ScrollView)
	if sv.scrollY <= 0 {
		t.Fatalf("wheel did not scroll: offset=%v contentH=%v host=%v", sv.scrollY, sv.contentH, sv.host.Frame)
	}

	mouse.ScrollY = 0
	root = BuildFrame(c, body, 400, 200, mouse, nil)
	sv = c.Widget("s", func() any { return NewScrollView(nil) }).(*ScrollView)
	if sv.viewEl.ScrollOffset <= 0 {
		t.Fatalf("paint pass reset scroll: offset=%v contentH=%v", sv.viewEl.ScrollOffset, sv.contentH)
	}
}

func TestScrollClampsOnFirstFrameAfterShorterContent(t *testing.T) {
	c := New(nil, NewFocusScope(), nil)
	tallBody := func(_ *Ctx) View {
		blocks := make([]View, 0, 20)
		for i := 0; i < 20; i++ {
			blocks = append(blocks, Raw(layout.New(layout.Box().H(40).FlexShrink(0))))
		}
		return Scroll("page", Column(blocks...).Gap(4)).Grow(1)
	}
	shortBody := func(_ *Ctx) View {
		return Scroll("page", Column(
			Raw(layout.New(layout.Box().H(40).FlexShrink(0))),
			Raw(layout.New(layout.Box().H(40).FlexShrink(0))),
		).Gap(4)).Grow(1)
	}

	mouse := &input.Mouse{X: 40, Y: 40}
	root := BuildFrame(c, tallBody, 400, 200, mouse, nil)
	layout.Dispatch(root, mouse)
	mouse.ScrollY = -40
	layout.Dispatch(root, mouse)

	sv := c.Widget("page", func() any { return NewScrollView(nil) }).(*ScrollView)
	if sv.scrollY <= 0 {
		t.Fatalf("setup: expected scrolled tall page, offset=%v contentH=%v", sv.scrollY, sv.contentH)
	}
	tallOffset := sv.scrollY

	// Single BuildFrame with shorter content must clamp immediately — idle
	// WaitEvents would otherwise leave the stale offset until the next click.
	root = BuildFrame(c, shortBody, 400, 200, &input.Mouse{X: 40, Y: 40}, nil)
	_ = root
	sv = c.Widget("page", func() any { return NewScrollView(nil) }).(*ScrollView)
	if sv.scrollY >= tallOffset {
		t.Fatalf("scroll not clamped on first short frame: offset=%v (was %v) contentH=%v", sv.scrollY, tallOffset, sv.contentH)
	}
	if sv.scrollY > 0.5 {
		t.Fatalf("short content should clamp near 0: offset=%v contentH=%v hostH=%v", sv.scrollY, sv.contentH, sv.host.Frame.H)
	}
	if sv.contentH > 200 {
		t.Fatalf("contentH still tall after short page: %v", sv.contentH)
	}
	if sv.viewEl.ScrollOffset > 0.5 {
		t.Fatalf("ScrollOffset not updated: %v", sv.viewEl.ScrollOffset)
	}
}

func TestScrollGrowDoesNotInflateChrome(t *testing.T) {
	c := New(nil, NewFocusScope(), nil)
	blocks := make([]View, 0, 20)
	for i := 0; i < 20; i++ {
		blocks = append(blocks, Raw(layout.New(layout.Box().H(40).FlexShrink(0))))
	}
	root := BuildFrame(c, func(_ *Ctx) View {
		return Column(
			Raw(layout.New(layout.Box().H(40).FlexShrink(0))), // chrome
			Scroll("tall", Column(blocks...).Gap(4)).Grow(1),
		).Grow(1)
	}, 400, 200, nil, nil)

	if len(root.Children) < 2 {
		t.Fatalf("expected chrome+scroll, children=%d", len(root.Children))
	}
	chrome := root.Children[0]
	if chrome.Frame.H < 39 {
		t.Fatalf("tall scroll content shrunk chrome: h=%v", chrome.Frame.H)
	}
	if chrome.Frame.Y != 0 {
		t.Fatalf("chrome y: got %v want 0", chrome.Frame.Y)
	}
}
