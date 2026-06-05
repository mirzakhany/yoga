package shape

import "hash/fnv"

// LineCache caches shaped lines keyed by content hash.
type LineCache struct {
	shaper *Shaper
	data   map[uint64]Line
}

// NewLineCache returns a cache backed by shaper.
func NewLineCache(shaper *Shaper) *LineCache {
	return &LineCache{shaper: shaper, data: make(map[uint64]Line)}
}

// Get shapes text, returning a cached line when possible.
func (c *LineCache) Get(text string) Line {
	h := hashLine(text)
	if ln, ok := c.data[h]; ok {
		return ln
	}
	ln := c.shaper.ShapeLine(text)
	c.data[h] = ln
	return ln
}

// Invalidate clears the cache (e.g. after font scale change).
func (c *LineCache) Invalidate() {
	c.data = make(map[uint64]Line)
}

func hashLine(s string) uint64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(s))
	return h.Sum64()
}
