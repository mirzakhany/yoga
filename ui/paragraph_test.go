package ui

import (
	"strings"
	"testing"

	"github.com/mirzakhany/yoga/shape"
)

func newParagraphCtx(t *testing.T) (*Ctx, *shape.Engine) {
	t.Helper()
	text, err := shape.NewEngine(1, false)
	if err != nil {
		t.Fatal(err)
	}
	SetFrameResources(text, nil, nil)
	return New(text, NewFocusScope(), nil), text
}

func TestWrapTextSplitsLongWords(t *testing.T) {
	_, text := newParagraphCtx(t)
	long := "http://example.com/" + strings.Repeat("segment/", 20)
	lines := wrapText(text, "Get "+long+": refused", 13, 120)
	if len(lines) < 3 {
		t.Fatalf("want several lines, got %q", lines)
	}
	for _, l := range lines {
		if w, _ := text.MeasureAt(l, 13); w > 120 {
			t.Fatalf("line %q is %v wide, max 120", l, w)
		}
	}
	if got := strings.Join(lines, ""); strings.ReplaceAll(got, " ", "") != strings.ReplaceAll("Get "+long+": refused", " ", "") {
		t.Fatalf("wrapping lost text: %q", lines)
	}
}

func TestWrapTextKeepsNewlines(t *testing.T) {
	_, text := newParagraphCtx(t)
	lines := wrapText(text, "one\n\ntwo", 13, 500)
	if len(lines) != 3 || lines[0] != "one" || lines[1] != "" || lines[2] != "two" {
		t.Fatalf("got %q", lines)
	}
}

// A Paragraph in a Column takes the column width and grows tall enough for
// its wrapped lines in the same frame.
func TestParagraphHeightFollowsWrappedWidth(t *testing.T) {
	c, text := newParagraphCtx(t)
	msg := strings.Repeat("word ", 40)
	_, lineH := text.MeasureAt("Ag", c.Theme().Typography.Body.Size)

	build := func(w float32) float32 {
		c.BeginFrame(w, 800, nil, nil)
		root := Column(Paragraph(msg)).Layout(c)
		root.Calculate(w, 800)
		return root.Children[0].Frame.H
	}

	wide := build(2000)
	if wide != lineH {
		t.Fatalf("wide: height %v, want one line %v", wide, lineH)
	}
	narrow := build(150)
	want := float32(len(wrapText(text, msg, c.Theme().Typography.Body.Size, 150))) * lineH
	if narrow != want || narrow <= lineH {
		t.Fatalf("narrow: height %v, want %v", narrow, want)
	}
	// Steady frame at the same width: the cached wrap sizes it on the first pass.
	if again := build(150); again != want {
		t.Fatalf("steady frame: height %v, want %v", again, want)
	}
}
