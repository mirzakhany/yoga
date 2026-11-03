package ui

import (
	"strings"
	"testing"

	"github.com/mirzakhany/yoga/input"
)

// BenchmarkLineOpsLargeFile moves, duplicates and comments a line in the
// middle of a 10,000-line file, once per iteration each.
func BenchmarkLineOpsLargeFile(b *testing.B) {
	doc := strings.Repeat("\tfmt.Println(\"some line of code\", x, y)\n", 10000)
	e := newUndoEditor(b, doc)
	e.Path = "x.go"
	mid := e.Offset(5000, 3)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		e.SetSelection(mid, mid)
		e.MoveLines(1)
		e.DuplicateLines(1)
		e.ToggleComment()
		e.HandleKeys(key(input.KeyRight, input.ModAlt))
	}
}
