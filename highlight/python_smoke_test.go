//go:build !js

package highlight

import (
	"testing"
	"time"
)

func TestPythonHighlightSmoke(t *testing.T) {
	h := NewPython()
	defer h.Close()
	src := []byte(`# set token
@decorator
class Foo(Base):
    def run(self, x: int) -> None:
        if x is None or x > 1.5:
            chapar.set_env("token", f"Bearer {x}")
        return True
`)
	h.Update(src)

	var toks []Token
	for i := 0; i < 200; i++ {
		if got, ok := h.Poll(); ok {
			toks = got
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if len(toks) == 0 {
		t.Fatalf("expected Python tokens, got none")
	}

	got := map[string]ColorClass{}
	for _, tk := range toks {
		got[string(src[tk.Start:tk.End])] = tk.Class
	}
	want := map[string]ColorClass{
		"# set token": ClassComment,
		"@decorator":  ClassType,
		"Foo":         ClassType,
		"class":       ClassKeyword,
		"def":         ClassKeyword,
		"int":         ClassType,
		"None":        ClassKeyword,
		"1.5":         ClassNumber,
		`"token"`:     ClassString,
		"x":           ClassDefault, // inside the f-string interpolation
		"True":        ClassKeyword,
		"return":      ClassKeyword,
	}
	for text, class := range want {
		c, ok := got[text]
		if class == ClassDefault {
			if ok {
				t.Errorf("%q: colored %v, want uncolored", text, c)
			}
			continue
		}
		if !ok || c != class {
			t.Errorf("%q: got %v (present=%v), want %v", text, c, ok, class)
		}
	}
}
