package ui

import (
	"strings"
	"testing"

	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/theme"
)

// envSuggest completes the name typed after the last "{{" from names.
func envSuggest(names ...string) SuggestFunc {
	return func(value string, caret int) ([]Suggestion, int, int) {
		open := strings.LastIndex(value[:caret], "{{")
		if open < 0 {
			return nil, 0, 0
		}
		partial := value[open+2 : caret]
		var items []Suggestion
		for _, n := range names {
			if strings.HasPrefix(n, partial) {
				items = append(items, Suggestion{Label: n, Detail: "env", Insert: n + "}}"})
			}
		}
		return items, open + 2, caret
	}
}

func TestTextFieldSuggestOpensAndInserts(t *testing.T) {
	c, _ := textFieldTestEnv(t)
	tf, el := layoutTextField(t, c, "tf", "", false)
	tf.Suggest = envSuggest("host", "hostPort", "token")

	clickField(tf, el, el.Frame.X+2)
	for _, r := range "{{ho" {
		tf.HandleText([]rune{r})
	}
	if !tf.SuggestOpen() {
		t.Fatalf("popup should be open after typing {{ho")
	}
	if got := len(tf.sugg.menu.items); got != 2 {
		t.Fatalf("candidates: got %d want 2", got)
	}

	// Down moves to the second candidate, Enter accepts it.
	tf.HandleKeys([]input.KeyEvent{{Key: input.KeyDown}})
	if got := tf.sugg.menu.HoverIndex(); got != 1 {
		t.Fatalf("hover after Down: got %d want 1", got)
	}
	tf.HandleKeys([]input.KeyEvent{{Key: input.KeyEnter}})
	if tf.Value != "{{hostPort}}" {
		t.Fatalf("accepted value: got %q want %q", tf.Value, "{{hostPort}}")
	}
	if tf.caret != len(tf.Value) {
		t.Fatalf("caret after accept: got %d want %d", tf.caret, len(tf.Value))
	}
	if tf.SuggestOpen() {
		t.Fatalf("popup should close once a suggestion is accepted")
	}
}

func TestTextFieldSuggestEnterDoesNotSubmit(t *testing.T) {
	c, _ := textFieldTestEnv(t)
	tf, el := layoutTextField(t, c, "tf", "", false)
	tf.Suggest = envSuggest("host")
	submits := 0
	tf.OnSubmit = func(string) { submits++ }

	clickField(tf, el, el.Frame.X+2)
	for _, r := range "{{h" {
		tf.HandleText([]rune{r})
	}
	tf.HandleKeys([]input.KeyEvent{{Key: input.KeyEnter}})
	if submits != 0 {
		t.Fatalf("Enter accepted a suggestion, it must not submit (submits=%d)", submits)
	}
	// With the popup closed, Enter submits again.
	tf.HandleKeys([]input.KeyEvent{{Key: input.KeyEnter}})
	if submits != 1 {
		t.Fatalf("Enter after the popup closed: submits=%d want 1", submits)
	}
}

func TestTextFieldSuggestClosesOnEscapeAndBlur(t *testing.T) {
	c, _ := textFieldTestEnv(t)
	tf, el := layoutTextField(t, c, "tf", "", false)
	tf.Suggest = envSuggest("host")

	clickField(tf, el, el.Frame.X+2)
	tf.HandleText([]rune("{{"))
	if !tf.SuggestOpen() {
		t.Fatalf("popup should open after {{")
	}
	tf.HandleKeys([]input.KeyEvent{{Key: input.KeyEscape}})
	if tf.SuggestOpen() {
		t.Fatalf("Escape should close the popup")
	}

	tf.HandleText([]rune("h"))
	if !tf.SuggestOpen() {
		t.Fatalf("typing should reopen the popup")
	}
	tf.Blur()
	if tf.SuggestOpen() {
		t.Fatalf("blur should close the popup")
	}
}

func TestTextFieldSuggestBackspaceRefiltersThenCloses(t *testing.T) {
	c, _ := textFieldTestEnv(t)
	tf, el := layoutTextField(t, c, "tf", "", false)
	tf.Suggest = envSuggest("host", "token")

	clickField(tf, el, el.Frame.X+2)
	tf.HandleText([]rune("{{hoX"))
	if tf.SuggestOpen() {
		t.Fatalf("no candidate matches {{hoX, popup should be closed")
	}
	tf.HandleKeys([]input.KeyEvent{{Key: input.KeyBackspace}})
	if !tf.SuggestOpen() {
		t.Fatalf("backspace back to {{ho should reopen the popup")
	}
	for i := 0; i < 4; i++ {
		tf.HandleKeys([]input.KeyEvent{{Key: input.KeyBackspace}})
	}
	if tf.SuggestOpen() {
		t.Fatalf("popup should close once {{ is gone (value=%q)", tf.Value)
	}
}

func TestTextFieldSuggestSkippedForPasswordFields(t *testing.T) {
	c, _ := textFieldTestEnv(t)
	tf, el := layoutTextField(t, c, "tf", "", true)
	tf.Suggest = envSuggest("host")

	clickField(tf, el, el.Frame.X+2)
	tf.HandleText([]rune("{{h"))
	if tf.SuggestOpen() {
		t.Fatalf("a masked field must not offer completions")
	}
}

func TestClampSpansSortsAndDropsOverlaps(t *testing.T) {
	red := render.Color{R: 1, A: 1}
	spans := []TextSpan{
		{Start: 10, End: 14, Color: red},
		{Start: 2, End: 6, Color: red},
		{Start: 4, End: 8, Color: red},   // overlaps the span kept before it
		{Start: 0, End: 0, Color: red},   // empty
		{Start: 12, End: 99, Color: red}, // past the end
		{Start: -1, End: 3, Color: red},
	}
	got := clampSpans(spans, 16)
	want := []TextSpan{{Start: 2, End: 6, Color: red}, {Start: 10, End: 14, Color: red}}
	if len(got) != len(want) {
		t.Fatalf("spans: got %v want %v", got, want)
	}
	for i := range want {
		if got[i].Start != want[i].Start || got[i].End != want[i].End {
			t.Fatalf("span %d: got %v want %v", i, got[i], want[i])
		}
	}
}

func TestTextFieldHighlightSkipsPlaceholderAndMask(t *testing.T) {
	c, _ := textFieldTestEnv(t)
	tf, _ := layoutTextField(t, c, "tf", "{{host}}/v1", false)
	tf.Highlight = func(value string) []TextSpan {
		return []TextSpan{{Start: 0, End: 8, Color: render.Color{R: 1, A: 1}}}
	}
	if got := tf.highlightSpans(tf.Value); len(got) != 1 {
		t.Fatalf("value spans: got %d want 1", len(got))
	}
	if got := tf.highlightSpans("type a URL"); got != nil {
		t.Fatalf("placeholder must not be highlighted, got %v", got)
	}
	tf.cfg.Password = true
	if got := tf.highlightSpans(tf.Value); got != nil {
		t.Fatalf("masked value must not be highlighted, got %v", got)
	}
}

// paintField lays a field out inside a root element and paints it, returning the
// draw list so a test can look for the colors that reached the screen.
func paintField(t *testing.T, tf *TextInput, c *Ctx) *render.DrawList {
	t.Helper()
	c.BeginFrame(400, 80, nil, nil)
	host := tf.Layout(c)
	root := layout.New(layout.Box(), host)
	root.Calculate(400, 80)
	dl := &render.DrawList{}
	layout.Paint(root, dl, frameText())
	return dl
}

func hasColor(dl *render.DrawList, col render.Color) bool {
	for _, v := range dl.Vertices {
		if v.Col[0] == col.R && v.Col[1] == col.G && v.Col[2] == col.B && v.Col[3] == col.A {
			return true
		}
	}
	return false
}

func TestTextFieldPaintsHighlightedSpans(t *testing.T) {
	c, _ := textFieldTestEnv(t)
	tf, _ := layoutTextField(t, c, "tf", "{{host}}/x", false)
	marker := render.Color{R: 0.9, G: 0.2, B: 0.6, A: 1}
	tf.Highlight = func(string) []TextSpan {
		return []TextSpan{{Start: 0, End: 8, Color: marker}}
	}
	dl := paintField(t, tf, c)
	if !hasColor(dl, marker) {
		t.Fatal("the highlighted placeholder was painted in the plain text color")
	}
	if !hasColor(dl, theme.Current().Foreground) {
		t.Fatal("the text outside the span should keep the plain color")
	}
}

func TestTextFieldSuggestPopupSitsBelowTheField(t *testing.T) {
	c, _ := textFieldTestEnv(t)
	// A window with room under the field, as a real one has.
	c.BeginFrame(400, 300, nil, nil)
	el := TextField("tf", "").Width(300).Layout(c)
	root := layout.New(layout.Box().PaddingXY(0, 40), el)
	root.Calculate(400, 300)
	tf := c.Widget("tf", func() any { return NewTextInput(TextFieldConfig{}) }).(*TextInput)
	tf.Suggest = envSuggest("host", "hostPort")

	clickField(tf, el, el.Frame.X+2)
	tf.HandleText([]rune("{{h"))
	if !tf.SuggestOpen() {
		t.Fatal("popup should be open")
	}
	f, popup := tf.host.Frame, tf.sugg.menu.overlay().Frame
	// The popup must clear the field: below it when there is room, above it
	// otherwise. Covering the field would hide the name being typed.
	if popup.Y < f.Y+f.H && popup.Y+popup.H > f.Y {
		t.Fatalf("popup [%v,%v] covers the field [%v,%v]", popup.Y, popup.Y+popup.H, f.Y, f.Y+f.H)
	}
	if popup.Y < f.Y {
		t.Fatalf("with room below, the popup should open under the field (popup %v, field %v)", popup.Y, f.Y)
	}
	if popup.W < f.W {
		t.Fatalf("popup width %v should be at least the field width %v", popup.W, f.W)
	}
	if popup.X < 0 || popup.X+popup.W > 400 {
		t.Fatalf("popup [%v,%v] escapes the 400px viewport", popup.X, popup.X+popup.W)
	}
}
