package ui

import (
	"path/filepath"
	"slices"
	"testing"

	"github.com/mirzakhany/yoga/shape"
)

func TestFormNumberAcceptsTypedValueWithinRange(t *testing.T) {
	text, err := shape.NewEngine(1, false)
	if err != nil {
		t.Fatal(err)
	}
	SetFrameResources(text, nil, nil)
	c := New(text, NewFocusScope(), nil)

	value := 14.0
	frame := func() {
		c.BeginFrame(400, 200, nil, nil)
		Form("f", FormNumber("size", "Size", "", value, 10, 22, 1, func(v float64) { value = v })).Layout(c)
		c.EndFrame()
	}
	frame()
	tf := c.Widget("size", nil).(*TextInput)
	tf.focused = true
	tf.selAnchor, tf.caret = 0, len(tf.Value)

	for _, r := range "16" {
		tf.HandleText([]rune{r})
		frame()
	}
	if value != 16 {
		t.Fatalf("value = %v, want 16", value)
	}
	if tf.Value != "16" {
		t.Fatalf("field text = %q, want %q", tf.Value, "16")
	}

	// Empty and out-of-range drafts survive while editing without committing.
	tf.selAnchor, tf.caret = 0, len(tf.Value)
	tf.HandleText([]rune{'3'})
	frame()
	if tf.Value != "3" || value != 16 {
		t.Fatalf("draft: text %q value %v, want %q and 16", tf.Value, value, "3")
	}

	// Blur discards the invalid draft and shows the committed value again.
	tf.Blur()
	frame()
	if tf.Value != "16" {
		t.Fatalf("after blur text = %q, want %q", tf.Value, "16")
	}
}

// A long description wraps beside the control instead of running under it.
func TestFormDescriptionWrapsBesideControl(t *testing.T) {
	item := FormText("o", "Name override", "The name used to verify the common name in the server certificate", "", nil)
	n := Form("f", item)
	_, root := pathListFrame(t, n, 400, 300)
	_, root = pathListFrame(t, n, 400, 300)

	row := root.Children[0].Children[0]
	if len(row.Children) != 2 {
		t.Fatalf("row has %d children, want text column and control", len(row.Children))
	}
	textCol, control := row.Children[0], row.Children[1]
	for _, el := range textCol.Children {
		if right := el.Frame.X + el.Frame.W; right > control.Frame.X {
			t.Fatalf("text ends at x=%v, past the control at x=%v", right, control.Frame.X)
		}
	}
	desc := textCol.Children[1]
	if desc.Frame.Y+desc.Frame.H > row.Frame.Y+row.Frame.H {
		t.Fatalf("wrapped description (bottom %v) spills out of its row (bottom %v)",
			desc.Frame.Y+desc.Frame.H, row.Frame.Y+row.Frame.H)
	}
}

func TestFormFilePicksAndClears(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ca.pem")
	var got []string
	n := Form("f", FormFile("cert", "Root certificate", "", path,
		[]FileFilter{{Label: "Certificates", Exts: []string{".pem"}}},
		func(p string) { got = append(got, p) }))
	c, root := pathListFrame(t, n, 600, 300)

	control := root.Children[0].Children[0].Children[1]
	if len(control.Children) != 2 {
		t.Fatalf("control has %d children, want pick and clear buttons", len(control.Children))
	}
	clickElement(root, control.Children[0])
	d := c.Files()
	if !d.Open {
		t.Fatal("clicking the file button should open the file dialog")
	}
	if d.dir != dir {
		t.Fatalf("dialog opened in %q, want the current file's folder %q", d.dir, dir)
	}
	if len(d.opts.Filters) != 1 || d.opts.Filters[0].Label != "Certificates" {
		t.Fatalf("dialog filters = %+v, want the row's filters", d.opts.Filters)
	}
	d.opts.OnConfirm([]string{"/other/client.pem"})

	clickElement(root, control.Children[1])
	if want := []string{"/other/client.pem", ""}; !slices.Equal(got, want) {
		t.Fatalf("OnText calls = %q, want %q", got, want)
	}
}

func TestFormFileWithoutPathHasNoClear(t *testing.T) {
	n := Form("f", FormFile("cert", "Root certificate", "", "", nil, nil))
	_, root := pathListFrame(t, n, 600, 300)
	control := root.Children[0].Children[0].Children[1]
	if control.Frame.W != 180 {
		t.Fatalf("empty file row control is %v wide, want only the 180px pick button", control.Frame.W)
	}
}
