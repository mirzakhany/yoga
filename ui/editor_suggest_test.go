package ui

import (
	"strings"
	"testing"

	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/theme"
)

// varSuggest completes a name typed after the last "{{", the way a template
// variable source does.
func varSuggest(names ...string) SuggestFunc {
	return func(text string, caret int) ([]Suggestion, int, int) {
		open := strings.LastIndex(text[:caret], "{{")
		if open < 0 {
			return nil, 0, 0
		}
		partial := text[open+2 : caret]
		if strings.ContainsAny(partial, " \n}") {
			return nil, 0, 0
		}
		// Add only the closing braces that are not already there, the way a
		// real source must, since the editor auto-closes brackets.
		closing := "}}"
		if rest := text[caret:]; strings.HasPrefix(rest, "}}") {
			closing = ""
		} else if strings.HasPrefix(rest, "}") {
			closing = "}"
		}
		var items []Suggestion
		for _, n := range names {
			if strings.HasPrefix(n, partial) {
				items = append(items, Suggestion{Label: n, Detail: "env", Insert: n + closing})
			}
		}
		return items, open + 2, caret
	}
}

func typeInEditor(e *Editor, s string) {
	for _, r := range s {
		e.HandleText([]rune{r})
	}
}

func TestEditorSuggestCompletesWithoutLanguageServer(t *testing.T) {
	e := newTestEditor(t, []byte(""))
	e.Suggest = varSuggest("host", "hostPort")
	e.Focus()

	typeInEditor(e, "{{ho")
	if !e.lspUI.compOpen {
		t.Fatal("the application source should open the popup with no server attached")
	}
	if got := len(e.lspUI.compItems); got != 2 {
		t.Fatalf("candidates: got %d want 2", got)
	}

	e.HandleKeys([]input.KeyEvent{{Key: input.KeyDown}})
	e.HandleKeys([]input.KeyEvent{{Key: input.KeyEnter}})
	if got := string(e.Bytes()); got != "{{hostPort}}" {
		t.Fatalf("accepted text: got %q want %q", got, "{{hostPort}}")
	}
	if e.lspUI.compOpen {
		t.Fatal("popup should close once a candidate is accepted")
	}
}

func TestEditorSuggestClosesWhenNothingMatches(t *testing.T) {
	e := newTestEditor(t, []byte(""))
	e.Suggest = varSuggest("host")
	e.Focus()

	typeInEditor(e, "{{ho")
	if !e.lspUI.compOpen {
		t.Fatal("popup should be open")
	}
	typeInEditor(e, "X")
	if e.lspUI.compOpen {
		t.Fatal("no candidate matches {{hoX, popup should have closed")
	}
	e.HandleKeys([]input.KeyEvent{{Key: input.KeyBackspace}})
	if !e.lspUI.compOpen {
		t.Fatal("backspacing to {{ho should reopen the popup")
	}
	e.HandleKeys([]input.KeyEvent{{Key: input.KeyEscape}})
	if e.lspUI.compOpen {
		t.Fatal("Escape should close the popup")
	}
}

func TestEditorSuggestLeavesOrdinaryTypingAlone(t *testing.T) {
	e := newTestEditor(t, []byte(""))
	e.Suggest = varSuggest("host")
	e.Focus()

	typeInEditor(e, "GET /users")
	if e.lspUI.compOpen {
		t.Fatal("plain text should not open a popup")
	}
	if got := string(e.Bytes()); got != "GET /users" {
		t.Fatalf("text: got %q", got)
	}
}

func TestEditorHoverCardWithoutLanguageServer(t *testing.T) {
	e := newTestEditor(t, []byte("{{host}}/v1"))
	e.HoverInfo = func(text string, off int) (HoverCard, bool) {
		if off < 8 {
			return HoverCard{Title: "host · env", Body: "https://api.example.com", Start: 0, End: 8}, true
		}
		return HoverCard{}, false
	}

	card, ok := e.hoverCardAt(2)
	if !ok || card.Title != "host · env" {
		t.Fatalf("hover over the placeholder: got %+v ok=%v", card, ok)
	}
	if _, ok := e.hoverCardAt(9); ok {
		t.Fatal("hover over plain text should say nothing")
	}

	// The card's lines lead the tooltip.
	e.lspUI.hoverCard, e.lspUI.hoverCardOK = card, true
	lines := e.tooltipLines(theme.Current())
	if len(lines) != 2 || lines[0].text != "host · env" || lines[1].text != "https://api.example.com" {
		t.Fatalf("tooltip lines: %+v", lines)
	}
}
