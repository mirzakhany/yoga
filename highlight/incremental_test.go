//go:build !js

package highlight

import (
	"testing"
	"time"
)

func TestJSONIncrementalHighlight(t *testing.T) {
	h := NewJSON()
	defer h.Close()

	src := []byte(`{"name": "alice"}`)
	h.Update(src)

	var baseline []Token
	for i := 0; i < 200; i++ {
		if got, ok := h.Poll(); ok {
			baseline = got
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if len(baseline) == 0 {
		t.Fatal("expected baseline tokens from full parse")
	}

	// Insert a new field after the opening brace.
	newSrc := []byte(`{"age": 30, "name": "alice"}`)
	h.UpdateEdit(newSrc, Edit{
		StartByte: 1, OldEndByte: 1, NewEndByte: 12,
		Start: Pt{Row: 0, Col: 1},
		OldEnd: Pt{Row: 0, Col: 1},
		NewEnd: Pt{Row: 0, Col: 12},
	})

	var after []Token
	for i := 0; i < 200; i++ {
		if got, ok := h.Poll(); ok {
			after = got
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if len(after) == 0 {
		t.Fatal("expected tokens after incremental parse")
	}

	var numbers int
	for _, tk := range after {
		if tk.Class == ClassNumber {
			numbers++
		}
	}
	if numbers == 0 {
		t.Fatalf("expected number token after incremental edit, tokens=%d", len(after))
	}
}
