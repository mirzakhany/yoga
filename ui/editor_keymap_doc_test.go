package ui

import (
	"os"
	"strings"
	"testing"
)

const editorShortcutDoc = "../docs/editor-shortcuts.md"

// editorShortcutMarkdown renders the default keymap as the shortcut
// reference in docs/.
func editorShortcutMarkdown() string {
	var b strings.Builder
	b.WriteString("# Editor keyboard shortcuts\n\n")
	b.WriteString("Generated from `ui.DefaultEditorKeymap()` by `go test ./ui -run TestEditorShortcutDoc` ")
	b.WriteString("with `YOGA_UPDATE_DOCS=1`. Apps change bindings with `Editor.SetKeymap` or `EditorDefaults.Keymap`, ")
	b.WriteString("and list them with `EditorKeymapTable`.\n")
	scopes := []struct {
		title string
		in    func(EditorScope) bool
	}{
		{"Text", func(s EditorScope) bool { return s == ScopeText }},
		{"Find bar", func(s EditorScope) bool { return s != ScopeText }},
	}
	for _, sc := range scopes {
		b.WriteString("\n## " + sc.title + "\n\n| Action | macOS | Windows/Linux |\n|---|---|---|\n")
		for _, r := range EditorKeymapTable(nil) {
			if !sc.in(r.When) {
				continue
			}
			title := r.Title
			switch r.When {
			case ScopeFindQuery:
				title += " (find field)"
			case ScopeFindReplace:
				title += " (replace field)"
			}
			b.WriteString("| " + title + " | " + mdKey(r.Mac) + " | " + mdKey(r.Other) + " |\n")
		}
	}
	return b.String()
}

func mdKey(s string) string {
	if s == "" {
		return "—"
	}
	return "`" + strings.ReplaceAll(s, "|", `\|`) + "`"
}

func TestEditorShortcutDoc(t *testing.T) {
	want := editorShortcutMarkdown()
	if os.Getenv("YOGA_UPDATE_DOCS") != "" {
		if err := os.WriteFile(editorShortcutDoc, []byte(want), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	got, err := os.ReadFile(editorShortcutDoc)
	if err != nil || string(got) != want {
		t.Fatalf("%s is out of date; rerun with YOGA_UPDATE_DOCS=1", editorShortcutDoc)
	}
}
