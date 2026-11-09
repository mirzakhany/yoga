package input

import (
	"slices"
	"testing"
)

func TestMouseDropLastsOneFrame(t *testing.T) {
	var m Mouse
	m.Drop([]string{"/a.png"})
	m.Drop([]string{"/b.txt"})
	if want := []string{"/a.png", "/b.txt"}; !slices.Equal(m.Dropped, want) {
		t.Fatalf("Dropped = %v, want %v", m.Dropped, want)
	}
	m.EndFrame()
	if m.Dropped != nil {
		t.Fatalf("Dropped after EndFrame = %v, want nil", m.Dropped)
	}
}

func TestMemClipboardImage(t *testing.T) {
	tests := []struct {
		name     string
		set      func(c *MemClipboard)
		wantText string
		wantImg  bool
	}{
		{"empty", func(*MemClipboard) {}, "", false},
		{"text", func(c *MemClipboard) { c.Set("hi") }, "hi", false},
		{"image", func(c *MemClipboard) { c.SetImage([]byte("png")) }, "", true},
		{"text replaces image", func(c *MemClipboard) { c.SetImage([]byte("png")); c.Set("hi") }, "hi", false},
		{"image replaces text", func(c *MemClipboard) { c.Set("hi"); c.SetImage([]byte("png")) }, "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var c MemClipboard
			var _ ImageClipboard = &c
			tt.set(&c)
			if got := c.Get(); got != tt.wantText {
				t.Errorf("Get = %q, want %q", got, tt.wantText)
			}
			if _, ok := c.GetImage(); ok != tt.wantImg {
				t.Errorf("GetImage ok = %v, want %v", ok, tt.wantImg)
			}
		})
	}
}
