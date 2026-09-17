//go:build !js

package highlight

import (
	"fmt"
	"strings"
	"testing"
	"time"

	tree_sitter_xml "github.com/tree-sitter-grammars/tree-sitter-xml/bindings/go"
	tree_sitter "github.com/tree-sitter/go-tree-sitter"
	tree_sitter_go "github.com/tree-sitter/tree-sitter-go/bindings/go"
	tree_sitter_json "github.com/tree-sitter/tree-sitter-json/bindings/go"
	"unsafe"
)

func parseWith(t testing.TB, langFn func() unsafe.Pointer, src []byte) *tree_sitter.Tree {
	t.Helper()
	p := tree_sitter.NewParser()
	t.Cleanup(p.Close)
	if err := p.SetLanguage(tree_sitter.NewLanguage(langFn())); err != nil {
		t.Fatal(err)
	}
	tree := p.Parse(src, nil)
	if tree == nil {
		t.Fatal("parse returned nil")
	}
	t.Cleanup(tree.Close)
	return tree
}

func classifyWith(fn classifyFunc, tree *tree_sitter.Tree, lo, hi int) []Token {
	w := newRangeWalker(lo, hi)
	fn(w, tree.RootNode())
	return w.toks
}

// filterToRange keeps the tokens of a whole-document classification that
// intersect [lo, hi), which is what a range-scoped walk must reproduce.
func filterToRange(toks []Token, lo, hi int) []Token {
	var out []Token
	for _, tk := range toks {
		if tk.End > lo && tk.Start < hi {
			out = append(out, tk)
		}
	}
	return out
}

func sameTokens(a, b []Token) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// TestRangeClassifyMatchesWholeDocument is the correctness contract: asking for
// a byte range must give exactly the tokens a whole-document walk would give
// for that range, at every scroll position and for every language.
func TestRangeClassifyMatchesWholeDocument(t *testing.T) {
	cases := []struct {
		name     string
		langFn   func() unsafe.Pointer
		classify classifyFunc
		src      []byte
	}{
		{"json", tree_sitter_json.Language, classifyJSON, []byte(jsonDoc())},
		{"go", tree_sitter_go.Language, classifyGo, []byte(goDoc())},
		{"xml", tree_sitter_xml.LanguageXML, classifyXML, []byte(xmlDoc())},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tree := parseWith(t, tc.langFn, tc.src)
			whole := classifyWith(tc.classify, tree, 0, maxInt)
			if len(whole) == 0 {
				t.Fatal("whole-document classify produced no tokens")
			}

			n := len(tc.src)
			// Windows across the document, plus degenerate ones.
			windows := [][2]int{
				{0, 0}, {0, 1}, {0, 200},
				{n / 4, n/4 + 500},
				{n / 2, n/2 + 500},
				{n - 500, n},
				{n - 1, n}, {n, n}, {n, n + 1000},
				{0, n},
			}
			for _, wnd := range windows {
				lo, hi := wnd[0], wnd[1]
				got := classifyWith(tc.classify, tree, lo, hi)
				want := filterToRange(whole, lo, hi)
				if !sameTokens(got, want) {
					t.Errorf("range [%d,%d): got %d tokens, want %d\ngot:  %v\nwant: %v",
						lo, hi, len(got), len(want), head(got), head(want))
				}
			}
		})
	}
}

// TestRangeClassifySlidingWindow sweeps a viewport-sized window across the
// whole document, so every boundary is exercised.
func TestRangeClassifySlidingWindow(t *testing.T) {
	src := []byte(jsonDoc())
	tree := parseWith(t, tree_sitter_json.Language, src)
	whole := classifyWith(classifyJSON, tree, 0, maxInt)

	const window = 300
	for lo := 0; lo < len(src); lo += 97 { // stride co-prime with the content
		hi := lo + window
		got := classifyWith(classifyJSON, tree, lo, hi)
		want := filterToRange(whole, lo, hi)
		if !sameTokens(got, want) {
			t.Fatalf("window [%d,%d): got %d want %d", lo, hi, len(got), len(want))
		}
	}
}

// TestSetRangeReclassifiesWithoutReparse covers the scroll path: a new range
// must produce a new result from the tree already in hand.
func TestSetRangeReclassifiesWithoutReparse(t *testing.T) {
	h := NewJSON().(*tsHighlighter)
	defer h.Close()

	src := []byte(jsonDoc())
	h.SetRange(0, 400)
	h.Update(src)

	first, ok := pollFor(h, 2*time.Second)
	if !ok {
		t.Fatal("no initial tokens")
	}
	for _, tk := range first {
		if tk.Start >= 400 {
			t.Fatalf("token at %d is outside the requested range [0,400)", tk.Start)
		}
	}

	// Scroll: same document, different range, no Update call.
	lo, hi := len(src)/2, len(src)/2+400
	h.SetRange(lo, hi)
	second, ok := pollFor(h, 2*time.Second)
	if !ok {
		t.Fatal("no tokens after SetRange")
	}
	if len(second) == 0 {
		t.Fatal("scrolled range produced no tokens")
	}
	for _, tk := range second {
		if tk.End <= lo || tk.Start >= hi {
			t.Errorf("token [%d,%d) outside requested range [%d,%d)", tk.Start, tk.End, lo, hi)
		}
	}
}

// TestSetRangeIdempotent guards against a request-per-frame feedback loop:
// repeating the current range must not produce more results.
func TestSetRangeIdempotent(t *testing.T) {
	h := NewJSON().(*tsHighlighter)
	defer h.Close()

	h.SetRange(0, 400)
	h.Update([]byte(jsonDoc()))
	if _, ok := pollFor(h, 2*time.Second); !ok {
		t.Fatal("no initial tokens")
	}

	for i := 0; i < 20; i++ {
		h.SetRange(0, 400)
	}
	time.Sleep(150 * time.Millisecond)
	if toks, ok := h.Poll(); ok {
		t.Errorf("unchanged range delivered another result (%d tokens)", len(toks))
	}
}

// TestNoRangeStillClassifiesWholeDocument keeps the old behaviour for consumers
// that never ask for a range.
func TestNoRangeStillClassifiesWholeDocument(t *testing.T) {
	h := NewJSON()
	defer h.Close()

	src := []byte(jsonDoc())
	h.Update(src)
	toks, ok := pollFor(h, 2*time.Second)
	if !ok {
		t.Fatal("no tokens")
	}
	last := toks[len(toks)-1]
	if last.End < len(src)/2 {
		t.Errorf("tokens stop at %d for a %d byte document; expected whole-document coverage",
			last.End, len(src))
	}
}

func head(toks []Token) []Token {
	if len(toks) > 6 {
		return toks[:6]
	}
	return toks
}

func jsonDoc() string {
	var b strings.Builder
	b.WriteString("{\n  \"items\": [\n")
	for i := 0; i < 400; i++ {
		fmt.Fprintf(&b, "    {\"id\": %d, \"name\": \"item-%03d\", \"ok\": %v, \"tag\": null},\n",
			i, i, i%2 == 0)
	}
	b.WriteString("    {}\n  ]\n}\n")
	return b.String()
}

func goDoc() string {
	var b strings.Builder
	b.WriteString("package main\n\nimport \"fmt\"\n\n")
	for i := 0; i < 120; i++ {
		fmt.Fprintf(&b, `
// Handler%d is a comment.
func Handler%d(id int, name string) (string, error) {
	if id <= 0 {
		return "", fmt.Errorf("bad %%d", id)
	}
	const x = 3.14
	return fmt.Sprintf("%%s", name), nil
}
`, i, i)
	}
	return b.String()
}

func xmlDoc() string {
	var b strings.Builder
	b.WriteString("<?xml version=\"1.0\"?>\n<root attr=\"top\">\n")
	for i := 0; i < 200; i++ {
		fmt.Fprintf(&b, "  <item id=\"%d\" name=\"item-%03d\"><!-- note --><value>%d</value></item>\n", i, i, i)
	}
	b.WriteString("</root>\n")
	return b.String()
}
