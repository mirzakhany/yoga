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
		{"toml", NewTOML, "# c\n[server]\nname = \"hm\"\nport = 8080\ndebug = true\n",
			map[string]ColorClass{"# c": ClassComment, "server": ClassType, "name": ClassProperty, "\"hm\"": ClassString,
				"8080": ClassNumber, "true": ClassConstant, "=": ClassOperator}},
		{"c", NewC, "#include <stdio.h>\nstruct point { int x; };\nint main(void) { printf(\"%d\", 42); return 0; }\n",
			map[string]ColorClass{"#include": ClassKeyword, "struct": ClassKeyword, "point": ClassType, "x": ClassProperty,
				"main": ClassFunction, "printf": ClassFunction, "42": ClassNumber, "return": ClassKeyword}},
		{"cpp", NewCPP, "namespace app {\nclass Shape { public: virtual double area() const; };\n}\nauto s = std::make_unique<Shape>();\n",
			map[string]ColorClass{"namespace": ClassKeyword, "class": ClassKeyword, "Shape": ClassType, "area": ClassFunction,
				"virtual": ClassKeyword, "make_unique": ClassFunction}},
		{"csharp", NewCSharp, "namespace App;\npublic class User { public string Name { get; set; } void Run() { Console.WriteLine(1); } }\n",
			map[string]ColorClass{"class": ClassKeyword, "User": ClassType, "Name": ClassProperty, "Run": ClassFunction,
				"WriteLine": ClassFunction, "1": ClassNumber, "public": ClassKeyword}},
		{"lua", NewLua, "-- c\nlocal function greet(who)\n  print(\"hi\")\nend\nlocal t = { size = 3 }\n",
			map[string]ColorClass{"-- c": ClassComment, "local": ClassKeyword, "greet": ClassFunction, "who": ClassParameter,
				"print": ClassBuiltin, "3": ClassNumber, "end": ClassKeyword}},
		{"ruby", NewRuby, "# c\nclass User < Base\n  def greet(who)\n    puts \"hi\"\n  end\nend\nx = :sym\n",
			map[string]ColorClass{"# c": ClassComment, "class": ClassKeyword, "User": ClassType, "greet": ClassMethod,
				"who": ClassParameter, "def": ClassKeyword, ":sym": ClassString, "\"hi\"": ClassString}},
		{"php", NewPHP, "<?php\nclass User { public function greet(string $name): string { return \"hi\"; } }\n$u = new User();\n",
			map[string]ColorClass{"class": ClassKeyword, "User": ClassType, "greet": ClassMethod, "function": ClassKeyword,
				"\"hi\"": ClassString, "return": ClassKeyword}},
		{"zig", NewZig, "const std = @import(\"std\");\npub fn main() void {\n    const MAX = @as(u8, 10);\n    std.debug.print(\"{d}\\n\", .{MAX});\n}\n",
			map[string]ColorClass{"const": ClassKeyword, "@import": ClassKeyword, "@as": ClassBuiltin, "MAX": ClassConstant, "main": ClassFunction, "fn": ClassKeyword,
				"10": ClassNumber, "void": ClassType, "print": ClassFunction}},
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

func TestNewLanguagesByPath(t *testing.T) {
	for _, p := range []string{"/src/Cargo.toml", "main.c", "util.hpp", "Program.cs", "init.lua",
		"Gemfile", "app.rb", "index.php", "build.zig", "/repo/.gitignore", ".dockerignore",
		".env", ".env.local", ".editorconfig", "setup.cfg", "Dockerfile", "go.mod", "go.work", "fix.patch"} {
		h := ForPath(p)
		if _, noop := h.(Noop); noop {
			t.Errorf("%s: want a highlighter", p)
		}
		h.Close()
	}
}
