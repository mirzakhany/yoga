package main

import (
	"fmt"
	"time"

	"github.com/mirzakhany/yoga"
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
	kvTable  *components.Table
	kvFilter *components.TextField
	dialogs  *components.DialogHost
	toasts   *components.ToastHost
	statusEl *layout.Element
}

func buildComponentGallery(dialogs *components.DialogHost, toasts *components.ToastHost) *ComponentGallery {
	th := theme.Current()
	g := &ComponentGallery{
		dialogs: dialogs, toasts: toasts, status: "Interact with the widgets above",
	}
	g.spinner = components.NewSpinner(24)

	sections := layout.New(layout.Box().Direction(layout.Column).Gap(th.Spacing.L).PaddingAll(th.Spacing.L))

	// 1. Labels
	sections.Children = append(sections.Children, galleryVStack(th,
		components.NewLabel("Labels & Typography", components.LabelSubtitle).El,
		galleryRow(th,
			components.NewLabel("Body label", components.LabelBody).El,
			components.NewLabel("Caption", components.LabelCaption).El,
			components.NewLabel("Muted", components.LabelMuted).El,
			components.NewLabel("Strong", components.LabelStrong).El,
			components.NewLabel("Title", components.LabelTitle).El,
		),
	))

	// 2. Buttons
	btnPrimary := components.NewButton("Primary").Primary().Action(func() { g.setStatus("Primary clicked") })
	btnSecondary := components.NewButton("Secondary").Action(func() { g.setStatus("Secondary clicked") })
	btnSubtle := components.NewButton("Subtle").Subtle().Action(func() { g.setStatus("Subtle clicked") })
	iconSettings := components.NewIconButton("settings", th.Metrics.ControlHeight, nil).Action(func() { g.setStatus("Settings icon") })
	iconAdd := components.NewIconButton("add", th.Metrics.ControlHeight, nil).Action(func() { g.setStatus("Add icon") })
	iconClose := components.NewIconButton("close", th.Metrics.ControlHeight, nil).Action(func() { g.setStatus("Close icon") })
	sections.Children = append(sections.Children, galleryCard(th, "Buttons", "Button variants and icon buttons",
		galleryRow(th, btnPrimary.El, btnSecondary.El, btnSubtle.El),
		galleryRow(th, iconSettings.El, iconAdd.El, iconClose.El),
	))

	// 3. Form controls
	g.checkA = components.NewCheckbox("Enable notifications").Changed(func(c bool) {
		g.setStatus(fmt.Sprintf("notifications: %v", c))
	})
	g.checkB = components.NewCheckbox("Dark mode sync").Check(true)

	g.radioGrp = components.NewRadioGroup().Changed(func(v int) {
		g.setStatus(fmt.Sprintf("radio: %d", v))
	})
	r1 := g.radioGrp.Add("Option A")
	r2 := g.radioGrp.Add("Option B")
	r3 := g.radioGrp.Add("Option C")
	g.radioGrp.Select(0)

	g.selectW = components.NewSelect(200, []components.SelectOption{
		{Label: "Go", Value: "go"},
		{Label: "Rust", Value: "rust"},
		{Label: "TypeScript", Value: "ts"},
	}).Changed(func(v string) { g.setStatus("selected: " + v) })

	demoField := components.NewTextField(components.TextFieldConfig{Placeholder: "Type here..."}).WithIconStart("edit")

	g.tagEdit = components.NewTagEdit(400)
	g.tagEdit.Tags = []string{"ui", "yoga"}
	g.tagEdit.OnChange = func(tags []string) { g.setStatus(fmt.Sprintf("tags: %v", tags)) }

	sections.Children = append(sections.Children, galleryCard(th, "Form Controls", "Checkbox, radio, select, text field, tag edit",
		g.checkA.El, g.checkB.El,
		galleryRow(th, r1.El, r2.El, r3.El),
		galleryRow(th, g.selectW.El, demoField.El),
		g.tagEdit.El,
	))

	// 4. Tables
	kvColumns := []components.TableColumn{
		{ID: "sel", Label: "", Kind: components.TableColCheckbox, Width: 36},
		{ID: "key", Label: "Key", Kind: components.TableColEditable, Width: 0, Sortable: true},
		{ID: "val", Label: "Value", Kind: components.TableColEditable, Width: 0, Sortable: true},
		{ID: "act", Label: "", Kind: components.TableColActions, Width: 40, Locked: true},
	}
	g.kvTable = components.NewTable(kvColumns, []components.TableAction{
		{Icon: "close", Tooltip: "Delete"},
	})
	g.kvTable.El.Style = g.kvTable.El.Style.H(220)
	g.kvFilter = components.NewTextField(components.TextFieldConfig{Placeholder: "Filter rows..."}).
		WithIconStart("search").
		Changed(g.kvTable.SetFilter)
	g.kvTable.SetRows([]components.TableRow{
		{ID: "h1", Cells: map[string]string{"key": "Content-Type", "val": "application/json"}},
		{ID: "h2", Cells: map[string]string{"key": "Authorization", "val": "Bearer token"}},
		{ID: "h3", Cells: map[string]string{"key": "Accept", "val": "text/html"}},
		{ID: "h4", Cells: map[string]string{"key": "User-Agent", "val": "yoga-example/1.0"}},
	})
	g.kvTable.OnCellChange = func(rowID, colID, value string) {
		g.setStatus(fmt.Sprintf("cell %s.%s = %q", rowID, colID, value))
	}
	g.kvTable.OnSelectionChange = func() {
		g.setStatus(fmt.Sprintf("selected: %v", g.kvTable.SelectedIDs()))
	}
	g.kvTable.OnSortChange = func(colID string, asc bool) {
		dir := "asc"
		if !asc {
			dir = "desc"
		}
		g.setStatus(fmt.Sprintf("sorted by %s (%s)", colID, dir))
	}
	g.kvTable.OnDelete = func(rowID string) { g.setStatus("deleted row " + rowID) }
	g.kvTable.Actions[0].OnClick = func(rowID string) {
		g.kvTable.RemoveRow(rowID)
		if g.kvTable.OnDelete != nil {
			g.kvTable.OnDelete(rowID)
		}
	}
	nextRowID := 5
	addRowBtn := components.NewButton("Add Row").Action(func() {
		id := fmt.Sprintf("h%d", nextRowID)
		nextRowID++
		g.kvTable.AddRow(components.TableRow{ID: id, Cells: map[string]string{"key": "", "val": ""}})
		g.setStatus("added row " + id)
	})
	sections.Children = append(sections.Children, galleryCard(th, "Tables", "Key-value editor with filter, sortable headers, resizable columns",
		g.kvFilter.El,
		g.kvTable.El,
		addRowBtn.El,
	))

	// 5. Navigation
	g.dropdown = components.NewDropdown("Actions", 160, []components.MenuItem{
		{Label: "Copy", OnSelect: func() { g.setStatus("Copy") }},
		{Label: "Paste", OnSelect: func() { g.setStatus("Paste") }},
	})
	bc := components.NewBreadcrumb([]components.BreadcrumbSegment{
		{Label: "Home", OnSelect: func() { g.setStatus("Home") }},
		{Label: "Projects", OnSelect: func() { g.setStatus("Projects") }},
		{Label: "Yoga UI"},
	})
	g.navVert = components.NewNavigation(components.NavVertical, components.NavIconLeft)
	g.navVert.El.Style = g.navVert.El.Style.W(180)
	g.navVert.Add(components.NavItem{ID: "home", Label: "Home", Icon: "folder"})
	g.navVert.Add(components.NavItem{ID: "settings", Label: "Settings", Icon: "settings"})
	g.navVert.Add(components.NavItem{ID: "search", Label: "Search", Icon: "search"})
	g.navVert.Select(0)
	g.navVert.OnSelect = func(_ int, id string) { g.setStatus("nav: " + id) }

	g.navHoriz = components.NewNavigation(components.NavHorizontal, components.NavIconTop)
	g.navHoriz.Add(components.NavItem{ID: "new", Label: "New", Icon: "add"})
	g.navHoriz.Add(components.NavItem{ID: "save", Label: "Save", Icon: "save"})
	g.navHoriz.Add(components.NavItem{ID: "run", Label: "Run", Icon: "play_arrow"})
	g.navHoriz.Select(0)
	g.navHoriz.OnSelect = func(_ int, id string) { g.setStatus("toolbar: " + id) }

	sections.Children = append(sections.Children, galleryCard(th, "Navigation", "Breadcrumb, dropdown, and nav strips",
		bc.El,
		g.dropdown.Button.El,
		g.navVert.El,
		g.navHoriz.El,
	))

	// 6. Surfaces
	sections.Children = append(sections.Children, galleryCard(th, "Surfaces", "Cards and alerts",
		components.NewCard("Sample Card", "A surfaced container", galleryCardBody(th)).El,
		components.NewCard("Elevated Card", "Stronger drop shadow", galleryCardBody(th)).Elevated().El,
		components.NewAlert("This is an informational alert.", components.AlertInfo).El,
		components.NewAlert("Warning: unsaved changes.", components.AlertWarning).El,
		components.NewAlert("Error: could not connect.", components.AlertError).El,
		components.NewAlert("Success: file saved.", components.AlertSuccess).El,
	))

	// 7. Feedback
	sections.Children = append(sections.Children, galleryCard(th, "Feedback", "Spinner and toasts",
		galleryRow(th,
			g.spinner.El,
			components.NewButton("Show Info Toast").Action(func() {
				toasts.Show("Info toast message", components.ToastInfo, 3*time.Second)
				g.setStatus("toast shown")
			}).El,
			components.NewButton("Show Error Toast").Action(func() {
				toasts.Show("Something went wrong", components.ToastError, 3*time.Second)
			}).El,
		),
	))

	// 8. Dialogs
	sections.Children = append(sections.Children, galleryCard(th, "Dialogs", "Modal overlays",
		galleryRow(th,
			components.NewButton("Show Error Dialog").Action(func() {
				dialogs.ShowError("Error", "Something failed unexpectedly.", func() {
					g.setStatus("error dialog dismissed")
				})
			}).El,
			components.NewButton("Show Input Dialog").Action(func() {
				dialogs.ShowInput("Rename", "new-name", func(v string) {
					g.setStatus("input: " + v)
				}, func() {
					g.setStatus("input cancelled")
				})
			}).El,
		),
	))

	g.statusEl = layout.New(layout.Box().H(28).PaddingXY(th.Spacing.M, 0))
	g.statusEl.Paint = func(dl *render.DrawList, eng *shape.Engine) {
		dl.AddRect(g.statusEl.Frame, th.ChromeMuted)
		eng.DrawStringTop(dl, g.status, g.statusEl.Frame.X+th.Spacing.M, g.statusEl.Frame.Y+6, th.ForegroundMuted)
	}
	sections.Children = append(sections.Children, g.statusEl)

	g.scroll = components.NewScrollView(sections)
	g.root = layout.New(layout.Box().Direction(layout.Column).FlexGrow(1), g.scroll.El)

	g.focus = components.NewFocusManager()
	g.focus.Add(demoField, g.checkA, g.checkB, r1, r2, r3, g.selectW, g.tagEdit, g.kvFilter, g.kvTable, btnPrimary, btnSecondary)
	return g
}

func (g *ComponentGallery) overlayEls() []*layout.Element {
	return []*layout.Element{g.dropdown.Menu.El, g.selectW.MenuEl(), g.kvTable.EditEl()}
}

func (g *ComponentGallery) setStatus(s string) { g.status = s }

func (g *ComponentGallery) update(m *input.Mouse, kb *input.Keyboard) {
	g.scroll.Update(m)
	g.spinner.Update()
	g.tagEdit.Update(m)
	g.kvTable.Update(m)
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

// ── Gallery layout helpers ────────────────────────────────────────────────────

// galleryRow builds a centered horizontal flex row with standard gap.
func galleryRow(th *theme.Theme, items ...*layout.Element) *layout.Element {
	el := layout.HStack(th.Spacing.S, items...)
	el.Style = el.Style.PaddingXY(3, 3)
	return el
}

// galleryVStack stacks items vertically with standard gap (no card wrapper).
func galleryVStack(th *theme.Theme, items ...*layout.Element) *layout.Element {
	return layout.VStack(th.Spacing.S, items...)
}

// galleryCard wraps body items in a card with a title and subtitle.
func galleryCard(th *theme.Theme, title, subtitle string, body ...*layout.Element) *layout.Element {
	col := layout.VStack(th.Spacing.S, body...)
	return components.NewCard(title, subtitle, col).El
}

// galleryCardBody returns a simple placeholder body element for card examples.
func galleryCardBody(th *theme.Theme) *layout.Element {
	style := th.Typography.Body
	_, lh := yoga.Text().MeasureAt("Card body content goes here.", style.Size)
	el := layout.New(layout.Box().H(lh + 2*th.Spacing.S).PaddingAll(th.Spacing.S))
	el.Paint = func(dl *render.DrawList, eng *shape.Engine) {
		eng.DrawStringTop(dl, "Card body content goes here.",
			el.Frame.X+th.Spacing.S, el.Frame.Y+th.Spacing.S, th.ForegroundMuted)
	}
	return el
}
