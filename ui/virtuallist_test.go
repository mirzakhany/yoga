package ui

import (
	"testing"

	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/layout"
)

func TestVirtualListBuildsOnlyRowsInView(t *testing.T) {
	c := New(nil, NewFocusScope(), nil)
	built := map[int]bool{}
	body := func(_ *Ctx) View {
		return VirtualList("v", 10000, 20, func(i int) View {
			built[i] = true
			return Raw(layout.New(layout.Box()))
		}).Grow(1)
	}
	mouse := &input.Mouse{X: 40, Y: 40}
	BuildFrame(c, body, 400, 200, mouse, nil)
	clear(built)
	root := BuildFrame(c, body, 400, 200, mouse, nil)
	if len(built) == 0 || len(built) > 200/20+2*virtualOverscan+2 {
		t.Fatalf("built %d rows, want about one view's worth", len(built))
	}
	sv := c.Widget("v", nil).(*ScrollView)
	if sv.contentH != 10000*20 {
		t.Fatalf("content height %v, want %v", sv.contentH, 10000*20)
	}

	sv.scrollY = 5000 * 20
	clear(built)
	BuildFrame(c, body, 400, 200, mouse, nil)
	_ = root
	if !built[5000] || built[0] {
		t.Fatalf("after scrolling, rows built %v; want around 5000", len(built))
	}
}

func TestVirtualRange(t *testing.T) {
	if lo, hi := virtualRange(0, 20, 0, 100); lo != 0 || hi != 0 {
		t.Fatalf("empty: %d..%d", lo, hi)
	}
	if lo, hi := virtualRange(5, 20, 0, 100); lo != 0 || hi != 5 {
		t.Fatalf("short list: %d..%d", lo, hi)
	}
	if lo, hi := virtualRange(1000, 20, 400, 100); lo != 20-virtualOverscan || hi != 26+virtualOverscan {
		t.Fatalf("middle: %d..%d", lo, hi)
	}
}
