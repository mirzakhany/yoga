package lsp

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"
)

func TestManagerNavigationRequests(t *testing.T) {
	registerFake(t, ".nav", "nav", "nav")
	m := NewManager()
	path := filepath.Join(t.TempDir(), "x.nav")
	d := m.Open(path, content("func foo()\nfoo()\n"))
	defer d.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	defs, err := m.Definition(ctx, path, Position{Line: 1, Character: 1})
	if err != nil || len(defs) != 1 || defs[0].Path() != path || defs[0].Range.Start.Character != 5 {
		t.Fatalf("definition %+v %v", defs, err)
	}
	refs, err := m.References(ctx, path, Position{Line: 0, Character: 6}, true)
	if err != nil || len(refs) != 2 {
		t.Fatalf("references %+v %v", refs, err)
	}
	we, err := m.Rename(ctx, path, Position{Line: 0, Character: 6}, "bar")
	if err != nil {
		t.Fatal(err)
	}
	edits := we.Edits()[path]
	got, err := ApplyEdits([]byte("func foo()\nfoo()\n"), edits, m.Encoding(path))
	if err != nil || string(got) != "func bar()\nbar()\n" {
		t.Fatalf("rename result %q %v", got, err)
	}
	fmtEdits, err := m.Format(ctx, path, FormattingOptions{TabSize: 4})
	if err != nil || len(fmtEdits) != 1 {
		t.Fatalf("format %+v %v", fmtEdits, err)
	}
	if !m.Supports(path, "renameProvider") || m.Supports(path, "hoverProvider") {
		t.Fatal("Supports misreads capabilities")
	}
	if _, err := m.SignatureHelp(ctx, path, Position{}); !errors.Is(err, ErrUnsupported) {
		t.Fatalf("unsupported request: %v", err)
	}

	applied := make(chan WorkspaceEdit, 1)
	m.OnApplyEdit(func(we WorkspaceEdit) bool { applied <- we; return true })
	if err := m.ExecuteCommand(ctx, path, Command{Command: "fix"}); err != nil {
		t.Fatal(err)
	}
	select {
	case we := <-applied:
		if len(we.Edits()) != 1 {
			t.Fatalf("applyEdit %+v", we)
		}
	case <-ctx.Done():
		t.Fatal("no applyEdit")
	}
	if _, err := m.Definition(ctx, filepath.Join(t.TempDir(), "other.nav"), Position{}); !errors.Is(err, ErrNoDocument) {
		t.Fatalf("closed document: %v", err)
	}
}

func TestOffsetsAndEncodings(t *testing.T) {
	text := []byte("aé😀b\nxy")
	for _, enc := range []string{"utf-8", "utf-16", "utf-32"} {
		for _, off := range []int{0, 1, 3, 7, 8, 9, 11} {
			p := PositionAt(text, off, enc)
			if back := OffsetAt(text, p, enc); back != off {
				t.Errorf("%s: offset %d -> %+v -> %d", enc, off, p, back)
			}
		}
	}
	if p := PositionAt(text, 7, "utf-16"); p.Character != 4 {
		t.Errorf("utf-16 column for b: %+v", p)
	}
	if off := OffsetAt(text, Position{Line: 0, Character: 99}, "utf-16"); off != 8 {
		t.Errorf("clamp to line end: %d", off)
	}
	if _, err := ApplyEdits(text, []TextEdit{
		{Range: Range{End: Position{Character: 2}}}, {Range: Range{Start: Position{Character: 1}, End: Position{Character: 3}}},
	}, "utf-8"); err == nil {
		t.Error("overlapping edits must fail")
	}
}
