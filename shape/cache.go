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
	return c.GetAt(text, 0)
}

// GetAt shapes text at logicalSize, returning a cached line when possible.
func (c *LineCache) GetAt(text string, logicalSize float32) Line {
	h := hashLineAt(text, logicalSize)
	if ln, ok := c.data[h]; ok {
		return ln
	}
	ln := c.shaper.ShapeLineAt(text, logicalSize)
	c.data[h] = ln
	return ln
}

// Invalidate clears the cache (e.g. after font scale change).
func (c *LineCache) Invalidate() {
	c.data = make(map[uint64]Line)
}

func hashLine(s string) uint64 { return hashLineAt(s, 0) }

func hashLineAt(s string, logicalSize float32) uint64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(s))
	if logicalSize != 0 {
		var buf [4]byte
		bits := uint32(logicalSize * 1000)
		buf[0] = byte(bits)
		buf[1] = byte(bits >> 8)
		buf[2] = byte(bits >> 16)
		buf[3] = byte(bits >> 24)
		_, _ = h.Write(buf[:])
	}
	return h.Sum64()
}
