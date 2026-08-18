package main

import (
	"fmt"
	"time"

	"github.com/mirzakhany/yoga/theme"
	"github.com/mirzakhany/yoga/ui"
)

type ComponentGallery struct {
	status   string
	spinner  *ui.Spinner
	checkA   bool
	checkB   bool
	radioGrp *ui.RadioGroup
	radios   []*ui.Radio
	selectW  *ui.Select
	tagEdit  *ui.TagEdit
	dropdown *ui.Dropdown
	navVert  *ui.Navigation
	navHoriz *ui.Navigation
	kvTable  *ui.Table
	kvFilter string
	dialogs  *ui.DialogHost
	toasts   *ui.ToastHost
	demoText string
}

func buildComponentGallery(dialogs *ui.DialogHost, toasts *ui.ToastHost) *ComponentGallery {
	th := theme.Current()
	g := &ComponentGallery{dialogs: dialogs, toasts: toasts, status: "Interact with the widgets above", checkB: true, demoText: ""}
	g.spinner = ui.NewSpinner(24)
	g.radioGrp = ui.NewRadioGroup()
	g.radios = []*ui.Radio{
		g.radioGrp.Add("Option A"),
		g.radioGrp.Add("Option B"),
		g.radioGrp.Add("Option C"),
	}
	g.radioGrp.Select(0)
	g.radioGrp.OnChange = func(v int) { g.setStatus(fmt.Sprintf("radio: %d", v)) }

	g.selectW = ui.NewSelect(200, []ui.SelectOption{
		{Label: "Go", Value: "go"},
		{Label: "Rust", Value: "rust"},
		{Label: "TypeScript", Value: "ts"},
	}).Changed(func(v string) { g.setStatus("selected: " + v) })

	g.tagEdit = ui.NewTagEdit(400, "ui", "yoga")
	g.tagEdit.OnChange = func(tags []string) { g.setStatus(fmt.Sprintf("tags: %v", tags)) }

	g.kvTable = ui.NewTable([]ui.TableColumn{
		{ID: "sel", Label: "", Kind: ui.TableColCheckbox, Width: 36},
		{ID: "key", Label: "Key", Kind: ui.TableColEditable, Width: 0, Sortable: true},
		{ID: "val", Label: "Value", Kind: ui.TableColEditable, Width: 0, Sortable: true},
		{ID: "act", Label: "", Kind: ui.TableColActions, Width: 40, Locked: true},
	}, []ui.TableAction{{Icon: "delete", Tooltip: "Delete"}})
	g.kvTable.SetRows([]ui.TableRow{
		{ID: "h1", Cells: map[string]string{"key": "Content-Type", "val": "application/json"}},
		{ID: "h2", Cells: map[string]string{"key": "Authorization", "val": "Bearer token"}},
	})
	g.kvTable.Actions[0].OnClick = func(rowID string) { g.kvTable.RemoveRow(rowID) }

	g.dropdown = ui.NewDropdown("gallery-actions", "Actions", 160, []ui.MenuItem{
		{Label: "Copy", OnSelect: func() { g.setStatus("Copy") }},
		{Label: "Paste", OnSelect: func() { g.setStatus("Paste") }},
	})
	g.navVert = ui.NewNavigation(ui.NavVertical, ui.NavIconLeft)
	g.navVert.Add(ui.NavItem{ID: "home", Label: "Home", Icon: "folder"})
	g.navVert.Add(ui.NavItem{ID: "settings", Label: "Settings", Icon: "settings"})
	g.navVert.Select(0)
	g.navVert.OnSelect = func(_ int, id string) { g.setStatus("nav: " + id) }

	g.navHoriz = ui.NewNavigation(ui.NavHorizontal, ui.NavIconTop)
	g.navHoriz.Add(ui.NavItem{ID: "new", Label: "New", Icon: "add"})
	g.navHoriz.Select(0)
	_ = th
	return g
}

func (g *ComponentGallery) setStatus(s string) { g.status = s }

func (g *ComponentGallery) Layout(c *ui.Ctx) ui.View {
	th := c.Theme()

	radioViews := make([]ui.View, len(g.radios))
	for i, r := range g.radios {
		radioViews[i] = r
	}

	return ui.Column(
		ui.Subtitle("Labels & Typography"),
		ui.Row(ui.Text("Body"), ui.Caption("Caption"), ui.Muted("Muted"), ui.Strong("Strong"), ui.Title("Title")).Gap(th.Spacing.S),
		ui.Subtitle("Buttons"),
		ui.Row(
			ui.Button("g-primary", ui.Text("Primary")).Primary().OnClick(func() { g.setStatus("Primary clicked") }),
			ui.Button("g-secondary", ui.Text("Secondary")).OnClick(func() { g.setStatus("Secondary clicked") }),
			ui.Button("g-subtle", ui.Text("Subtle")).Subtle().OnClick(func() { g.setStatus("Subtle clicked") }),
			ui.IconButton("g-settings", "settings").OnClick(func() { g.setStatus("Settings icon") }),
		).Gap(th.Spacing.S),
		ui.Subtitle("Form Controls"),
		ui.Checkbox("g-check-a", "Enable notifications").Check(g.checkA).OnToggle(func(v bool) {
			g.checkA = v
			g.setStatus(fmt.Sprintf("notifications: %v", v))
		}),
		ui.Checkbox("g-check-b", "Dark mode sync").Check(g.checkB).OnToggle(func(v bool) { g.checkB = v }),
		ui.Row(radioViews...).Gap(th.Spacing.S),
		ui.ViewOf(g.selectW),
		ui.TextField("g-demo", g.demoText).Placeholder("Type here...").IconStart("edit").OnChange(func(s string) { g.demoText = s }).Grow(1),
		ui.ViewOf(g.tagEdit),
		ui.Subtitle("Tables"),
		ui.TextField("g-filter", g.kvFilter).Placeholder("Filter rows...").IconStart("search").OnChange(func(s string) {
			g.kvFilter = s
			g.kvTable.SetFilter(s)
		}),
		ui.ViewOf(g.kvTable).Height(220),
		ui.Button("g-add-row", ui.Text("Add Row")).OnClick(func() {
			id := fmt.Sprintf("h%d", time.Now().UnixNano())
			g.kvTable.AddRow(ui.TableRow{ID: id, Cells: map[string]string{"key": "", "val": ""}})
		}),
		ui.Subtitle("Navigation"),
		ui.ViewOf(g.dropdown),
		ui.ViewOf(g.navVert).Width(180),
		ui.ViewOf(g.navHoriz),
		ui.Subtitle("Feedback"),
		ui.Row(
			ui.ViewOf(g.spinner),
			ui.Button("g-toast-info", ui.Text("Show Info Toast")).OnClick(func() {
				g.toasts.Show("Info toast message", ui.ToastInfo, 3*time.Second)
			}),
			ui.Button("g-toast-err", ui.Text("Show Error Toast")).OnClick(func() {
				g.toasts.Show("Something went wrong", ui.ToastError, 3*time.Second)
			}),
			ui.Button("g-dlg-err", ui.Text("Show Error Dialog")).OnClick(func() {
				g.dialogs.ShowError("Error", "Something failed unexpectedly.", func() { g.setStatus("error dialog dismissed") })
			}),
		).Gap(th.Spacing.S),
		ui.Text(g.status).Style(ui.Spec{}.TextColor(ui.TokenForegroundMuted)),
	).Gap(th.Spacing.M).Padding(th.Spacing.L).Grow(1)
}
