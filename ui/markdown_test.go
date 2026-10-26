package ui

import (
	"os"
	"strings"
	"testing"

	"github.com/mirzakhany/yoga/highlight"
	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
)

const mdSample = "# Title\n\nSome **bold**, *em*, ~~gone~~ and `code` with a [link](https://x.dev).\n\n" +
	"- one\n- two\n  1. nested\n- [x] done\n\n> quoted\n\n```go\nfunc main() {}\n```\n\n" +
	"| a | b |\n|---|--:|\n| 1 | 2 |\n\n---\n"

func TestParseMarkdownBlocks(t *testing.T) {
	bs := parseMarkdown(mdSample)
	var kinds []mdKind
	for _, b := range bs {
		kinds = append(kinds, b.kind)
	}
	want := []mdKind{mdHeading, mdParagraph, mdList, mdQuote, mdCode, mdTable, mdRule}
	if len(kinds) != len(want) {
		t.Fatalf("kinds %v, want %v", kinds, want)
	}
	for i := range want {
		if kinds[i] != want[i] {
			t.Fatalf("kinds %v, want %v", kinds, want)
		}
	}
	p := bs[1].text
	if p.text != "Some bold, em, gone and code with a link." {
		t.Fatalf("paragraph text %q", p.text)
	}
	styles := map[string]mdStyle{}
	for _, r := range p.runs {
		styles[p.text[r.lo:r.hi]] = r.style
	}
	if !styles["bold"].bold || !styles["em"].italic || !styles["gone"].strike || !styles["code"].code ||
		styles["link"].link != "https://x.dev" {
		t.Fatalf("styles %+v", styles)
	}
	list := bs[2]
	if len(list.children) != 3 || list.children[2].task != 2 || list.children[2].children[0].text.text != "done" {
		t.Fatalf("list %+v", list.children)
	}
	if nested := list.children[1].children; len(nested) != 2 || nested[1].kind != mdList || !nested[1].ordered {
		t.Fatalf("nested list missing: %+v", nested)
	}
	if bs[4].lang != "go" || bs[4].code != "func main() {}" {
		t.Fatalf("code %q %q", bs[4].lang, bs[4].code)
	}
	if len(bs[5].rows) != 2 || bs[5].rows[1][1].text != "2" {
		t.Fatalf("table rows %+v", bs[5].rows)
	}
}

// An unclosed fence, as in a reply still streaming, is a code block.
func TestParseMarkdownUnclosedFence(t *testing.T) {
	bs := parseMarkdown("Here:\n\n```py\nprint(1)\n")
	if len(bs) != 2 || bs[1].kind != mdCode || bs[1].code != "print(1)" {
		t.Fatalf("got %+v", bs)
	}
}

func layoutMarkdown(c *Ctx, src string, w float32) *layout.Element {
	c.BeginFrame(w, 2000, nil, nil)
	root := Column(Markdown("md", src)).Layout(c)
	root.Calculate(w, 2000)
	return root
}

func TestMarkdownLaysOutAndWraps(t *testing.T) {
	c, text := newParagraphCtx(t)
	src := strings.Repeat("word ", 60)
	wide := layoutMarkdown(c, src, 2000).Children[0].Frame.H
	narrow := layoutMarkdown(c, src, 200).Children[0].Frame.H
	if wide <= 0 || narrow <= wide*2 {
		t.Fatalf("wide %v narrow %v: narrow text should wrap to more lines", wide, narrow)
	}
	// Every line fits the width.
	st := c.peekWidget("md.0").(*richState)
	for _, f := range st.frags {
		if f.x+f.w > 200+0.5 {
			t.Fatalf("fragment %q overflows: x %v w %v", st.text[f.lo:f.hi], f.x, f.w)
		}
	}
	_ = text

	// The whole sample lays out and paints without trouble.
	root := layoutMarkdown(c, mdSample, 500)
	var dl render.DrawList
	layout.Paint(root, &dl, text)
	c.EndFrame()
}

func TestRichWrapKeepsWordsAcrossStyles(t *testing.T) {
	c, text := newParagraphCtx(t)
	_ = c
	rt := &richText{}
	rt.add("aaa ", mdStyle{})
	rt.add("bb", mdStyle{bold: true})
	rt.add("cc", mdStyle{})
	v := &richView{rt: rt, size: 14, weight: 400}
	st := &richState{}
	w1, _ := text.MeasureAt("aaa ", 14)
	// Room for "aaa " and "bb" but not "bbcc": the word moves down whole.
	bb := v.measure(text, "bb", mdStyle{bold: true})
	v.wrap(text, st, w1+bb+1)
	if st.lines != 2 {
		t.Fatalf("lines %d, frags %+v", st.lines, st.frags)
	}
	for _, f := range st.frags {
		if rt.text[f.lo:f.hi] == "bb" && f.line != 1 {
			t.Fatalf("bb split from cc: %+v", st.frags)
		}
	}
}

func TestRichOffsetAt(t *testing.T) {
	_, text := newParagraphCtx(t)
	rt := &richText{}
	rt.add("hello world", mdStyle{})
	v := &richView{rt: rt, size: 14, weight: 400}
	st := &richState{}
	v.wrap(text, st, 1000)
	if got := v.offsetAt(text, st, -5, 1, 20); got != 0 {
		t.Fatalf("before start: %d", got)
	}
	if got := v.offsetAt(text, st, 900, 1, 20); got != len(rt.text) {
		t.Fatalf("past end: %d", got)
	}
	w, _ := text.MeasureAt("hello ", 14)
	if got := v.offsetAt(text, st, w+0.5, 1, 20); got != 6 {
		t.Fatalf("at 'w': %d", got)
	}
}

func TestRichSegmentsSplitAtTokens(t *testing.T) {
	rt := &richText{}
	rt.add("func main", mdStyle{})
	v := &richView{rt: rt, tokens: []highlight.Token{{Start: 0, End: 4, Class: highlight.ClassKeyword}}}
	segs := v.segments()
	if len(segs) != 2 || segs[0].class != highlight.ClassKeyword || segs[1].lo != 4 {
		t.Fatalf("segs %+v", segs)
	}
}

// A link click calls OnLink with the destination.
func TestMarkdownLinkClick(t *testing.T) {
	c, text := newParagraphCtx(t)
	var got string
	frame := func(m *input.Mouse) *layout.Element {
		c.BeginFrame(400, 400, m, nil)
		root := Column(Markdown("md", "[go](https://go.dev)").OnLink(func(u string) { got = u })).Layout(c)
		root.Calculate(400, 400)
		if m != nil {
			layout.Dispatch(root, m)
		}
		var dl render.DrawList
		layout.Paint(root, &dl, text)
		c.EndFrame()
		return root
	}
	frame(nil)
	frame(&input.Mouse{X: 5, Y: 10, Pressed: true, Down: true})
	frame(&input.Mouse{X: 5, Y: 10, Released: true})
	if got != "https://go.dev" {
		t.Fatalf("OnLink got %q", got)
	}
}

// Code block highlighters stop when the view leaves the tree.
func TestMarkdownReleasesHighlighters(t *testing.T) {
	c, _ := newParagraphCtx(t)
	layoutMarkdown(c, "```go\nx := 1\n```", 400)
	c.EndFrame()
	st := c.peekWidget("md").(*markdownState)
	if len(st.code) != 1 {
		t.Fatalf("code states %d", len(st.code))
	}
	c.BeginFrame(400, 400, nil, nil)
	Column().Layout(c)
	c.EndFrame()
	if len(st.code) != 0 {
		t.Fatalf("highlighter not released")
	}
}

// A steady frame of a long document reuses the parse and the wraps.
func BenchmarkMarkdownSteadyFrame(b *testing.B) {
	src, err := os.ReadFile("../README.md")
	if err != nil {
		b.Skip(err)
	}
	t := &testing.T{}
	c, _ := newParagraphCtx(t)
	layoutMarkdown(c, string(src), 800)
	c.EndFrame()
	b.ReportAllocs()
	for b.Loop() {
		layoutMarkdown(c, string(src), 800)
		c.EndFrame()
	}
}
