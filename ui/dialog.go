package ui

import (
	"fmt"

	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/theme"
)

// DialogAction is a button in a dialog footer.
type DialogAction struct {
	Label    string
	Primary  bool
	Disabled bool
	OnClick  func()
}

// DialogOpts configures one Show of a DialogHost.
type DialogOpts struct {
	Title     string
	Width     float32 // default 480 if 0
	Height    float32 // default 360 if 0
	Body      func(c *Ctx) View
	Actions   []DialogAction
	OnDismiss func() // Escape
}

// DialogHost manages modal dialogs with scrim.
type DialogHost struct {
	scrim *Scrim
	panel *layout.Element
	Open  bool
	opts  DialogOpts
	// inputValue holds the controlled text for ShowInput.
	inputValue string
}

var _ View = (*DialogHost)(nil)
var _ Focusable = (*DialogHost)(nil)

// NewDialogHost builds a dialog host. The window Ctx owns a default host
// (c.Dialogs()); construct a dedicated one only for tests or a second
// picker, and place that one in the view tree so Layout can register overlays.
func NewDialogHost() *DialogHost {
	return &DialogHost{scrim: NewScrim()}
}

// Show opens a dialog with custom size, body layout, and footer actions.
func (d *DialogHost) Show(opts DialogOpts) {
	d.opts = opts
	d.Open = true
}

// ShowError opens an error message dialog.
func (d *DialogHost) ShowError(title, message string, onOK func()) {
	th := theme.Current()
	width := float32(360)
	pad := th.Spacing.L
	style := th.Typography.Body
	bodyLines := wrapText(frameText(), message, style.Size, width-2*pad)
	bodyH := float32(len(bodyLines))*style.LineHeight + pad
	titleH := th.Typography.Subtitle.LineHeight
	footerH := th.Metrics.ControlHeight + pad
	height := pad + titleH + pad + bodyH + pad + footerH

	dismiss := func() {
		if onOK != nil {
			onOK()
		}
	}

	d.Show(DialogOpts{
		Title:  title,
		Width:  width,
		Height: height,
		Body: func(c *Ctx) View {
			th := c.Theme()
			pad := th.Spacing.L
			lines := make([]View, 0, len(bodyLines))
			for _, line := range bodyLines {
				lines = append(lines, Muted(line))
			}
			return Column(lines...).Padding(pad)
		},
		Actions:   []DialogAction{{Label: "OK", Primary: true, OnClick: dismiss}},
		OnDismiss: dismiss,
	})
}

// ShowInput opens a dialog with a text field.
func (d *DialogHost) ShowInput(title, placeholder string, onOK func(value string), onCancel func()) {
	th := theme.Current()
	pad := th.Spacing.L
	titleH := th.Typography.Subtitle.LineHeight
	bodyH := th.Metrics.ControlHeight + pad
	footerH := th.Metrics.ControlHeight + pad
	height := pad + titleH + pad + bodyH + pad + footerH

	d.inputValue = ""
	d.Show(DialogOpts{
		Title:  title,
		Width:  360,
		Height: height,
		Body: func(c *Ctx) View {
			th := c.Theme()
			pad := th.Spacing.L
			return Column(
				TextField("__dialog-input", d.inputValue).
					Placeholder(placeholder).
					OnChange(func(s string) { d.inputValue = s }).
					DefaultFocus(),
			).Padding(pad).Gap(th.Spacing.S)
		},
		Actions: []DialogAction{
			{Label: "Cancel", OnClick: onCancel},
			{Label: "OK", Primary: true, OnClick: func() {
				if onOK != nil {
					onOK(d.inputValue)
				}
			}},
		},
		OnDismiss: onCancel,
	})
}

// Layout registers the scrim and panel while open.
func (d *DialogHost) Layout(c *Ctx) *layout.Element {
	if !d.Open {
		return layout.New(layout.Box().Size(0, 0))
	}
	vw, vh := c.Viewport()
	th := c.Theme()
	dw, dh := clampModalSize(vw, vh, d.opts.Width, d.opts.Height, 0, 0, th)
	d.panel = layoutModalPanel(c, d.scrim, d.chrome(c), dw, dh)
	if c.Focus() != nil {
		c.Focus().SetModal(d)
	}
	return layout.New(layout.Box().Size(0, 0))
}

func (d *DialogHost) chrome(c *Ctx) View {
	th := c.Theme()
	var body View
	if d.opts.Body != nil {
		body = d.opts.Body(c)
	}

	buttons := make([]View, 0, len(d.opts.Actions)+1)
	buttons = append(buttons, Spacer())
	for i, act := range d.opts.Actions {
		btn := Button(fmt.Sprintf("dlg-act-%d", i), Text(act.Label))
		if act.Primary {
			btn = btn.Primary()
		}
		if act.Disabled {
			btn = btn.Disabled(true)
		}
		actCopy := act
		btn = btn.OnClick(func() {
			d.Close()
			if actCopy.OnClick != nil {
				actCopy.OnClick()
			}
		})
		buttons = append(buttons, btn)
	}
	footer := Row(buttons...).Gap(th.Spacing.S).Padding(th.Spacing.M)

	kids := make([]View, 0, 4)
	if d.opts.Title != "" {
		kids = append(kids,
			Row(Text(d.opts.Title).Style(Spec{}.TextColor(TokenForeground))).Padding(th.Spacing.M),
			HLine(th.Stroke.Thin, th.Border),
		)
	}
	if body != nil {
		kids = append(kids, ViewOf(body).Grow(1))
	}
	kids = append(kids, footer)

	return Column(kids...).Grow(1).Background(TokenChrome).
		Style(Spec{}.Radius(th.Radius.Large).Border(TokenBorder, th.Stroke.Thin))
}

// Close hides the dialog.
func (d *DialogHost) Close() {
	d.Open = false
	d.scrim.Hide()
}

func (d *DialogHost) Focus()                   {}
func (d *DialogHost) Blur()                    {}
func (d *DialogHost) Focused() bool            { return d.Open }
func (d *DialogHost) CapturesTab() bool        { return false }
func (d *DialogHost) FocusOnClick() bool       { return false }
func (d *DialogHost) FocusEl() *layout.Element { return d.panel }
func (d *DialogHost) HandleText([]rune)        {}

func (d *DialogHost) HandleKeys(keys []input.KeyEvent) {
	if !d.Open {
		return
	}
	for _, ev := range keys {
		if ev.Key == input.KeyEscape {
			d.Close()
			if d.opts.OnDismiss != nil {
				d.opts.OnDismiss()
			}
			return
		}
	}
}
