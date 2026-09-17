package shape

import (
	"fmt"
	"testing"
)

func newTestCache(t *testing.T) *LineCache {
	t.Helper()
	eng, err := NewEngine(1, false)
	if err != nil {
		t.Skip(err)
	}
	return eng.Cache
}

// TestLineCacheBounded is the regression guard for unbounded growth: scrolling
// through a large document used to retain every distinct line for the life of
// the process (~3.4 kB each).
func TestLineCacheBounded(t *testing.T) {
	c := newTestCache(t)
	c.SetBudget(64 << 10) // small budget so the bound is reached quickly

	for i := 0; i < 20000; i++ {
		c.GetMono(fmt.Sprintf(`      "field_%06d": "value-%06d",`, i, i))
	}

	// hot is capped by budget and cold holds at most one prior generation, so
	// retention stays within a small multiple of the budget regardless of how
	// many distinct lines were shaped.
	if got := c.bytes; got > c.budget {
		t.Errorf("hot generation over budget: %d > %d", got, c.budget)
	}
	if got := c.Len(); got >= 20000 {
		t.Errorf("cache retained every line: %d entries", got)
	}
	t.Logf("20000 distinct lines -> %d entries retained", c.Len())
}

// TestLineCacheReusesShapedLine covers the measure-then-draw path: layout
// measures a string and the paint pass draws it, and both must hit one shaping.
func TestLineCacheReusesShapedLine(t *testing.T) {
	c := newTestCache(t)

	first := c.GetAtWeight("Send Request", 13, WeightRegular)
	if len(first.Glyphs) == 0 {
		t.Fatal("expected shaped glyphs")
	}
	before := c.Len()
	second := c.GetAtWeight("Send Request", 13, WeightRegular)
	if c.Len() != before {
		t.Errorf("repeat lookup grew the cache: %d -> %d", before, c.Len())
	}
	if second.Width != first.Width || len(second.Glyphs) != len(first.Glyphs) {
		t.Error("repeat lookup returned a different line")
	}
}

// TestLineCachePromotesFromCold keeps a line that is still in use alive across
// an eviction, so steady-state UI text does not re-shape every generation.
func TestLineCachePromotesFromCold(t *testing.T) {
	c := newTestCache(t)
	c.SetBudget(8 << 10)

	const keep = "Requests"
	c.Get(keep)
	h := hashLineAt(keep, 0, false, WeightRegular)

	// Churn until the keep line has been demoted out of hot.
	for i := 0; i < 5000; i++ {
		c.Get(fmt.Sprintf("filler-%d", i))
		if _, inHot := c.hot[h]; !inHot {
			break
		}
	}
	if _, inHot := c.hot[h]; inHot {
		t.Skip("budget never forced an eviction")
	}
	if _, inCold := c.cold[h]; !inCold {
		t.Skip("line aged out of both generations")
	}

	c.Get(keep)
	if _, inHot := c.hot[h]; !inHot {
		t.Error("cold hit was not promoted back into hot")
	}
}

// TestLineCacheSkipsHugeLines keeps one pathological line from consuming a
// whole generation.
func TestLineCacheSkipsHugeLines(t *testing.T) {
	c := newTestCache(t)

	huge := make([]byte, maxCachedGlyphs+100)
	for i := range huge {
		huge[i] = 'x'
	}
	before := c.Len()
	ln := c.GetMono(string(huge))
	if len(ln.Glyphs) <= maxCachedGlyphs {
		t.Skipf("line shaped to %d glyphs, under the cap", len(ln.Glyphs))
	}
	if c.Len() != before {
		t.Errorf("huge line was cached: %d -> %d entries", before, c.Len())
	}
	if ln.Width <= 0 {
		t.Error("uncached line must still be shaped and returned")
	}
}

// BenchmarkFrameMeasure models one frame of layout: every widget measures its
// label on every rebuild. Before measurement went through the cache this was
// ~193 µs and ~1500 allocations per frame for these 16 strings.
func BenchmarkFrameMeasure(b *testing.B) {
	eng, err := NewEngine(1, false)
	if err != nil {
		b.Skip(err)
	}
	labels := []string{
		"Requests", "Environments", "Proto files", "Workspaces", "Commands",
		"Send", "Save", "Cancel", "Settings", "No Environment",
		"Default Workspace", "Body", "Headers", "Params", "Auth", "Pre Request",
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, s := range labels {
			eng.MeasureAtWeight(s, 13, WeightRegular)
		}
	}
}
