package highlight

import (
	"testing"
	"time"
)

func TestJSONHighlightSmoke(t *testing.T) {
	h := NewJSON()
	defer h.Close()
	src := []byte(`{"name": "alice", "age": 30, "active": true, "tags": [1, 2.5, null]}`)
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
		t.Fatalf("expected JSON tokens, got none")
	}

	var keys, strings, numbers, keywords int
	for _, tk := range toks {
		switch tk.Class {
		case ClassType:
			keys++
		case ClassString:
			strings++
		case ClassNumber:
			numbers++
		case ClassKeyword:
			keywords++
		}
	}
	t.Logf("tokens=%d keys=%d strings=%d numbers=%d keywords=%d", len(toks), keys, strings, numbers, keywords)
	if keys == 0 || numbers == 0 || keywords == 0 {
		t.Fatalf("missing expected classes: keys=%d numbers=%d keywords=%d", keys, numbers, keywords)
	}
}
