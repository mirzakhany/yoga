package ui

import (
	"testing"

	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/shape"
	"github.com/mirzakhany/yoga/theme"
)

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
	pad := th.Spacing.M
	tw, _ := text.MeasureAt("OK", th.Typography.Body.Size)
	bw := tw + 2*th.Spacing.M
	f := d.panel.Frame
	mouse := &input.Mouse{
		X:        f.X + f.W - pad - bw/2,
		Y:        f.Y + f.H - pad - th.Metrics.ControlHeight/2,
		Pressed:  true,
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

func TestDialogPanelStacksAboveScrimWithPriorOverlay(t *testing.T) {
	text, err := shape.NewEngine(1, false)
	if err != nil {
		t.Fatal(err)
	}
	SetFrameResources(text, nil, nil)

	d := NewDialogHost()
	d.Show(DialogOpts{Title: "Test", Width: 400, Height: 300})

	prior := layout.New(layout.Box())
	prior.Overlay = true

	c := New(text, NewFocusScope(), nil)
	root := BuildFrame(c, func(c *Ctx) View {
		c.Overlay(prior)
		return d
	}, 800, 600, nil, nil)

	ov := c.Overlays()
	scrimI, panelI := -1, -1
	for i, e := range ov {
		if e == d.scrim.host {
			scrimI = i
		}
		if e == d.panel {
			panelI = i
		}
	}
	if scrimI < 0 || panelI < 0 {
		t.Fatalf("missing scrim/panel in overlays (n=%d scrim=%d panel=%d)", len(ov), scrimI, panelI)
	}
	if !(scrimI < panelI) {
		t.Fatalf("panel must paint above scrim: scrim=%d panel=%d", scrimI, panelI)
	}
	if ov[0] != prior {
		t.Fatal("prior overlay should remain first")
	}

	f := d.panel.Frame
	mouse := &input.Mouse{X: f.X + f.W/2, Y: f.Y + f.H/2, Released: true}
	layout.Dispatch(root, mouse)
	if !mouse.Consumed {
		t.Fatal("click on the panel should be consumed by the dialog")
	}
	_ = root
}

func TestDialogEscapeRunsOnDismiss(t *testing.T) {
	d := NewDialogHost()
	dismissed := false
	d.Show(DialogOpts{
		Title:     "Hi",
		OnDismiss: func() { dismissed = true },
	})
	d.HandleKeys([]input.KeyEvent{{Key: input.KeyEscape}})
	if d.Open {
		t.Fatal("escape should close")
	}
	if !dismissed {
		t.Fatal("OnDismiss should run")
	}
}

func TestDialogShowUsesRequestedSize(t *testing.T) {
	text, err := shape.NewEngine(1, false)
	if err != nil {
		t.Fatal(err)
	}
	SetFrameResources(text, nil, nil)

	d := NewDialogHost()
	d.Show(DialogOpts{Title: "Sized", Width: 520, Height: 400})

	c := New(text, NewFocusScope(), nil)
	BuildFrame(c, func(_ *Ctx) View { return d }, 800, 600, nil, nil)
	f := d.panel.Frame
	if f.W != 520 || f.H != 400 {
		t.Fatalf("panel size: got %.0fx%.0f want 520x400", f.W, f.H)
	}
}

func TestDialogShowInfoAndWarning(t *testing.T) {
	for _, tc := range []struct {
		name     string
		show     func(d *DialogHost, onOK func())
		title    string
		severity DialogSeverity
	}{
		{"info", func(d *DialogHost, onOK func()) { d.ShowInfo("Info", "note", onOK) }, "Info", DialogSeverityInfo},
		{"warning", func(d *DialogHost, onOK func()) { d.ShowWarning("Warning", "careful", onOK) }, "Warning", DialogSeverityWarning},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d := NewDialogHost()
			ok := false
			tc.show(d, func() { ok = true })
			if !d.Open {
				t.Fatal("dialog should be open")
			}
			if d.opts.Title != tc.title {
				t.Fatalf("title: got %q want %q", d.opts.Title, tc.title)
			}
			if d.opts.Severity != tc.severity {
				t.Fatalf("severity: got %v want %v", d.opts.Severity, tc.severity)
			}
			if len(d.opts.Actions) != 1 || d.opts.Actions[0].Label != "OK" || !d.opts.Actions[0].Primary {
				t.Fatalf("want single primary OK action, got %+v", d.opts.Actions)
			}
			d.Close()
			if d.opts.Actions[0].OnClick != nil {
				d.opts.Actions[0].OnClick()
			}
			if !ok {
				t.Fatal("OK should run onOK")
			}
		})
	}
}

func TestDialogShowErrorSeverity(t *testing.T) {
	d := NewDialogHost()
	d.ShowError("Error", "bad", nil)
	if d.opts.Severity != DialogSeverityError {
		t.Fatalf("severity: got %v want error", d.opts.Severity)
	}
}

func TestDialogShowActionYesAndEscape(t *testing.T) {
	d := NewDialogHost()
	var yes, no bool
	d.ShowAction("Delete?", "Cannot undo.", func() { yes = true }, func() { no = true })
	if !d.Open {
		t.Fatal("dialog should be open")
	}
	if len(d.opts.Actions) != 2 || d.opts.Actions[0].Label != "No" || d.opts.Actions[1].Label != "Yes" {
		t.Fatalf("want No then Yes, got %+v", d.opts.Actions)
	}
	if !d.opts.Actions[1].Primary {
		t.Fatal("Yes should be primary")
	}

	d.Close()
	if d.opts.Actions[1].OnClick != nil {
		d.opts.Actions[1].OnClick()
	}
	if !yes {
		t.Fatal("Yes should run onYes")
	}

	yes, no = false, false
	d.ShowAction("Delete?", "Cannot undo.", func() { yes = true }, func() { no = true })
	d.HandleKeys([]input.KeyEvent{{Key: input.KeyEscape}})
	if d.Open {
		t.Fatal("escape should close")
	}
	if !no {
		t.Fatal("escape should run onNo")
	}
	if yes {
		t.Fatal("escape must not run onYes")
	}
}

func TestDialogShowInputOKAndCancel(t *testing.T) {
	d := NewDialogHost()
	var okVal string
	var canceled bool
	d.ShowInput("Name", "enter", func(v string) { okVal = v }, func() { canceled = true })
	d.inputValue = "alice"

	for _, act := range d.opts.Actions {
		if act.Label == "OK" {
			d.Close()
			if act.OnClick != nil {
				act.OnClick()
			}
			break
		}
	}
	if okVal != "alice" {
		t.Fatalf("OK: got %q want alice", okVal)
	}

	d.ShowInput("Name", "enter", nil, func() { canceled = true })
	d.HandleKeys([]input.KeyEvent{{Key: input.KeyEscape}})
	if !canceled {
		t.Fatal("escape should run onCancel")
	}
}

func TestDialogShowInputNoIDCollisionWithButton(t *testing.T) {
	text, err := shape.NewEngine(1, false)
	if err != nil {
		t.Fatal(err)
	}
	SetFrameResources(text, nil, nil)

	c := New(text, NewFocusScope(), nil)
	body := func(c *Ctx) View {
		return Row(
			Button("dlg-input", Text("Open")).OnClick(func() { c.Dialogs().ShowInput("Rename", "name", nil, nil) }),
		)
	}
	BuildFrame(c, body, 800, 600, nil, nil)

	c.Dialogs().ShowInput("Rename", "name", nil, nil)
	// Must not panic: dialog TextField must not reuse a user button id.
	BuildFrame(c, body, 800, 600, nil, nil)
}

func TestSwitchToggle(t *testing.T) {
	text, err := shape.NewEngine(1, false)
	if err != nil {
		t.Fatal(err)
	}
	SetFrameResources(text, nil, nil)

	got := false
	c := New(text, NewFocusScope(), nil)
	c.BeginFrame(200, 100, nil, nil)
	el := Switch("sw").Check(got).OnToggle(func(v bool) { got = v }).Layout(c)
	mouse := &input.Mouse{
		X:        el.Frame.X + el.Frame.W/2,
		Y:        el.Frame.Y + el.Frame.H/2,
		Released: true,
	}
	layout.Dispatch(el, mouse)
	if !got {
		t.Fatal("switch click should toggle on")
	}
}

func TestFormLayout(t *testing.T) {
	c := New(nil, NewFocusScope(), nil)
	c.BeginFrame(400, 400, nil, nil)
	n := Form("settings",
		FormSwitch("s1", "Notifications", "Show alerts", true, nil),
		FormSelect("s2", "Theme", "Color scheme", []SelectOption{{Label: "Dark", Value: "dark"}}, 0, nil),
	)
	el := n.Layout(c)
	if el == nil {
		t.Fatal("form should layout")
	}
}
