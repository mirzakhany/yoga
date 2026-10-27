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
