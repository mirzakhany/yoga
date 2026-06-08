package components

import (
	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
	"github.com/mirzakhany/yoga/theme"
)

// DialogAction is a button in a dialog footer.
type DialogAction struct {
	Label   string
	Primary bool
	OnClick func()
}

// DialogHost manages modal dialogs with scrim.
type DialogHost struct {
	theme   *theme.Theme
	text    *shape.Engine
	sheet   *render.SpriteSheet
	clip    input.Clipboard
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
}

type dialogMode int

const (
	dialogMessage dialogMode = iota
	dialogInput
)

// NewDialogHost builds a dialog host. Mount El and scrim.El on app root.
func NewDialogHost(eng *shape.Engine, th *theme.Theme, sheet *render.SpriteSheet, clip input.Clipboard) *DialogHost {
	d := &DialogHost{theme: th, text: eng, sheet: sheet, clip: clip, scrim: NewScrim(th), width: 360}
	d.El = layout.New(layout.Box())
	d.El.Overlay = true
	d.El.Paint = d.paint
	d.El.OnMouse = d.onMouse
	return d
}

// ScrimEl returns the scrim overlay element.
func (d *DialogHost) ScrimEl() *layout.Element { return d.scrim.El }

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
	d.input = NewTextField(d.text, d.theme, d.sheet, d.clip, TextFieldConfig{
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
	d.Open = true
	pad := d.theme.Spacing.L
	bodyH := float32(80)
	if d.mode == dialogInput {
		bodyH = d.theme.Metrics.ControlHeight + pad
	}
	titleH := d.theme.Typography.Subtitle.LineHeight
	footerH := d.theme.Metrics.ControlHeight + pad
	d.height = pad + titleH + pad + bodyH + pad + footerH + pad
	d.El.Style = layout.Box().Absolute(0, 0).Size(d.width, d.height)
	d.El.ReapplyStyle()
}

// Close hides the dialog.
func (d *DialogHost) Close() {
	d.Open = false
	d.scrim.Hide()
}

// Position centers the dialog in the viewport.
func (d *DialogHost) Position(viewW, viewH float32) {
	if !d.Open {
		return
	}
	d.scrim.Show(0, 0, viewW, viewH)
	x := (viewW - d.width) / 2
	y := (viewH - d.height) / 2
	d.El.Style = layout.Box().Absolute(x, y).Size(d.width, d.height)
	d.El.ReapplyStyle()
}

func (d *DialogHost) paint(dl *render.DrawList, text *shape.Engine) {
	if !d.Open {
		return
	}
	f := d.El.Frame
	r := d.theme.Radius.Large
	drawElevationShadow(dl, f, r, d.theme.Elevation.ShadowLg)
	dl.AddRoundedRectBorder(f, r, d.theme.Stroke.Thin, d.theme.Chrome, d.theme.Border)
	pad := d.theme.Spacing.L
	y := f.Y + pad
	if d.title != "" {
		style := d.theme.Typography.Subtitle
		text.DrawStringTopAt(dl, d.title, f.X+pad, y, d.theme.Foreground, style.Size)
		y += style.LineHeight + pad
	}
	if d.mode == dialogMessage && d.body != "" {
		style := d.theme.Typography.Body
		text.DrawStringTopAt(dl, d.body, f.X+pad, y, d.theme.ForegroundMuted, style.Size)
		y += style.LineHeight + pad
	}
	if d.mode == dialogInput && d.input != nil {
		d.input.El.Frame = render.Rect{X: f.X + pad, Y: y, W: f.W - 2*pad, H: d.theme.Metrics.ControlHeight}
		d.input.El.Paint(dl, text)
		y += d.theme.Metrics.ControlHeight + pad
	}
	// footer buttons right-aligned
	btnY := f.Y + f.H - pad - d.theme.Metrics.ControlHeight
	bx := f.X + f.W - pad
	for i := len(d.actions) - 1; i >= 0; i-- {
		act := d.actions[i]
		variant := VariantSecondary
		if act.Primary {
			variant = VariantPrimary
		}
		btn := NewButtonVariant(text, d.theme, act.Label, variant, act.OnClick)
		bw := btn.El.Frame.W
		if bw <= 0 {
			tw, _ := text.MeasureAt(act.Label, d.theme.Typography.Body.Size)
			bw = tw + 2*d.theme.Spacing.M
		}
		bx -= bw
		btn.El.Frame = render.Rect{X: bx, Y: btnY, W: bw, H: d.theme.Metrics.ControlHeight}
		btn.El.Paint(dl, text)
		bx -= d.theme.Spacing.S
	}
}

func (d *DialogHost) onMouse(e *layout.Element, m *input.Mouse) {
	if !d.Open {
		return
	}
	if e.Frame.Contains(m.X, m.Y) {
		m.Consumed = true
		if d.input != nil {
			d.input.onMouse(d.input.El, m)
		}
		// hit-test footer buttons
		pad := d.theme.Spacing.L
		btnY := e.Frame.Y + e.Frame.H - pad - d.theme.Metrics.ControlHeight
		bx := e.Frame.X + e.Frame.W - pad
		for i := len(d.actions) - 1; i >= 0; i-- {
			act := d.actions[i]
			tw, _ := d.text.MeasureAt(act.Label, d.theme.Typography.Body.Size)
			bw := tw + 2*d.theme.Spacing.M
			bx -= bw
			br := render.Rect{X: bx, Y: btnY, W: bw, H: d.theme.Metrics.ControlHeight}
			if br.Contains(m.X, m.Y) && m.Released && act.OnClick != nil {
				act.OnClick()
			}
			bx -= d.theme.Spacing.S
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
