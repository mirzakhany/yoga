//go:build !js

package highlight

import (
	"strings"
	"testing"
	"time"
)

func TestQueryLanguagesColorNamesByRole(t *testing.T) {
	cases := []struct {
		name string
		h    Factory
		src  string
		want map[string]ColorClass
	}{
		{"go", NewGo, "package p\n\nfunc (s *S) Run(n int) { s.count = len(n); fmt.Println(n) }\n",
			map[string]ColorClass{"Run": ClassMethod, "count": ClassProperty, "len": ClassBuiltin,
				"Println": ClassMethod, "int": ClassType, "func": ClassKeyword}},
		{"java", NewJava, "class A { int n; void run() { this.n = Math.max(1, 2); } }\n",
			map[string]ColorClass{"class": ClassKeyword, "A": ClassType, "run": ClassMethod, "max": ClassMethod, "1": ClassNumber}},
		{"proto", NewProto, "syntax = \"proto3\";\nmessage User { string name = 1; }\nservice S { rpc Get(User) returns (User); }\n",
			map[string]ColorClass{"message": ClassKeyword, "User": ClassType, "name": ClassProperty, "Get": ClassMethod, "1": ClassNumber}},
		{"make", NewMake, "# c\nCC := gcc\nOUT = $(CC)\n.PHONY: all\nall: app\napp: main.o\n\t$(CC) -o $@ $^\nifeq ($(OS),x)\nendif\n",
			map[string]ColorClass{"# c": ClassComment, "CC": ClassConstant, ":=": ClassOperator, "app": ClassFunction,
				".PHONY": ClassConstant, "all": ClassConstant, "ifeq": ClassKeyword, "OS": ClassConstant}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := pollTokens(t, tc.h(), tc.src)
			for text, class := range tc.want {
				if g := got[text]; g != class {
					t.Errorf("%q: class %v, want %v (tokens %v)", text, g, class, got)
				}
			}
		})
	}
}

// A range-scoped request must see the same colors as the whole document,
// predicates included (they read the kept source).
func TestQueryRangeMatchesWholeDocument(t *testing.T) {
	src := strings.Repeat("func f() { x := len(y); g.h() }\n", 200)
	src = "package p\n" + src
	whole := pollTokens(t, NewGo(), src)

	h := NewGo().(RangeHighlighter)
	defer h.Close()
	lo, hi := 2000, 3000
	h.SetRange(lo, hi)
	h.Update([]byte(src))
	var toks []Token
	for range 300 {
		if got, ok := h.Poll(); ok {
			toks = got
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if len(toks) == 0 {
		t.Fatal("no tokens for the range")
	}
	for _, tk := range toks {
		if tk.Start < lo || tk.End > hi {
			t.Fatalf("token %+v outside [%d, %d)", tk, lo, hi)
		}
		text := src[tk.Start:tk.End]
		if want, ok := whole[text]; ok && want != tk.Class && tk.Start > lo && tk.End < hi {
			t.Fatalf("%q: class %v in range, %v in the whole document", text, tk.Class, want)
		}
	}
}

func TestFlattenInnerAndLaterWin(t *testing.T) {
	spans := []span{
		{0, 10, 0, ClassString},   // a string
		{3, 5, 1, ClassConstant},  // an escape inside it
		{3, 5, 0, ClassVariable},  // an earlier pattern on the same node
		{20, 25, 0, ClassComment}, // apart
	}
	got := flatten(spans, 0, 30)
	want := []Token{{0, 3, ClassString}, {3, 5, ClassConstant}, {5, 10, ClassString}, {20, 25, ClassComment}}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}

func TestCaptureClassUsesLongestPrefix(t *testing.T) {
	for name, want := range map[string]ColorClass{
		"function.method.call": ClassMethod, "function.call": ClassFunction,
		"keyword.return": ClassKeyword, "punctuation.bracket": ClassPunctuation,
	} {
		if got, ok := captureClass(name); !ok || got != want {
			t.Errorf("%s: %v %v, want %v", name, got, ok, want)
		}
	}
	if _, ok := captureClass("spell"); ok {
		t.Error("spell should not be colored")
	}
}

func TestQueryFactoryExtendsBuiltin(t *testing.T) {
	q := BuiltinQuery("go") + "\n((identifier) @constant (#match? @constant \"^[A-Z][A-Z0-9_]+$\"))\n"
	f, err := QueryFactory("go", q)
	if err != nil {
		t.Fatal(err)
	}
	got := pollTokens(t, f(), "package p\n\nvar x = MAX_SIZE + y\n")
	if got["MAX_SIZE"] != ClassConstant || got["y"] != ClassVariable {
		t.Fatalf("tokens %v", got)
	}
	if _, err := QueryFactory("go", "(not_a_node) @x"); err == nil {
		t.Fatal("a bad query must be reported")
	}
	if _, err := QueryFactory("cobol", ""); err == nil {
		t.Fatal("an unknown grammar must be reported")
	}
}

func TestMakefileByName(t *testing.T) {
	for _, p := range []string{"/src/Makefile", "GNUmakefile", "rules.mk"} {
		h := ForPath(p)
		if _, noop := h.(Noop); noop {
			t.Errorf("%s: want the make highlighter", p)
		}
		h.Close()
	}
}
