package ui

import (
	"testing"
	"time"

	"github.com/mirzakhany/yoga/shape"
)

func TestBuildFrameLaysOutWindowDialog(t *testing.T) {
	text, err := shape.NewEngine(1, false)
	if err != nil {
		t.Fatal(err)
	}
	SetFrameResources(text, nil, nil)

	c := New(text, NewFocusScope(), nil)
	c.Dialogs().ShowError("oops", "bad", nil)

	root := BuildFrame(c, func(_ *Ctx) View {
		return Text("page")
	}, 800, 600, nil, nil)
	if root == nil {
		t.Fatal("nil root")
	}
	d := c.Dialogs()
	if !d.Open {
		t.Fatal("dialog should stay open")
	}
	if d.panel == nil {
		t.Fatal("window dialog should layout without being in the body tree")
	}
	if d.panel.Frame.W <= 0 || d.panel.Frame.H <= 0 {
		t.Fatalf("panel frame: %+v", d.panel.Frame)
	}
}

func TestBuildFrameLaysOutWindowFileDialog(t *testing.T) {
	text, err := shape.NewEngine(1, false)
	if err != nil {
		t.Fatal(err)
	}
	SetFrameResources(text, nil, nil)

	c := New(text, NewFocusScope(), nil)
	c.Files().Show(FileDialogOpts{Dir: t.TempDir(), Title: "Open"})

	BuildFrame(c, func(_ *Ctx) View { return Text("page") }, 800, 600, nil, nil)
	f := c.Files()
	if !f.Open {
		t.Fatal("file dialog should stay open")
	}
	if f.panel == nil || f.panel.Frame.W <= 0 {
		t.Fatal("window file picker should layout without being in the body tree")
	}
}

func TestBuildFrameLaysOutWindowToasts(t *testing.T) {
	c := New(nil, NewFocusScope(), nil)
	c.Toasts().Show("saved", ToastSuccess, time.Second)

	BuildFrame(c, func(_ *Ctx) View { return Text("page") }, 400, 300, nil, nil)
	if d, ok := c.Toasts().AnimationWait(); !ok || d > time.Second {
		t.Fatalf("toast AnimationWait = (%v, %v)", d, ok)
	}
	if len(c.Overlays()) == 0 {
		t.Fatal("window toasts should register an overlay without being in the body tree")
	}
}
