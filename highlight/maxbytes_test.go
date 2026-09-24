//go:build !js

package highlight

import (
	"fmt"
	"math"
	"strings"
	"testing"
	"time"
)

func jsonOfSize(n int) []byte {
	var b strings.Builder
	b.Grow(n + 64)
	b.WriteString(`{"items":[`)
	for i := 0; b.Len() < n; i++ {
		if i > 0 {
			b.WriteByte(',')
		}
		fmt.Fprintf(&b, `{"id":%d,"name":"item-%06d"}`, i, i)
	}
	b.WriteString(`]}`)
	return []byte(b.String())
}

func pollFor(h Highlighter, d time.Duration) ([]Token, bool) {
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		if toks, ok := h.Poll(); ok {
			return toks, true
		}
		time.Sleep(5 * time.Millisecond)
	}
	return nil, false
}

// TestOversizeSourceIsNotParsed is the guard against the dominant memory cost
// of a large response body: a 10 MB JSON parses to ~3.5M Tree-sitter nodes and
// ~210 MB of resident C memory that the allocator never returns to the OS, all
// to color the ~50 lines a viewport shows.
func TestOversizeSourceIsNotParsed(t *testing.T) {
	h := NewJSON()
	defer h.Close()

	big := jsonOfSize(DefaultMaxBytes * 2)
	h.Update(big)

	toks, ok := pollFor(h, 2*time.Second)
	if !ok {
		t.Fatal("expected an empty token set to be delivered for an oversize source")
	}
	if len(toks) != 0 {
		t.Errorf("oversize source produced %d tokens; it must not be parsed", len(toks))
	}
	if ts, isTS := h.(*tsHighlighter); isTS {
		if !ts.wasOversize() {
			t.Error("highlighter did not record the source as oversize")
		}
		ts.mu.Lock()
		queued := ts.pending.valid
		ts.mu.Unlock()
		if queued {
			t.Error("oversize source was queued for parsing")
		}
	}
}

// TestUnderLimitStillHighlights makes sure the guard does not disable normal
// highlighting.
func TestUnderLimitStillHighlights(t *testing.T) {
	h := NewJSON()
	defer h.Close()

	h.Update([]byte(`{"key":"value","n":42}`))
	toks, ok := pollFor(h, 2*time.Second)
	if !ok {
		t.Fatal("no tokens delivered for a small source")
	}
	if len(toks) == 0 {
		t.Error("expected tokens for a small source")
	}
}

// TestCrossingBackUnderLimitReparses covers a document that shrinks below the
// limit after having exceeded it.
func TestCrossingBackUnderLimitReparses(t *testing.T) {
	h := NewJSON()
	defer h.Close()

	h.Update(jsonOfSize(DefaultMaxBytes * 2))
	if _, ok := pollFor(h, 2*time.Second); !ok {
		t.Fatal("expected the oversize clear to be delivered")
	}

	h.Update([]byte(`{"key":"value"}`))
	toks, ok := pollFor(h, 2*time.Second)
	if !ok {
		t.Fatal("no tokens after dropping back under the limit")
	}
	if len(toks) == 0 {
		t.Error("expected highlighting to resume under the limit")
	}
}

// TestMaxBytesIsConfigurable lets an app trade memory for highlighting on
// larger documents.
func TestMaxBytesIsConfigurable(t *testing.T) {
	orig := MaxBytes
	MaxBytes = 32
	defer func() { MaxBytes = orig }()

	h := NewJSON()
	defer h.Close()

	h.Update([]byte(`{"this":"is well over thirty-two bytes of json input"}`))
	toks, ok := pollFor(h, 2*time.Second)
	if !ok {
		t.Fatal("expected the oversize clear to be delivered")
	}
	if len(toks) != 0 {
		t.Errorf("MaxBytes was not honored: got %d tokens", len(toks))
	}
}

// TestSizeLimitedReportsAndLifts covers the SizeLimited capability: a consumer
// can tell an oversize document was left uncolored, and highlight it anyway.
func TestSizeLimitedReportsAndLifts(t *testing.T) {
	orig := MaxBytes
	MaxBytes = 64
	defer func() { MaxBytes = orig }()

	h, ok := NewJSON().(SizeLimited)
	if !ok {
		t.Skip("JSON highlighter is not size-limited on this platform")
	}
	defer h.Close()

	src := jsonOfSize(256)
	h.Update(src)
	if !h.Oversize() {
		t.Fatal("Oversize() = false for a source over the limit")
	}
	pollFor(h, 2*time.Second) // the clear

	h.SetMaxBytes(math.MaxInt)
	h.Update(src)
	if h.Oversize() {
		t.Fatal("Oversize() = true after lifting the limit")
	}
	toks, ok := pollFor(h, 2*time.Second)
	if !ok || len(toks) == 0 {
		t.Fatal("no tokens after lifting the limit")
	}
}
