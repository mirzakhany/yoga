package ui

import (
	"testing"

	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/shape"
	"github.com/mirzakhany/yoga/theme"
)

func TestSelectLayoutRegistersMenuOnlyWhenOpen(t *testing.T) {
	c := New(nil, NewFocusScope(), nil)
	n := Select("s", []SelectOption{{Label: "A", Value: "a"}, {Label: "B", Value: "b"}}).Width(120)

	c.BeginFrame(200, 200, nil, nil)
	n.Layout(c)
	if got := len(c.Overlays()); got != 0 {
		t.Fatalf("closed select should register no overlay, got %d", got)
	}

	st := c.Widget("s", func() any { return &selectState{} }).(*selectState)
	if st.menu == nil {
		t.Fatal("menu not allocated")
	}
	st.menu.OpenAt(0, 0)
	c.BeginFrame(200, 200, nil, nil)
	n.Layout(c)
	if got := len(c.Overlays()); got != 1 {
		t.Fatalf("open select should register its menu overlay, got %d", got)
	}
}

func TestSelectOnChangeUpdatesCaller(t *testing.T) {
	c := New(nil, NewFocusScope(), nil)
	got := "none"
	opts := []SelectOption{
		{Label: "None", Value: "none"},
		{Label: "Text", Value: "text"},
		{Label: "JSON", Value: "json"},
	}
	c.BeginFrame(200, 200, nil, nil)
	n := Select("body-type", opts).Selected(optionIndexForTest(got, opts)).OnChange(func(v string) { got = v })
	n.Layout(c)
	st := c.Widget("body-type", func() any { return &selectState{} }).(*selectState)
	if st.menu == nil || len(st.menu.items) < 3 {
		t.Fatal("select menu items missing")
	}
	st.menu.items[2].OnSelect()
	if got != "json" {
		t.Fatalf("OnChange: got %q want json", got)
	}
}

func optionIndexForTest(v string, opts []SelectOption) int {
	for i, o := range opts {
		if o.Value == v {
			return i
		}
	}
	return 0
}

func TestDialogLayoutRegistersScrimAndBodyWhenOpen(t *testing.T) {
	d := NewDialogHost()
	d.ShowError("oops", "bad", nil)

	c := New(nil, NewFocusScope(), nil)
	c.BeginFrame(400, 300, nil, nil)
	d.Layout(c)
	if got := len(c.Overlays()); got != 2 {
		t.Fatalf("open dialog should register scrim+body (2 overlays), got %d", got)
	}
}

func TestDialogOKClickCloses(t *testing.T) {
	text, err := shape.NewEngine(1, false)
	if err != nil {
		t.Fatal(err)
	}
	SetFrameResources(text, nil, nil)

	d := NewDialogHost()
	ok := false
	d.ShowError("Error", "Something failed unexpectedly.", func() { ok = true })

	c := New(text, NewFocusScope(), nil)
	root := BuildFrame(c, func(_ *Ctx) View { return d }, 800, 600, nil, nil)

	th := theme.Current()
	pad := th.Spacing.L
	tw, _ := text.MeasureAt("OK", th.Typography.Body.Size)
	bw := tw + 2*th.Spacing.M
	f := d.host.Frame
	mouse := &input.Mouse{
		X:        f.X + f.W - pad - bw/2,
		Y:        f.Y + f.H - pad - th.Metrics.ControlHeight/2,
		Released: true,
	}
	layout.Dispatch(root, mouse)

	if d.Open {
		t.Fatal("OK click should close the dialog")
	}
	if !ok {
		t.Fatal("OK click should run the ShowError callback")
	}
}
