// Package highlight maps source text to colored token ranges using Tree-sitter.
//
// Parsing runs on a dedicated goroutine (the "worker loop") so the UI thread is
// never blocked by a parse. The editor pushes the latest source via Update and
// polls for finished results via Poll; both are non-blocking. Results cross the
// goroutine boundary as a plain Go slice of byte ranges, so the UI never touches
// a live Tree-sitter tree.
//
// Cgo lifecycle: the Tree-sitter Parser and every Tree allocate C memory that
// the Go GC does not track. Per the binding's documentation, SetFinalizer is
// unreliable here, so the worker owns these objects and calls Close()
// deterministically — the previous tree is closed once its successor is parsed,
// and the parser/tree are closed when the loop exits on Close().
package highlight

import (
	"path/filepath"
	"strings"
	"unsafe"

	tree_sitter "github.com/tree-sitter/go-tree-sitter"
	tree_sitter_go "github.com/tree-sitter/tree-sitter-go/bindings/go"
	tree_sitter_json "github.com/tree-sitter/tree-sitter-json/bindings/go"
)

// ColorClass is a semantic token category the renderer maps to a theme color.
type ColorClass uint8

const (
	ClassDefault ColorClass = iota
	ClassKeyword
	ClassString
	ClassComment
	ClassNumber
	ClassType
)

// Token is a half-open byte range [Start, End) with a color class.
type Token struct {
	Start, End int
	Class      ColorClass
}

// Highlighter is the async syntax-highlighting interface the editor depends on.
// Swapping in a different engine (or the Noop highlighter) only requires
// implementing these three methods.
type Highlighter interface {
	// Update requests a (re)parse of source. It is non-blocking; if a parse is
	// already queued it is replaced with the newest source (coalescing).
	Update(source []byte)
	// Poll returns the most recent finished token set, or ok=false if nothing
	// new has completed since the last call.
	Poll() (tokens []Token, ok bool)
	// Close stops the worker and frees its native resources.
	Close()
}

// ForPath returns a highlighter appropriate for a file path, chosen by its
// extension. This is the single place to register new languages: add a grammar
// binding and a classifier, then map the extension here. Unknown types fall
// back to plain text (Noop).
func ForPath(path string) Highlighter {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".go":
		return NewGo()
	case ".json":
		return NewJSON()
	default:
		return Noop{}
	}
}

// classifyFunc walks a parsed syntax tree and emits a flat, ordered list of
// colored token ranges. Each language supplies one.
type classifyFunc func(root *tree_sitter.Node, src []byte) []Token

// tsHighlighter is a generic Tree-sitter-backed highlighter. It is parameterized
// by a grammar (langFn returns the C language pointer) and a classifier, so a
// new language is just those two values — the async worker loop, result
// coalescing, and Cgo tree lifecycle are shared.
type tsHighlighter struct {
	jobs     chan []byte
	results  chan []Token
	done     chan struct{}
	langFn   func() unsafe.Pointer
	classify classifyFunc
}

// newTS starts a worker loop for the given grammar/classifier and returns it.
func newTS(langFn func() unsafe.Pointer, classify classifyFunc) Highlighter {
	h := &tsHighlighter{
		jobs:     make(chan []byte, 1),
		results:  make(chan []Token, 1),
		done:     make(chan struct{}),
		langFn:   langFn,
		classify: classify,
	}
	go h.loop()
	return h
}

// NewGo starts a worker loop highlighting Go source and returns its handle.
func NewGo() Highlighter { return newTS(tree_sitter_go.Language, classifyGo) }

// NewJSON starts a worker loop highlighting JSON and returns its handle.
func NewJSON() Highlighter { return newTS(tree_sitter_json.Language, classifyJSON) }

func (h *tsHighlighter) loop() {
	parser := tree_sitter.NewParser()
	defer parser.Close()

	lang := tree_sitter.NewLanguage(h.langFn())
	if err := parser.SetLanguage(lang); err != nil {
		return
	}

	var prev *tree_sitter.Tree
	defer func() {
		if prev != nil {
			prev.Close()
		}
	}()

	for {
		select {
		case <-h.done:
			return
		case src := <-h.jobs:
			// Full reparse for clarity. Incremental reparsing would call
			// prev.Edit(InputEdit) before Parse(src, prev); the worker structure
			// is identical either way.
			tree := parser.Parse(src, nil)
			if tree == nil {
				continue
			}
			if prev != nil {
				prev.Close()
			}
			prev = tree

			toks := h.classify(tree.RootNode(), src)
			deliver(h.results, toks)
		}
	}
}

// deliver pushes the latest result, discarding any unconsumed older one so the
// editor always sees the freshest tokens.
func deliver(ch chan []Token, toks []Token) {
	select {
	case ch <- toks:
	default:
		select {
		case <-ch:
		default:
		}
		select {
		case ch <- toks:
		default:
		}
	}
}

func (h *tsHighlighter) Update(source []byte) {
	// Copy: the worker reads this on another goroutine while the editor may keep
	// mutating its own buffer.
	cp := append([]byte(nil), source...)
	select {
	case h.jobs <- cp:
	default:
		select {
		case <-h.jobs:
		default:
		}
		select {
		case h.jobs <- cp:
		default:
		}
	}
}

func (h *tsHighlighter) Poll() ([]Token, bool) {
	select {
	case toks := <-h.results:
		return toks, true
	default:
		return nil, false
	}
}

func (h *tsHighlighter) Close() { close(h.done) }

// goKeywords is the set of Go keywords; Tree-sitter emits these as anonymous
// leaf nodes whose Kind() is the literal keyword text.
var goKeywords = map[string]bool{
	"break": true, "case": true, "chan": true, "const": true, "continue": true,
	"default": true, "defer": true, "else": true, "fallthrough": true, "for": true,
	"func": true, "go": true, "goto": true, "if": true, "import": true,
	"interface": true, "map": true, "package": true, "range": true, "return": true,
	"select": true, "struct": true, "switch": true, "type": true, "var": true,
}

// classifyGo walks a Go syntax tree and emits a flat, ordered list of colored
// token ranges. Container nodes recurse; leaf-ish nodes are classified directly.
func classifyGo(root *tree_sitter.Node, src []byte) []Token {
	var toks []Token
	add := func(n *tree_sitter.Node, c ColorClass) {
		toks = append(toks, Token{Start: int(n.StartByte()), End: int(n.EndByte()), Class: c})
	}

	var walk func(n *tree_sitter.Node)
	walk = func(n *tree_sitter.Node) {
		switch n.Kind() {
		case "comment":
			add(n, ClassComment)
			return
		case "interpreted_string_literal", "raw_string_literal", "rune_literal":
			add(n, ClassString)
			return
		case "int_literal", "float_literal", "imaginary_literal":
			add(n, ClassNumber)
			return
		case "type_identifier":
			add(n, ClassType)
			return
		}

		count := n.ChildCount()
		if count == 0 {
			if !n.IsNamed() && goKeywords[n.Kind()] {
				add(n, ClassKeyword)
			}
			return
		}
		for i := uint(0); i < count; i++ {
			if child := n.Child(i); child != nil {
				walk(child)
			}
		}
	}
	walk(root)
	return toks
}

// classifyJSON walks a JSON syntax tree. Object keys are colored distinctly
// (ClassType) from string values (ClassString); numbers, the literals
// true/false/null, and comments (JSONC) get their own classes.
func classifyJSON(root *tree_sitter.Node, src []byte) []Token {
	var toks []Token
	add := func(n *tree_sitter.Node, c ColorClass) {
		toks = append(toks, Token{Start: int(n.StartByte()), End: int(n.EndByte()), Class: c})
	}

	var walk func(n *tree_sitter.Node)
	walk = func(n *tree_sitter.Node) {
		switch n.Kind() {
		case "comment":
			add(n, ClassComment)
			return
		case "number":
			add(n, ClassNumber)
			return
		case "true", "false", "null":
			add(n, ClassKeyword)
			return
		case "string":
			add(n, ClassString)
			return
		case "pair":
			// A key/value member: color the key like a property name and recurse
			// only into the value, so the key string is not re-colored generically.
			if k := n.ChildByFieldName("key"); k != nil {
				add(k, ClassType)
			}
			if v := n.ChildByFieldName("value"); v != nil {
				walk(v)
			}
			return
		}

		count := n.ChildCount()
		for i := uint(0); i < count; i++ {
			if child := n.Child(i); child != nil {
				walk(child)
			}
		}
	}
	walk(root)
	return toks
}

// Noop is a highlighter that produces no tokens; the editor falls back to the
// default text color. Useful for tests, non-code text, or when Tree-sitter is
// undesirable.
type Noop struct{}

func (Noop) Update([]byte)         {}
func (Noop) Poll() ([]Token, bool) { return nil, false }
func (Noop) Close()                {}
