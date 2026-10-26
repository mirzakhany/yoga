package ui

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/mirzakhany/yoga/highlight"
	"github.com/mirzakhany/yoga/icons"
	"github.com/mirzakhany/yoga/layout"
	east "github.com/yuin/goldmark/extension/ast"
)

// markdownView renders Markdown source. See Markdown.
type markdownView struct {
	id     string
	src    string
	onLink func(url string)
}

// markdownState keeps the parse of the last source and the highlighters of
// its code blocks, so a steady frame does no parsing.
type markdownState struct {
	src    string
	parsed bool
	blocks []*mdBlock
	code   map[string]*mdCodeState
	gen    int
}

// mdCodeState highlights one code block. Its key is the block's position,
// so a block that keeps its place while a response streams in keeps its
// highlighter and only reparses.
type mdCodeState struct {
	lang    string
	src     string
	hl      highlight.Highlighter
	tokens  []highlight.Token
	pending bool
	gen     int
}

// release stops the code highlighters when the view leaves the tree.
func (s *markdownState) release() {
	for k, cs := range s.code {
		if cs.hl != nil {
			cs.hl.Close()
		}
		delete(s.code, k)
	}
}

// Markdown renders CommonMark source with the GitHub extensions: headings,
// paragraphs with bold, italic, strikethrough, inline code and links, block
// quotes, ordered, bullet and task lists, fenced code with syntax colors
// and a copy button, tables and rules. Images show as links with their alt
// text.
//
// The source is parsed again only when it changes, so it suits both a file
// preview and a reply that streams in token by token. id keys the parse and
// the selection of each block; give every Markdown in a window its own.
// Text in each block can be selected and copied.
//
// It fills the width it is given and is as tall as its content; put it in a
// Scroll for long documents. Links do nothing until OnLink is set.
func Markdown(id, src string) *Node {
	return &Node{kind: kindWrap, inner: &markdownView{id: id, src: src}}
}

// OnLink sets the handler for a click on a link in a Markdown view, called
// with the link's destination as written.
func (n *Node) OnLink(fn func(url string)) *Node {
	if v, ok := n.inner.(*markdownView); ok {
		v.onLink = fn
	}
	return n
}

func (v *markdownView) Layout(c *Ctx) *layout.Element {
	st := c.Widget(v.id, func() any { return &markdownState{code: map[string]*mdCodeState{}} }).(*markdownState)
	if !st.parsed || st.src != v.src {
		st.src, st.parsed = v.src, true
		st.blocks = parseMarkdown(v.src)
	}
	st.gen++
	r := &mdRender{c: c, st: st, id: v.id, onLink: v.onLink}
	th := c.Theme()
	view := Column(r.blocks(st.blocks, "", false, 0)...).Gap(th.Spacing.M)
	// Drop highlighters of code blocks that are gone.
	for k, cs := range st.code {
		if cs.gen != st.gen {
			if cs.hl != nil {
				cs.hl.Close()
			}
			delete(st.code, k)
		}
	}
	return view.Layout(c)
}

// mdRender turns parsed blocks into views for one frame.
type mdRender struct {
	c      *Ctx
	st     *markdownState
	id     string
	onLink func(string)
}

func (r *mdRender) blocks(bs []*mdBlock, path string, muted bool, depth int) []View {
	out := make([]View, 0, len(bs))
	for i, b := range bs {
		out = append(out, r.block(b, path+"."+strconv.Itoa(i), muted, depth))
	}
	return out
}

func (r *mdRender) text(path string, rt *richText, size float32, weight int, muted bool) View {
	return ViewOf(&richView{id: r.id + path, rt: rt, size: size, weight: weight, muted: muted, onLink: r.onLink})
}

func (r *mdRender) block(b *mdBlock, path string, muted bool, depth int) View {
	th := r.c.Theme()
	body := th.Typography.Body
	switch b.kind {
	case mdHeading:
		size := body.Size
		switch b.level {
		case 1:
			size = th.Typography.Title.Size * 1.4
		case 2:
			size = th.Typography.Title.Size * 1.15
		case 3:
			size = th.Typography.Title.Size
		case 4:
			size = th.Typography.Subtitle.Size
		}
		h := r.text(path, &b.text, size, 600, muted)
		if b.level > 2 {
			return h
		}
		return Column(h, HLine(th.Stroke.Thin, th.Border)).Gap(th.Spacing.XS)
	case mdParagraph:
		return r.text(path, &b.text, body.Size, body.Weight, muted)
	case mdCode:
		return r.code(b, path)
	case mdHTML:
		rt := &richText{text: b.code, runs: []mdRun{{hi: len(b.code)}}}
		return ViewOf(&richView{id: r.id + path, rt: rt, size: body.Size, weight: body.Weight, mono: true, pre: true, muted: true})
	case mdQuote:
		return Row(
			VLine(3, th.BorderStrong).Radius(th.Radius.Small),
			Column(r.blocks(b.children, path, true, depth)...).Gap(th.Spacing.S).Grow(1),
		).Align(AlignStretch).Gap(th.Spacing.M)
	case mdList:
		return r.list(b, path, muted, depth)
	case mdRule:
		return HLine(th.Stroke.Thin, th.Border)
	case mdTable:
		return r.table(b, path, muted)
	}
	return nil
}

func (r *mdRender) list(b *mdBlock, path string, muted bool, depth int) View {
	th := r.c.Theme()
	body := th.Typography.Body
	gap := th.Spacing.XS
	if !b.tight {
		gap = th.Spacing.S
	}
	// Markers share one width so item text lines up.
	markerW := body.Size * 1.2
	if b.ordered {
		last := strconv.Itoa(b.start+len(b.children)-1) + "."
		if eng := r.c.Text(); eng != nil {
			w, _ := eng.MeasureAt(last, body.Size)
			markerW = max(markerW, w)
		}
	}
	fg := th.Foreground
	if muted {
		fg = th.ForegroundMuted
	}
	top := mdLeading(body.Size, false) / 2
	items := make([]View, 0, len(b.children))
	for i, it := range b.children {
		var marker *Node
		switch {
		case it.task != 0:
			icon := icons.Square
			if it.task == 2 {
				icon = icons.SquareCheck
			}
			marker = Icon(icon, body.Size, fg).PaddingTop(top + 2)
		case b.ordered:
			marker = Text(fmt.Sprintf("%d.", b.start+i)).Style(Spec{}.TextColor(TokenForegroundMuted)).PaddingTop(top)
		default:
			bullet := "•"
			if depth%2 == 1 {
				bullet = "◦"
			}
			marker = Text(bullet).Style(Spec{}.TextColor(TokenForegroundMuted)).PaddingTop(top)
		}
		ipath := path + "." + strconv.Itoa(i)
		items = append(items, Row(
			Row(marker).Width(markerW).Justify(JustifyEnd).Align(AlignStart),
			Column(r.blocks(it.children, ipath, muted, depth+1)...).Gap(gap).Grow(1),
		).Align(AlignStart).Gap(th.Spacing.S))
	}
	return Column(items...).Gap(gap)
}

func (r *mdRender) table(b *mdBlock, path string, muted bool) View {
	th := r.c.Theme()
	body := th.Typography.Body
	cols := 0
	for _, row := range b.rows {
		cols = max(cols, len(row))
	}
	if cols == 0 {
		return nil
	}
	var cells []View
	for i, row := range b.rows {
		for j := 0; j < cols; j++ {
			var cell View
			if j < len(row) {
				weight := body.Weight
				if i == 0 {
					weight = 600
				}
				align := AlignStart
				if j < len(b.align) {
					switch b.align[j] {
					case east.AlignRight:
						align = AlignEnd
					case east.AlignCenter:
						align = AlignCenter
					}
				}
				cell = ViewOf(&richView{id: r.id + fmt.Sprintf("%s.%d.%d", path, i, j), rt: &row[j], size: body.Size,
					weight: weight, muted: muted, align: align, onLink: r.onLink})
			}
			n := Column(cell).PaddingXY(th.Spacing.S, th.Spacing.XS).BorderBottom(TokenBorder, th.Stroke.Thin)
			if i == 0 {
				n = n.Background(TokenChromeMuted)
			}
			cells = append(cells, n)
		}
	}
	return Grid(cols, cells...).Border(TokenBorder, th.Stroke.Thin).Radius(th.Radius.Small)
}

// mdLangAliases maps fence names to the highlighter names they mean.
var mdLangAliases = map[string]string{
	"golang": "go", "js": "javascript", "ts": "typescript", "py": "python",
	"sh": "bash", "shell": "bash", "zsh": "bash", "console": "bash",
	"yml": "yaml", "rs": "rust", "jsonc": "json",
}

// mdHighlighter returns a highlighter for a fence language, or nil.
func mdHighlighter(lang string) highlight.Highlighter {
	lang = strings.ToLower(strings.TrimSpace(lang))
	if lang == "" {
		return nil
	}
	if a, ok := mdLangAliases[lang]; ok {
		lang = a
	}
	if f, ok := highlight.ByName(lang); ok {
		return f()
	}
	// An extension an app registered, such as "toml" for ".toml".
	if h := highlight.ForPath("x." + lang); h != nil {
		if _, noop := h.(highlight.Noop); !noop {
			return h
		}
	}
	return nil
}

func (r *mdRender) code(b *mdBlock, path string) View {
	c, st := r.c, r.st
	th := c.Theme()
	cs := st.code[path]
	if cs == nil || cs.lang != b.lang {
		if cs != nil && cs.hl != nil {
			cs.hl.Close()
		}
		cs = &mdCodeState{lang: b.lang, hl: mdHighlighter(b.lang)}
		st.code[path] = cs
	}
	cs.gen = st.gen
	src := strings.ReplaceAll(b.code, "\t", "    ")
	if cs.hl != nil {
		if src != cs.src {
			cs.src = src
			cs.hl.Update([]byte(src))
			cs.pending = true
		}
		if toks, ok := cs.hl.Poll(); ok {
			cs.tokens, cs.pending = toks, false
		}
		if cs.pending {
			c.Animate(30 * time.Millisecond)
		}
	}
	// Tokens from an older source may run past the text while a reply
	// streams in; they are dropped by the segment split.
	toks := cs.tokens
	for len(toks) > 0 && toks[len(toks)-1].End > len(src) {
		toks = toks[:len(toks)-1]
	}

	code := b.code
	label := b.lang
	if label == "" {
		label = "text"
	}
	rt := &richText{text: src, runs: []mdRun{{hi: len(src)}}}
	size := th.Typography.Body.Size - 1
	return Column(
		Row(
			Caption(label).Style(Spec{}.TextColor(TokenForegroundMuted)),
			Spacer(),
			IconButton(r.id+path+"/copy", icons.Copy).Tooltip("Copy").OnClick(func() {
				if clip := c.Clipboard(); clip != nil {
					clip.Set(code)
				}
			}),
		).PaddingLeft(th.Spacing.M).PaddingRight(th.Spacing.XXS).Height(28).
			Background(TokenChromeMuted).RadiusTopLeft(th.Radius.Medium).RadiusTopRight(th.Radius.Medium).BorderBottom(TokenBorder, th.Stroke.Thin),
		Column(ViewOf(&richView{id: r.id + path, rt: rt, size: size, weight: th.Typography.Body.Weight,
			mono: true, pre: true, tokens: toks})).Padding(th.Spacing.M),
	).Background(TokenChrome).Border(TokenBorder, th.Stroke.Thin).Radius(th.Radius.Medium)
}
