package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/mirzakhany/yoga/ui"
)

type ComponentGallery struct {
	status   string
	checkA   bool
	checkB   bool
	radio    int
	selectV  string
	tags     []string
	kvTable  *ui.Table
	kvFilter string
	dialogs  *ui.DialogHost
	files    *ui.FileDialog
	toasts   *ui.ToastHost
	demoText string
	navVert  int
	navHoriz int
}

func buildComponentGallery(dialogs *ui.DialogHost, files *ui.FileDialog, toasts *ui.ToastHost) *ComponentGallery {
	g := &ComponentGallery{
		dialogs: dialogs, files: files, toasts: toasts,
		status: "Interact with the widgets above", checkB: true,
		selectV: "go", tags: []string{"ui", "yoga"},
	}
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
	return g
}

func (g *ComponentGallery) setStatus(s string) { g.status = s }

func (g *ComponentGallery) Layout(c *ui.Ctx) ui.View {
	th := c.Theme()
	return ui.Scroll("gallery",
		ui.Column(
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
			ui.Row(
				ui.Radio("g-ra", "Option A").Check(g.radio == 0).OnClick(func() {
					g.radio = 0
					g.setStatus("radio: 0")
				}),
				ui.Radio("g-rb", "Option B").Check(g.radio == 1).OnClick(func() {
					g.radio = 1
					g.setStatus("radio: 1")
				}),
				ui.Radio("g-rc", "Option C").Check(g.radio == 2).OnClick(func() {
					g.radio = 2
					g.setStatus("radio: 2")
				}),
			).Gap(th.Spacing.S),
			ui.Select("g-lang", []ui.SelectOption{
				{Label: "Go", Value: "go"},
				{Label: "Rust", Value: "rust"},
				{Label: "TypeScript", Value: "ts"},
			}).Width(200).Selected(selectIndex(g.selectV, []string{"go", "rust", "ts"})).OnChange(func(v string) {
				g.selectV = v
				g.setStatus("selected: " + v)
			}),
			ui.TextField("g-demo", g.demoText).Placeholder("Type here...").IconStart("edit").OnChange(func(s string) { g.demoText = s }).Grow(1),
			ui.TagEdit("g-tags", g.tags).OnTags(func(tags []string) {
				g.tags = tags
				g.setStatus(fmt.Sprintf("tags: %v", tags))
			}).Width(400),
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
			ui.Dropdown("gallery-actions", "Actions", []ui.MenuItem{
				{Label: "Copy", OnSelect: func() { g.setStatus("Copy") }},
				{Label: "Paste", OnSelect: func() { g.setStatus("Paste") }},
			}),
			ui.Nav("g-nav-v", ui.NavVertical, ui.NavIconLeft,
				ui.NavItem{ID: "home", Label: "Home", Icon: "folder"},
				ui.NavItem{ID: "settings", Label: "Settings", Icon: "settings"},
			).Selected(g.navVert).OnSelectItem(func(i int, id string) {
				g.navVert = i
				g.setStatus("nav: " + id)
			}).Width(180),
			ui.Nav("g-nav-h", ui.NavHorizontal, ui.NavIconTop,
				ui.NavItem{ID: "new", Label: "New", Icon: "add"},
			).Selected(g.navHoriz).OnSelectItem(func(i int, _ string) { g.navHoriz = i }),
			ui.Subtitle("Feedback"),
			ui.Row(
				ui.Spinner("g-spin", 24),
				ui.Button("g-toast-info", ui.Text("Show Info Toast")).OnClick(func() {
					g.toasts.Show("Info toast message", ui.ToastInfo, 3*time.Second)
				}),
				ui.Button("g-toast-err", ui.Text("Show Error Toast")).OnClick(func() {
					g.toasts.Show("Something went wrong", ui.ToastError, 3*time.Second)
				}),
				ui.Button("g-dlg-err", ui.Text("Show Error Dialog")).OnClick(func() {
					g.dialogs.ShowError("Error", "Something failed unexpectedly.", func() {
						g.setStatus("error dialog dismissed")
					})
				}),
			).Gap(th.Spacing.S),
			ui.Row(
				ui.Button("g-fd-file", ui.Text("Open File")).OnClick(func() {
					g.files.Show(ui.FileDialogOpts{
						Mode: ui.FileDialogOpenFile,
						Filters: []ui.FileFilter{
							{Label: "Go files", Exts: []string{".go"}},
							{Label: "All files", Exts: nil},
						},
						OnConfirm: func(paths []string) {
							g.setStatus("opened: " + strings.Join(paths, ", "))
							g.toasts.Show("Opened "+strings.Join(paths, ", "), ui.ToastSuccess, 3*time.Second)
						},
					})
				}),
				ui.Button("g-fd-files", ui.Text("Open Files")).OnClick(func() {
					g.files.Show(ui.FileDialogOpts{
						Mode:     ui.FileDialogOpenFile,
						Multiple: true,
						Filters: []ui.FileFilter{
							{Label: "Text", Exts: []string{".txt", ".md", ".go"}},
							{Label: "All files", Exts: nil},
						},
						OnConfirm: func(paths []string) {
							g.setStatus("opened: " + strings.Join(paths, ", "))
							g.toasts.Show("Opened "+strings.Join(paths, ", "), ui.ToastSuccess, 3*time.Second)
						},
					})
				}),
				ui.Button("g-fd-folder", ui.Text("Select Folder")).OnClick(func() {
					g.files.Show(ui.FileDialogOpts{
						Mode: ui.FileDialogOpenFolder,
						OnConfirm: func(paths []string) {
							g.setStatus("folder: " + strings.Join(paths, ", "))
							g.toasts.Show("Folder "+strings.Join(paths, ", "), ui.ToastInfo, 3*time.Second)
						},
					})
				}),
				ui.Button("g-fd-folders", ui.Text("Select Folders")).OnClick(func() {
					g.files.Show(ui.FileDialogOpts{
						Mode:     ui.FileDialogOpenFolder,
						Multiple: true,
						OnConfirm: func(paths []string) {
							g.setStatus("folders: " + strings.Join(paths, ", "))
							g.toasts.Show("Folders "+strings.Join(paths, ", "), ui.ToastInfo, 3*time.Second)
						},
					})
				}),
			).Gap(th.Spacing.S),
			ui.Text(g.status).Style(ui.Spec{}.TextColor(ui.TokenForegroundMuted)),
		).Gap(th.Spacing.M).Padding(th.Spacing.L),
	).Grow(1)
}

func selectIndex(v string, values []string) int {
	for i, s := range values {
		if s == v {
			return i
		}
	}
	return 0
}
