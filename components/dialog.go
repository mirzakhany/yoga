package components

import (
	"github.com/mirzakhany/yoga"
	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
	"github.com/mirzakhany/yoga/theme"
	"github.com/mirzakhany/yoga/ui"
)

// DialogAction is a button in a dialog footer.
type DialogAction struct {
	Label   string
	Primary bool
	OnClick func()
}

// DialogHost manages modal dialogs with scrim.
type DialogHost struct {
	scrim   *Scrim
	El      *layout.Element
	Open    bool
	title   string
	body    string
	actions []DialogAction
	input   *TextField
	mode    dialogMode
	width   float32
	height  float32
	// bodyLines is the message wrapped to the dialog's content width.
	bodyLines []string
	// Last known viewport size (from Position) so open() can center immediately.
	viewW, viewH float32
}

type dialogMode int

const (
	dialogMessage dialogMode = iota
	dialogInput
)

// NewDialogHost builds a dialog host. Mount El and scrim.El on app root.
func NewDialogHost() *DialogHost {
	d := &DialogHost{scrim: NewScrim(), width: 360}
	d.El = layout.New(layout.Box())
	d.El.Overlay = true
	d.El.Paint = d.paint
	d.El.OnMouse = d.onMouse
	return d
}

// Layout is the new ui.View entry point. While open, the dialog positions
// itself against the current viewport and self-registers the scrim (below) and
// the dialog body (above) as overlays — no manual ScrimEl/El mounting. The
// dialog is modal and routes its own keys via Update/HandleKeys.
func (d *DialogHost) Layout(c *ui.Ctx) *layout.Element {
	if d.Open {
		w, h := c.Viewport()
		d.Position(w, h)
		c.Overlay(d.scrim.El)
		c.Overlay(d.El)
	}
	return d.El
}

// ShowError opens an error message dialog.
func (d *DialogHost) ShowError(title, message string, onOK func()) {
	d.mode = dialogMessage
	d.title = title
	d.body = message
	d.input = nil
	d.actions = []DialogAction{{Label: "OK", Primary: true, OnClick: func() {
		d.Close()
		if onOK != nil {
			onOK()
		}
	}}}
	d.open()
}

// ShowInput opens a dialog with a text field.
func (d *DialogHost) ShowInput(title, placeholder string, onOK func(value string), onCancel func()) {
	d.mode = dialogInput
	d.title = title
	d.body = ""
	d.input = NewTextField(TextFieldConfig{
		Placeholder: placeholder,
	})
	d.actions = []DialogAction{
		{Label: "Cancel", OnClick: func() {
			d.Close()
			if onCancel != nil {
				onCancel()
			}
		}},
		{Label: "OK", Primary: true, OnClick: func() {
			val := ""
			if d.input != nil {
				val = d.input.Value
			}
			d.Close()
			if onOK != nil {
				onOK(val)
			}
		}},
	}
	d.open()
}

func (d *DialogHost) open() {
	th := theme.Current()
	d.Open = true
	pad := th.Spacing.L
	style := th.Typography.Body
	d.bodyLines = nil
	var bodyH float32
	switch {
	case d.mode == dialogInput:
		bodyH = th.Metrics.ControlHeight + pad
	case d.body != "":
		d.bodyLines = wrapText(yoga.Text(), d.body, style.Size, d.width-2*pad)
		bodyH = float32(len(d.bodyLines))*style.LineHeight + pad
	}
	titleH := th.Typography.Subtitle.LineHeight
	footerH := th.Metrics.ControlHeight + pad
	d.height = pad + titleH + pad + bodyH + pad + footerH
	d.place()
}

// place centers the dialog using the last known viewport size and seeds the
// element frame so the dialog paints in its final position on the very next
// frame instead of flashing at the window origin.
func (d *DialogHost) place() {
	x := f32max(0, (d.viewW-d.width)/2)
	y := f32max(0, (d.viewH-d.height)/2)
	d.El.Style = layout.Box().Absolute(x, y).Size(d.width, d.height)
	d.El.ReapplyStyle()
	d.El.Frame = render.Rect{X: x, Y: y, W: d.width, H: d.height}
	if d.viewW > 0 && d.viewH > 0 {
		d.scrim.Show(0, 0, d.viewW, d.viewH)
	}
}

// Close hides the dialog.
func (d *DialogHost) Close() {
	d.Open = false
	d.scrim.Hide()
}

// Position centers the dialog in the viewport. Call once per frame after
// layout; it also records the viewport size so a dialog opened mid-frame is
// centered immediately.
func (d *DialogHost) Position(viewW, viewH float32) {
	d.viewW, d.viewH = viewW, viewH
	if !d.Open {
		return
	}
	d.place()
}

func (d *DialogHost) paint(dl *render.DrawList, text *shape.Engine) {
	if !d.Open {
		return
	}
	th := theme.Current()
	f := d.El.Frame
	r := th.Radius.Large
	drawElevationShadow(dl, f, r, th.Elevation.ShadowLg)
	dl.AddRoundedRectBorder(f, r, th.Stroke.Thin, th.Chrome, th.Border)
	pad := th.Spacing.L
	y := f.Y + pad
	if d.title != "" {
		style := th.Typography.Subtitle
		text.DrawStringTopAt(dl, d.title, f.X+pad, y, th.Foreground, style.Size)
		y += style.LineHeight + pad
	}
	if d.mode == dialogMessage && len(d.bodyLines) > 0 {
		style := th.Typography.Body
		dl.PushClip(render.Rect{X: f.X + pad, Y: f.Y, W: f.W - 2*pad, H: f.H})
		for _, line := range d.bodyLines {
			text.DrawStringTopAt(dl, line, f.X+pad, y, th.ForegroundMuted, style.Size)
			y += style.LineHeight
		}
		dl.PopClip()
		y += pad
	}
	if d.mode == dialogInput && d.input != nil {
		d.input.El.Frame = render.Rect{X: f.X + pad, Y: y, W: f.W - 2*pad, H: th.Metrics.ControlHeight}
		d.input.El.Paint(dl, text)
		y += th.Metrics.ControlHeight + pad
	}
	// footer buttons right-aligned
	btnY := f.Y + f.H - pad - th.Metrics.ControlHeight
	bx := f.X + f.W - pad
	for i := len(d.actions) - 1; i >= 0; i-- {
		act := d.actions[i]
		variant := VariantSecondary
		if act.Primary {
			variant = VariantPrimary
		}
		btn := NewButtonVariant(act.Label, variant, act.OnClick)
		bw := btn.El.Frame.W
		if bw <= 0 {
			tw, _ := text.MeasureAt(act.Label, th.Typography.Body.Size)
			bw = tw + 2*th.Spacing.M
		}
		bx -= bw
		btn.El.Frame = render.Rect{X: bx, Y: btnY, W: bw, H: th.Metrics.ControlHeight}
		btn.El.Paint(dl, text)
		bx -= th.Spacing.S
	}
}

func (d *DialogHost) onMouse(e *layout.Element, m *input.Mouse) {
	if !d.Open {
		return
	}
	th := theme.Current()
	if e.Frame.Contains(m.X, m.Y) {
		m.Consumed = true
		if d.input != nil {
			d.input.onMouse(d.input.El, m)
		}
		// hit-test footer buttons
		pad := th.Spacing.L
		btnY := e.Frame.Y + e.Frame.H - pad - th.Metrics.ControlHeight
		bx := e.Frame.X + e.Frame.W - pad
		for i := len(d.actions) - 1; i >= 0; i-- {
			act := d.actions[i]
			tw, _ := yoga.Text().MeasureAt(act.Label, th.Typography.Body.Size)
			bw := tw + 2*th.Spacing.M
			bx -= bw
			br := render.Rect{X: bx, Y: btnY, W: bw, H: th.Metrics.ControlHeight}
			if br.Contains(m.X, m.Y) && m.Released && act.OnClick != nil {
				act.OnClick()
			}
			bx -= th.Spacing.S
		}
	}
}

// Update advances dialog sub-widgets.
func (d *DialogHost) Update(m *input.Mouse) {
	if d.Open && d.input != nil {
		d.input.Update(m)
	}
}

// HandleKeys routes keys to the input field when open.
func (d *DialogHost) HandleKeys(keys []input.KeyEvent) {
	if d.Open && d.input != nil {
		d.input.HandleKeys(keys)
	}
}

func (d *DialogHost) HandleText(runes []rune) {
	if d.Open && d.input != nil {
		d.input.HandleText(runes)
	}
}
