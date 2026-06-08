package main

import (
	"fmt"
	"time"

	"github.com/mirzakhany/yoga/components"
	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
	"github.com/mirzakhany/yoga/theme"
)

// ComponentGallery is the components demo page.
type ComponentGallery struct {
	root    *layout.Element
	theme   *theme.Theme
	text    *shape.Engine
	clip    input.Clipboard
	sheet   *render.SpriteSheet
	scroll  *components.ScrollView
	focus   *components.FocusManager
	status  string
	spinner *components.Spinner

	checkA   *components.Checkbox
	checkB   *components.Checkbox
	radioGrp *components.RadioGroup
	selectW  *components.Select
	tagEdit  *components.TagEdit
	dropdown *components.Dropdown
	navVert  *components.Navigation
	navHoriz *components.Navigation
	dialogs  *components.DialogHost
	toasts   *components.ToastHost
	statusEl *layout.Element
}

func buildComponentGallery(text *shape.Engine, clip input.Clipboard, sheet *render.SpriteSheet, dialogs *components.DialogHost, toasts *components.ToastHost) *ComponentGallery {
	th := theme.Current()
	g := &ComponentGallery{
		theme: th, text: text, clip: clip, sheet: sheet,
		dialogs: dialogs, toasts: toasts, status: "Interact with the widgets above",
	}
	g.spinner = components.NewSpinner(th, 24)

	sections := layout.New(layout.Box().Direction(layout.Column).Gap(th.Spacing.L).PaddingAll(th.Spacing.L))

	// 1. Labels
	sections.Children = append(sections.Children, sectionColumn(th,
		components.NewLabel(text, th, "Labels & Typography", components.LabelSubtitle).El,
		row(th,
			components.NewLabel(text, th, "Body label", components.LabelBody).El,
			components.NewLabel(text, th, "Caption", components.LabelCaption).El,
			components.NewLabel(text, th, "Muted", components.LabelMuted).El,
			components.NewLabel(text, th, "Strong", components.LabelStrong).El,
		),
	))

	// 2. Buttons
	btnPrimary := components.NewButtonVariant(text, th, "Primary", components.VariantPrimary, func() {
		g.setStatus("Primary clicked")
	})
	btnSecondary := components.NewButtonVariant(text, th, "Secondary", components.VariantSecondary, func() {
		g.setStatus("Secondary clicked")
	})
	btnSubtle := components.NewButtonVariant(text, th, "Subtle", components.VariantSubtle, func() {
		g.setStatus("Subtle clicked")
	})
	iconSettings := components.NewIconButton(th, sheet, "settings", th.Metrics.ControlHeight, func() {
		g.setStatus("Settings icon")
	})
	iconAdd := components.NewIconButton(th, sheet, "add", th.Metrics.ControlHeight, func() {
		g.setStatus("Add icon")
	})
	iconClose := components.NewIconButton(th, sheet, "close", th.Metrics.ControlHeight, func() {
		g.setStatus("Close icon")
	})
	sections.Children = append(sections.Children, sectionCard(text, th, "Buttons", "Button variants and icon buttons",
		row(th, btnPrimary.El, btnSecondary.El, btnSubtle.El),
		row(th, iconSettings.El, iconAdd.El, iconClose.El),
	))

	// 3. Form controls
	g.checkA = components.NewCheckbox(text, th, sheet, "Enable notifications")
	g.checkA.OnChange = func(c bool) { g.setStatus(fmt.Sprintf("notifications: %v", c)) }
	g.checkB = components.NewCheckbox(text, th, sheet, "Dark mode sync")
	g.checkB.Checked = true

	g.radioGrp = components.NewRadioGroup(th)
	g.radioGrp.OnChange = func(v int) { g.setStatus(fmt.Sprintf("radio: %d", v)) }
	r1 := g.radioGrp.Add(text, "Option A")
	r2 := g.radioGrp.Add(text, "Option B")
	r3 := g.radioGrp.Add(text, "Option C")
	g.radioGrp.Select(0)

	g.selectW = components.NewSelect(text, th, sheet, 200, []components.SelectOption{
		{Label: "Go", Value: "go"},
		{Label: "Rust", Value: "rust"},
		{Label: "TypeScript", Value: "ts"},
	})
	g.selectW.OnChange = func(v string) { g.setStatus("selected: " + v) }

	demoField := components.NewTextField(text, th, sheet, clip, components.TextFieldConfig{
		Placeholder: "Type here...",
		IconStart:   "edit",
	})

	g.tagEdit = components.NewTagEdit(text, th, sheet, clip, 400)
	g.tagEdit.Tags = []string{"ui", "yoga"}
	g.tagEdit.OnChange = func(tags []string) { g.setStatus(fmt.Sprintf("tags: %v", tags)) }

	sections.Children = append(sections.Children, sectionCard(text, th, "Form Controls", "Checkbox, radio, select, text field, tag edit",
		g.checkA.El, g.checkB.El,
		row(th, r1.El, r2.El, r3.El),
		row(th, g.selectW.El, demoField.El),
		g.tagEdit.El,
	))

	// 4. Navigation
	g.dropdown = components.NewDropdown(text, th, "Actions", 160, []components.MenuItem{
		{Label: "Copy", OnSelect: func() { g.setStatus("Copy") }},
		{Label: "Paste", OnSelect: func() { g.setStatus("Paste") }},
	})
	bc := components.NewBreadcrumb(text, th, sheet, []components.BreadcrumbSegment{
		{Label: "Home", OnSelect: func() { g.setStatus("Home") }},
		{Label: "Projects", OnSelect: func() { g.setStatus("Projects") }},
		{Label: "Yoga UI"},
	})
	g.navVert = components.NewNavigation(text, th, sheet, components.NavVertical, components.NavIconLeft)
	g.navVert.El.Style = g.navVert.El.Style.W(180)
	g.navVert.Add(components.NavItem{ID: "home", Label: "Home", Icon: "folder"})
	g.navVert.Add(components.NavItem{ID: "settings", Label: "Settings", Icon: "settings"})
	g.navVert.Add(components.NavItem{ID: "search", Label: "Search", Icon: "search"})
	g.navVert.Select(0)
	g.navVert.OnSelect = func(_ int, id string) { g.setStatus("nav: " + id) }

	g.navHoriz = components.NewNavigation(text, th, sheet, components.NavHorizontal, components.NavIconTop)
	g.navHoriz.Add(components.NavItem{ID: "new", Label: "New", Icon: "add"})
	g.navHoriz.Add(components.NavItem{ID: "save", Label: "Save", Icon: "save"})
	g.navHoriz.Add(components.NavItem{ID: "run", Label: "Run", Icon: "play_arrow"})
	g.navHoriz.Select(0)
	g.navHoriz.OnSelect = func(_ int, id string) { g.setStatus("toolbar: " + id) }

	sections.Children = append(sections.Children, sectionCard(text, th, "Navigation", "Breadcrumb, dropdown, and nav strips",
		bc.El,
		g.dropdown.Button.El,
		g.navVert.El,
		g.navHoriz.El,
	))

	// 5. Surfaces
	alertInfo := components.NewAlert(text, th, "This is an informational alert.", components.AlertInfo)
	alertWarn := components.NewAlert(text, th, "Warning: unsaved changes.", components.AlertWarning)
	alertErr := components.NewAlert(text, th, "Error: could not connect.", components.AlertError)
	alertOk := components.NewAlert(text, th, "Success: file saved.", components.AlertSuccess)
	bodyStyle := th.Typography.Body
	_, bodyLineH := text.MeasureAt("Card body content goes here.", bodyStyle.Size)
	cardBody := layout.New(layout.Box().H(bodyLineH+2*th.Spacing.S).PaddingAll(th.Spacing.S))
	cardBody.Paint = func(dl *render.DrawList, eng *shape.Engine) {
		eng.DrawStringTop(dl, "Card body content goes here.", cardBody.Frame.X+th.Spacing.S, cardBody.Frame.Y+th.Spacing.S, th.ForegroundMuted)
	}
	sampleCard := components.NewCard(text, th, "Sample Card", "A surfaced container", cardBody)
	sections.Children = append(sections.Children, sectionCard(text, th, "Surfaces", "Cards and alerts",
		sampleCard.El,
		alertInfo.El, alertWarn.El, alertErr.El, alertOk.El,
	))

	// 6. Feedback
	toastInfo := components.NewButton(text, th, "Show Info Toast", func() {
		toasts.Show("Info toast message", components.ToastInfo, 3*time.Second)
		g.setStatus("toast shown")
	})
	toastErr := components.NewButton(text, th, "Show Error Toast", func() {
		toasts.Show("Something went wrong", components.ToastError, 3*time.Second)
	})
	sections.Children = append(sections.Children, sectionCard(text, th, "Feedback", "Spinner and toasts",
		row(th, g.spinner.El, toastInfo.El, toastErr.El),
	))

	// 7. Dialogs
	errDlg := components.NewButton(text, th, "Show Error Dialog", func() {
		dialogs.ShowError("Error", "Something failed unexpectedly.", func() {
			g.setStatus("error dialog dismissed")
		})
	})
	inDlg := components.NewButton(text, th, "Show Input Dialog", func() {
		dialogs.ShowInput("Rename", "new-name", func(v string) {
			g.setStatus("input: " + v)
		}, func() {
			g.setStatus("input cancelled")
		})
	})
	sections.Children = append(sections.Children, sectionCard(text, th, "Dialogs", "Modal overlays",
		row(th, errDlg.El, inDlg.El),
	))

	g.statusEl = layout.New(layout.Box().H(28).PaddingXY(th.Spacing.M, 0))
	g.statusEl.Paint = func(dl *render.DrawList, eng *shape.Engine) {
		dl.AddRect(g.statusEl.Frame, th.ChromeMuted)
		eng.DrawStringTop(dl, g.status, g.statusEl.Frame.X+th.Spacing.M, g.statusEl.Frame.Y+6, th.ForegroundMuted)
	}
	sections.Children = append(sections.Children, g.statusEl)

	g.scroll = components.NewScrollView(th, sections)
	g.root = layout.New(layout.Box().Direction(layout.Column).FlexGrow(1), g.scroll.El)

	g.focus = components.NewFocusManager()
	g.focus.Add(demoField, g.checkA, g.checkB, r1, r2, r3, g.selectW, g.tagEdit, btnPrimary, btnSecondary)
	return g
}

func (g *ComponentGallery) overlayEls() []*layout.Element {
	return []*layout.Element{g.dropdown.Menu.El, g.selectW.MenuEl()}
}

func (g *ComponentGallery) setStatus(s string) { g.status = s }

func (g *ComponentGallery) update(m *input.Mouse, kb *input.Keyboard) {
	layout.Dispatch(g.root, m)
	g.scroll.Update(m)
	g.spinner.Update()
	g.tagEdit.Update(m)
	if g.focus != nil {
		g.focus.HandleMouse(m)
		if kb != nil {
			g.focus.Route(kb)
		}
	}
}

func (g *ComponentGallery) animationWait() (time.Duration, bool) {
	return g.spinner.AnimationWait()
}

func row(th *theme.Theme, items ...*layout.Element) *layout.Element {
	return layout.New(layout.Box().Direction(layout.Row).Gap(th.Spacing.S).AlignItems(layout.AlignCenter), items...)
}

func sectionColumn(th *theme.Theme, items ...*layout.Element) *layout.Element {
	return layout.New(layout.Box().Direction(layout.Column).Gap(th.Spacing.S), items...)
}

func sectionCard(text *shape.Engine, th *theme.Theme, title, subtitle string, body ...*layout.Element) *layout.Element {
	col := layout.New(layout.Box().Direction(layout.Column).Gap(th.Spacing.S), body...)
	card := components.NewCard(text, th, title, subtitle, col)
	return card.El
}
