package ui

import (
	"testing"

	"github.com/mirzakhany/yoga/lsp"
)

// TestRecomputeDiagSpansGrowsAndShrinks covers a diagnostics update that has
// more entries than the previous one (the first publish for a document, for
// example) and one that has fewer.
func TestRecomputeDiagSpansGrowsAndShrinks(t *testing.T) {
	e := newTestEditor(t, []byte("alpha\nbeta\ngamma\n"))
	at := func(line, from, to int) lsp.Diagnostic {
		return lsp.Diagnostic{
			Severity: lsp.SeverityError,
			Message:  "bad",
			Range:    lsp.Range{Start: lsp.Position{Line: line, Character: from}, End: lsp.Position{Line: line, Character: to}},
		}
	}

	e.lspUI.diags = []lsp.Diagnostic{at(0, 0, 5), at(1, 0, 4)}
	e.recomputeDiagSpans()
	if got := len(e.lspUI.diagSpans); got != 2 {
		t.Fatalf("spans = %d, want 2", got)
	}
	if s := e.lspUI.diagSpans[1]; s.lo != 6 || s.hi != 10 {
		t.Fatalf("second span = [%d,%d), want [6,10)", s.lo, s.hi)
	}

	e.lspUI.diags = []lsp.Diagnostic{at(2, 0, 5)}
	e.recomputeDiagSpans()
	if got := len(e.lspUI.diagSpans); got != 1 {
		t.Fatalf("spans = %d, want 1", got)
	}

	e.lspUI.diags = []lsp.Diagnostic{at(0, 0, 1), at(1, 0, 1), at(2, 0, 1)}
	e.recomputeDiagSpans()
	if got := len(e.lspUI.diagSpans); got != 3 {
		t.Fatalf("spans = %d, want 3", got)
	}
}
