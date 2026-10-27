//go:build !js

package highlight

import (
	"testing"
	"time"
)

func pollTokens(t *testing.T, h Highlighter, src string) map[string]ColorClass {
	t.Helper()
	defer h.Close()
	h.Update([]byte(src))
	for i := 0; i < 300; i++ {
		if toks, ok := h.Poll(); ok {
			out := map[string]ColorClass{}
			for _, tk := range toks {
				out[src[tk.Start:tk.End]] = tk.Class
			}
			return out
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("no tokens")
	return nil
}

func TestGrammarsColorTheBasics(t *testing.T) {
	cases := []struct {
		name string
		h    Factory
		src  string
		want map[string]ColorClass
	}{
		{"typescript", NewTypeScript, "// hi\nconst n: number = 42\nlet s = `a ${n} b`\n",
			map[string]ColorClass{"// hi": ClassComment, "const": ClassKeyword, "number": ClassType, "42": ClassNumber, "n": ClassVariable}},
		{"tsx", NewTSX, "const el = <Foo bar=\"x\" />\n",
			map[string]ColorClass{"const": ClassKeyword, "Foo": ClassType, "bar": ClassAttribute, `"x"`: ClassString}},
		{"javascript", NewJavaScript, "function f() { return null }\n",
			map[string]ColorClass{"function": ClassKeyword, "f": ClassFunction, "return": ClassKeyword, "null": ClassConstant}},
		{"rust", NewRust, "fn main() { let x: i32 = 1; println!(\"hi\"); }\n",
			map[string]ColorClass{"fn": ClassKeyword, "let": ClassKeyword, "i32": ClassType, "1": ClassNumber, `"hi"`: ClassString, "main": ClassFunction}},
		{"bash", NewBash, "# c\nif [ -n \"$X\" ]; then echo hi; fi\n",
			map[string]ColorClass{"# c": ClassComment, "if": ClassKeyword, "fi": ClassKeyword}},
		{"css", NewCSS, "a.b { color: red; width: 10px }\n",
			map[string]ColorClass{"a": ClassKeyword, "b": ClassType, "color": ClassType}},
		{"html", NewHTML, "<div class=\"x\"><!-- c --></div>\n",
			map[string]ColorClass{"div": ClassKeyword, "class": ClassType, "<!-- c -->": ClassComment}},
		{"yaml", NewYAML, "# c\nname: \"x\"\ncount: 3\non: true\n",
			map[string]ColorClass{"# c": ClassComment, "name": ClassType, `"x"`: ClassString, "3": ClassNumber, "true": ClassKeyword}},
		{"sql", NewSQL, "SELECT id FROM users WHERE name = 'a' -- c\n",
			map[string]ColorClass{"SELECT": ClassKeyword, "FROM": ClassKeyword, "'a'": ClassString, "-- c": ClassComment}},
		{"markdown", NewMarkdown, "# Title\n\nSome `code` and [link](http://x).\n\n- item\n\n```go\nx := 1\n```\n",
			map[string]ColorClass{"# Title": ClassKeyword, "`code`": ClassString, "(http://x)": ClassType, "-": ClassNumber}},
		{"gitignore", NewGitignore, "# deps\nnode_modules/\n*.log\n!keep.log\nfile[0-9].txt\n\\#literal\n",
			map[string]ColorClass{"# deps": ClassComment, "/": ClassPunctuation, "*": ClassOperator, "!": ClassKeyword,
				"[0-9]": ClassOperator, "\\#": ClassConstant, "node_modules": ClassDefault, "keep.log": ClassDefault}},
		{"dotenv", NewDotenv, "# db\nexport DB_HOST=localhost # local\nDB_URL=\"postgres://${DB_HOST}/app\"\nRAW='$NOPE'\n",
			map[string]ColorClass{"# db": ClassComment, "export": ClassKeyword, "DB_HOST": ClassProperty, "=": ClassOperator,
				"localhost ": ClassString, "# local": ClassComment, "${DB_HOST}": ClassVariable, "\"postgres://": ClassString,
				"'$NOPE'": ClassString}},
		{"ini", NewINI, "; c\n[core]\n\teditor = hm\n\tbare = false\n\tindent_size = 4\n\tignorecase\n",
			map[string]ColorClass{"; c": ClassComment, "[core]": ClassType, "editor": ClassProperty, "=": ClassOperator,
				"hm": ClassString, "false": ClassConstant, "4": ClassNumber, "ignorecase": ClassProperty}},
		{"dockerfile", NewDockerfile, "# syntax\nFROM golang:1.24 AS build\nCOPY --from=base . /src\nRUN go build \\\n  -o \"$OUT\"\nENV PATH=${PATH}\n",
			map[string]ColorClass{"# syntax": ClassComment, "FROM": ClassKeyword, "AS": ClassKeyword, "COPY": ClassKeyword,
				"--from": ClassAttribute, "RUN": ClassKeyword, "$OUT": ClassVariable, "${PATH}": ClassVariable, "go": ClassDefault}},
		{"gomod", NewGoMod, "module example.com/app // main\n\ngo 1.24\n\nrequire (\n\tgithub.com/x/y v1.2.3\n)\nreplace a => ../a\n",
			map[string]ColorClass{"module": ClassKeyword, "example.com/app": ClassNamespace, "// main": ClassComment,
				"go": ClassKeyword, "1.24": ClassNumber, "require": ClassKeyword, "(": ClassPunctuation,
				"github.com/x/y": ClassNamespace, "v1.2.3": ClassNumber, "=>": ClassOperator}},
		{"diff", NewDiff, "diff --git a/x b/x\n--- a/x\n+++ b/x\n@@ -1,2 +1,2 @@ func f\n ctx\n-old\n+new\n\\ No newline at end of file\n",
			map[string]ColorClass{"diff --git a/x b/x": ClassMuted, "--- a/x": ClassKeyword, "+++ b/x": ClassKeyword,
				"@@ -1,2 +1,2 @@": ClassType, "-old": ClassError, "+new": ClassSuccess, " ctx": ClassDefault,
				"\\ No newline at end of file": ClassMuted}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := pollTokens(t, tc.h(), tc.src)
			for text, class := range tc.want {
				if g, ok := got[text]; class == ClassDefault && ok || class != ClassDefault && g != class {
					t.Errorf("%q: class %v, want %v (tokens %v)", text, g, class, got)
				}
			}
		})
	}
}

func TestRegisterOverridesBuiltin(t *testing.T) {
	Register(".zz", NewYAML)
	defer Unregister(".zz")
	h := ForPath("a.ZZ")
	defer h.Close()
	if _, noop := h.(Noop); noop {
		t.Fatal("registered extension should get a highlighter")
	}
	if f, ok := ByName("tsx"); !ok || f == nil {
		t.Fatal("ByName(tsx)")
	}
}

func TestRegisterFileName(t *testing.T) {
	Register("Makefile", NewBash)
	defer Unregister("Makefile")
	h := ForPath("/src/makefile")
	defer h.Close()
	if _, noop := h.(Noop); noop {
		t.Fatal("registered file name should get a highlighter")
	}
	other := ForPath("/src/LICENSE")
	defer other.Close()
	if _, noop := other.(Noop); !noop {
		t.Fatal("other files without an extension should stay plain")
	}
}

// Every prefix of these inputs cuts a quote, bracket, flag or continuation
// short; the scanners must not panic on any of them, nor return tokens that
// overlap or run past the source.
func TestFileScannersSurviveTruncation(t *testing.T) {
	scanners := map[string]func([]byte) []Token{
		"gitignore": scanGitignore, "dotenv": layered(scanDotenv), "ini": scanINI,
		"dockerfile": layered(scanDockerfile), "gomod": scanGoMod, "diff": scanDiff,
	}
	inputs := []string{
		"# c\n!a/b\\*[x]?\n[\n\\",
		"export A=\"x ${B} $C\ny\" # t\nD='e' #f\nE=g #h\nF=${\nG=$\n=\n",
		"[s]\nk = v ; c\nb\n[open\nk: 1.5\n=\n",
		"FROM a AS b\nRUN x \\\n  --f=\"$Y ${Z\" '$Q'\n# c\nENV P=${\n--\n$\n\"",
		"module m // c\ngo 1\nrequire (\n\t\"q\" v1\n)\nreplace a => b v2\n`",
		"@@ -1 +1 @@ x\n+a\n-b\n\\ n\n@@\n+++\n---",
	}
	for name, scan := range scanners {
		for _, in := range inputs {
			for n := 0; n <= len(in); n++ {
				src := []byte(in[:n])
				end := 0
				for _, tk := range scan(src) {
					if tk.Start < end || tk.End <= tk.Start || tk.End > len(src) {
						t.Fatalf("%s %q: bad token %+v", name, src, tk)
					}
					end = tk.End
				}
			}
		}
	}
}
