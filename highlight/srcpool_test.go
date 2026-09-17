//go:build !js

package highlight

import (
	"fmt"
	"strings"
	"testing"
	"time"
	"unsafe"

	tree_sitter "github.com/tree-sitter/go-tree-sitter"
	tree_sitter_json "github.com/tree-sitter/tree-sitter-json/bindings/go"
)

// referenceTokens classifies src with a throwaway highlighter, giving an
// independent answer to compare the pooled path against.
func referenceTokens(t *testing.T, src []byte) []Token {
	t.Helper()
	h := NewJSON()
	defer h.Close()
	h.Update(src)
	toks, ok := pollFor(h, 10*time.Second)
	if !ok {
		t.Fatal("reference highlighter produced nothing")
	}
	return toks
}

// TestSourcePoolKeepsTokensCorrect is the guard on buffer reuse: if a recycled
// buffer were handed out while the worker still needed it — or if a tree
// retained the source it was parsed from — tokens would drift from the truth.
// Every edit is checked against an independent classification.
func TestSourcePoolKeepsTokensCorrect(t *testing.T) {
	h := NewJSON().(*tsHighlighter)
	defer h.Close()

	src := []byte(`{"name": "alice", "age": 30, "tags": ["a", "b"], "ok": true}`)
	h.Update(src)
	if _, ok := pollFor(h, 10*time.Second); !ok {
		t.Fatal("no initial tokens")
	}

	// Type a string value one character at a time, checking after each edit.
	const insertAt = 10 // inside the "alice" string literal
	for i := 0; i < 40; i++ {
		ch := byte('a' + i%26)
		next := make([]byte, 0, len(src)+1)
		next = append(next, src[:insertAt]...)
		next = append(next, ch)
		next = append(next, src[insertAt:]...)

		h.UpdateEdit(next, Edit{
			StartByte: insertAt, OldEndByte: insertAt, NewEndByte: insertAt + 1,
			Start:  Pt{Row: 0, Col: insertAt},
			OldEnd: Pt{Row: 0, Col: insertAt},
			NewEnd: Pt{Row: 0, Col: insertAt + 1},
		})
		got, ok := pollFor(h, 10*time.Second)
		if !ok {
			t.Fatalf("edit %d: no tokens", i)
		}
		src = next

		want := referenceTokens(t, src)
		if !sameTokens(got, want) {
			t.Fatalf("edit %d: tokens diverged from a fresh classification\nsrc:  %s\ngot:  %v\nwant: %v",
				i, src, head(got), head(want))
		}
	}
}

// TestSourcePoolReusesBuffers checks the pool actually recycles, rather than
// silently allocating every time.
func TestSourcePoolReusesBuffers(t *testing.T) {
	h := NewJSON().(*tsHighlighter)
	defer h.Close()

	src := []byte(largeJSONLine(64 << 10))
	h.Update(src)
	if _, ok := pollFor(h, 10*time.Second); !ok {
		t.Fatal("no initial tokens")
	}

	// After a parse the buffer must come back.
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		h.mu.Lock()
		n := len(h.srcPool)
		h.mu.Unlock()
		if n > 0 {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	h.mu.Lock()
	pooled := len(h.srcPool)
	var pooledCap int
	if pooled > 0 {
		pooledCap = cap(h.srcPool[0])
	}
	h.mu.Unlock()
	if pooled == 0 {
		t.Fatal("no buffer was recycled after parsing")
	}
	if pooledCap < len(src) {
		t.Errorf("recycled buffer lost its capacity: %d < %d", pooledCap, len(src))
	}

	// A follow-up edit must take that buffer rather than allocate.
	edited := append(append([]byte(nil), src...), ' ')
	h.UpdateEdit(edited, Edit{
		StartByte: len(src), OldEndByte: len(src), NewEndByte: len(src) + 1,
	})
	h.mu.Lock()
	after := len(h.srcPool)
	h.mu.Unlock()
	if after >= pooled {
		t.Errorf("edit did not consume a pooled buffer (pool %d -> %d)", pooled, after)
	}
	t.Logf("pool recycled a %d byte buffer and reused it", pooledCap)
}

// TestSourcePoolBounded keeps a burst of edits from pinning many
// document-sized buffers.
func TestSourcePoolBounded(t *testing.T) {
	h := NewJSON().(*tsHighlighter)
	defer h.Close()

	src := []byte(largeJSONLine(16 << 10))
	for i := 0; i < 50; i++ {
		h.Update(src)
	}
	time.Sleep(300 * time.Millisecond)

	h.mu.Lock()
	n := len(h.srcPool)
	h.mu.Unlock()
	if n > srcPoolMax {
		t.Errorf("pool holds %d buffers, cap is %d", n, srcPoolMax)
	}
}

// TestConcurrentUpdatesStayConsistent hammers the handoff from several
// goroutines, for the race detector.
func TestConcurrentUpdatesStayConsistent(t *testing.T) {
	h := NewJSON().(*tsHighlighter)
	defer h.Close()

	src := []byte(largeJSONLine(8 << 10))
	done := make(chan struct{})
	for g := 0; g < 4; g++ {
		go func(g int) {
			defer func() { done <- struct{}{} }()
			buf := append([]byte(nil), src...)
			for i := 0; i < 200; i++ {
				h.Update(buf)
				h.SetRange(i*16, i*16+512)
			}
		}(0)
	}
	for g := 0; g < 4; g++ {
		<-done
	}
	for i := 0; i < 50; i++ {
		h.Poll()
		time.Sleep(2 * time.Millisecond)
	}
}

func largeJSONLine(n int) string {
	var b strings.Builder
	b.WriteString(`{"items":[`)
	for i := 0; b.Len() < n; i++ {
		if i > 0 {
			b.WriteByte(',')
		}
		fmt.Fprintf(&b, `{"id":%d,"name":"item-%04d"}`, i, i)
	}
	b.WriteString(`]}`)
	return b.String()
}

// TestPooledBufferNeverAliasesPendingWork is the precise guard on the handoff.
//
// A buffer may only be pooled once the parse that read it has returned. If it
// were pooled earlier, a keystroke arriving mid-parse would refill the very
// bytes being parsed — which a sequential test cannot see, because nothing
// overwrites the buffer until the next edit. This checks the invariant itself:
// no pooled buffer may share a backing array with work that is still queued.
func TestPooledBufferNeverAliasesPendingWork(t *testing.T) {
	h := NewJSON().(*tsHighlighter)
	defer h.Close()

	src := []byte(largeJSONLine(512 << 10)) // big enough that a parse takes a while
	h.Update(src)

	edited := append([]byte(nil), src...)
	deadline := time.Now().Add(3 * time.Second)
	checks := 0
	for time.Now().Before(deadline) {
		// Keep edits arriving while parses are in flight.
		edited = append(edited, ' ')
		h.UpdateEdit(edited, Edit{
			StartByte: len(edited) - 1, OldEndByte: len(edited) - 1, NewEndByte: len(edited),
		})

		h.mu.Lock()
		for _, b := range h.srcPool {
			if h.pending.valid && len(b) > 0 && len(h.pending.src) > 0 &&
				sliceStart(b) == sliceStart(h.pending.src) {
				h.mu.Unlock()
				t.Fatal("a pooled buffer is still referenced by queued work: " +
					"it can be refilled while the worker parses it")
			}
		}
		h.mu.Unlock()
		checks++
		h.Poll()
	}
	if checks < 10 {
		t.Fatalf("only %d checks ran; the test did not exercise the handoff", checks)
	}
	t.Logf("%d interleaved edit/pool checks, no aliasing", checks)
}

// sliceStart identifies a slice's backing array.
func sliceStart(b []byte) uintptr { return uintptr(unsafe.Pointer(unsafe.SliceData(b))) }

// TestChunkedParseMatchesWholeBuffer guards the bounded read callback. Tokens
// that straddle a chunk boundary are the risk, so the document is sized to put
// many of them there and compared against the binding's own whole-buffer parse.
func TestChunkedParseMatchesWholeBuffer(t *testing.T) {
	p := tree_sitter.NewParser()
	defer p.Close()
	if err := p.SetLanguage(tree_sitter.NewLanguage(tree_sitter_json.Language())); err != nil {
		t.Fatal(err)
	}

	// Several chunks' worth, with long string literals that will cross the
	// 64 kB boundaries.
	var b strings.Builder
	b.WriteString(`{"items":[`)
	for i := 0; b.Len() < parseChunkBytes*5; i++ {
		if i > 0 {
			b.WriteByte(',')
		}
		fmt.Fprintf(&b, `{"id":%d,"blob":%q}`, i, strings.Repeat("x", 300))
	}
	b.WriteString(`]}`)
	src := []byte(b.String())
	if len(src) < parseChunkBytes*4 {
		t.Fatalf("fixture too small to cross chunk boundaries: %d bytes", len(src))
	}

	chunked := parse(p, src, nil)
	if chunked == nil {
		t.Fatal("chunked parse returned nil")
	}
	defer chunked.Close()
	whole := p.Parse(src, nil)
	if whole == nil {
		t.Fatal("whole-buffer parse returned nil")
	}
	defer whole.Close()

	if got, want := chunked.RootNode().ToSexp(), whole.RootNode().ToSexp(); got != want {
		t.Error("chunked parse produced a different tree than the whole-buffer parse")
	}

	gotToks := classifyWith(classifyJSON, chunked, 0, maxInt)
	wantToks := classifyWith(classifyJSON, whole, 0, maxInt)
	if !sameTokens(gotToks, wantToks) {
		t.Errorf("chunked tokens differ: got %d, want %d", len(gotToks), len(wantToks))
	}
	if len(gotToks) == 0 {
		t.Fatal("no tokens produced")
	}
	t.Logf("%d bytes over %d chunks -> %d identical tokens",
		len(src), (len(src)+parseChunkBytes-1)/parseChunkBytes, len(gotToks))
}
