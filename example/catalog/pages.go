package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/theme"
	"github.com/mirzakhany/yoga/ui"
)

func (app *CatalogApp) pageShell(c *ui.Ctx, title string, sections ...ui.View) ui.View {
	th := c.Theme()
	kids := []ui.View{ui.Title(title)}
	for i, sec := range sections {
		if i > 0 {
			kids = append(kids, ui.HLine(th.Stroke.Thin, th.Border))
		}
		kids = append(kids, sec)
	}
	if app.status != "" {
		kids = append(kids, ui.HLine(th.Stroke.Thin, th.Border))
		kids = append(kids, ui.Text(app.status).Style(ui.Spec{}.TextColor(ui.TokenForegroundMuted)))
	}
	return ui.Column(kids...).Gap(th.Spacing.L).Padding(th.Spacing.L)
}

func (app *CatalogApp) section(title string, body ui.View) ui.View {
	th := theme.Current()
	return ui.Column(
		ui.Subtitle(title),
		body,
	).Gap(th.Spacing.M)
}

// --- Content pages ---

func (app *CatalogApp) pageTypography(c *ui.Ctx) ui.View {
	th := c.Theme()
	return app.pageShell(c, "Typography",
		app.section("Text styles", ui.Column(
			ui.Text("Body text — the default reading size for paragraphs and labels."),
			ui.Title("Title"),
			ui.Subtitle("Subtitle"),
			ui.Caption("Caption — small supporting text"),
			ui.Strong("Strong — semibold emphasis"),
			ui.Muted("Muted — de-emphasized secondary text"),
		).Gap(th.Spacing.S)),
		app.section("Inline row", ui.Row(
			ui.Text("Body"),
			ui.Caption("Caption"),
			ui.Muted("Muted"),
			ui.Strong("Strong"),
		).Gap(th.Spacing.M).Wrap()),
	)
}

func (app *CatalogApp) pageSurfaces(c *ui.Ctx) ui.View {
	th := c.Theme()
	return app.pageShell(c, "Surfaces",
		app.section("Cards", ui.Row(
			ui.Card("Flat card", "No shadow", ui.Text("Card body content")).Flat().Width(200),
			ui.Card("Raised card", "Default elevation", ui.Text("Card body content")).Width(200),
			ui.Card("Elevated card", "Medium shadow", ui.Text("Card body content")).Elevated().Width(200),
		).Gap(th.Spacing.M).Wrap()),
		app.section("Alerts", ui.Column(
			ui.Alert("This is an informational alert.", ui.AlertInfo),
			ui.Alert("Warning — check your input before continuing.", ui.AlertWarning),
			ui.Alert("Error — something went wrong.", ui.AlertError),
			ui.Alert("Success — operation completed.", ui.AlertSuccess),
			ui.Alert("Dismissable alert — click to close.", ui.AlertInfo).
				Dismissable(func() {
					app.alertDismissed = true
					app.setStatus("alert dismissed")
				}),
		).Gap(th.Spacing.S)),
	)
}

func (app *CatalogApp) pageIcons(c *ui.Ctx) ui.View {
	th := c.Theme()
	var cells []ui.View
	for _, name := range render.IconNames() {
		n := name
		cells = append(cells, ui.Column(
			ui.Icon(n, th.Metrics.IconSizeMD, th.Foreground),
			ui.Caption(n),
		).Gap(th.Spacing.XS).Align(ui.AlignCenter).Width(72))
	}
	return app.pageShell(c, "Icons",
		app.section("Icon atlas", ui.Text("Named sprites from render/assets/icons/*.svg")),
		ui.Grid(6, cells...).Gap(th.Spacing.M),
	)
}

// --- Actions pages ---

func (app *CatalogApp) pageButtons(c *ui.Ctx) ui.View {
	th := c.Theme()
	return app.pageShell(c, "Buttons",
		app.section("Button variants", ui.Row(
			ui.Button("btn-primary", ui.Text("Primary")).Primary().OnClick(func() { app.setStatus("Primary clicked") }),
			ui.Button("btn-secondary", ui.Text("Secondary")).OnClick(func() { app.setStatus("Secondary clicked") }),
			ui.Button("btn-subtle", ui.Text("Subtle")).Subtle().OnClick(func() { app.setStatus("Subtle clicked") }),
		).Gap(th.Spacing.S)),
		app.section("Icon & hint", ui.Row(
			ui.Button("btn-icon", ui.Text("Save")).IconStart("save").Primary().Hint("⌘S").OnClick(func() { app.setStatus("Save clicked") }),
			ui.IconButton("btn-ib", "settings").OnClick(func() { app.setStatus("Settings icon clicked") }),
			ui.IconButton("btn-add", "add").OnClick(func() { app.setStatus("Add icon clicked") }),
		).Gap(th.Spacing.S)),
		app.section("States", ui.Row(
			ui.Button("btn-disabled", ui.Text("Disabled")).Primary().Disabled(true),
			ui.Button("btn-loading", ui.Text("Loading")).Primary().IconStart("refresh").OnClick(func() {}),
		).Gap(th.Spacing.S)),
	)
}

func (app *CatalogApp) pageSegmented(c *ui.Ctx) ui.View {
	th := c.Theme()
	return app.pageShell(c, "Segmented",
		app.section("Text segments", ui.Segmented("seg-text",
			ui.SegmentItem{Label: "Left", Value: "left"},
			ui.SegmentItem{Label: "Center", Value: "center"},
			ui.SegmentItem{Label: "Right", Value: "right"},
		).Selected(app.btnSeg).OnSelectItem(func(i int, v string) {
			app.btnSeg = i
			app.setStatus("segment: " + v)
		})),
		app.section("Icon segments", ui.Segmented("seg-icon",
			ui.SegmentItem{Icon: "split_horizontal", Value: "h"},
			ui.SegmentItem{Icon: "split_vertical", Value: "v"},
			ui.SegmentItem{Icon: "list", Value: "list"},
		).Selected(0).OnChange(func(v string) { app.setStatus("layout: " + v) })),
		app.section("Toolbar style", ui.Row(
			ui.Segmented("seg-fmt",
				ui.SegmentItem{Label: "B", Value: "bold"},
				ui.SegmentItem{Label: "I", Value: "italic"},
			).Selected(0).OnChange(func(v string) { app.setStatus("format: " + v) }),
			ui.Segmented("seg-align",
				ui.SegmentItem{Icon: "menu", Value: "left"},
				ui.SegmentItem{Icon: "more_horiz", Value: "center"},
				ui.SegmentItem{Icon: "more_vert", Value: "right"},
			).Selected(1).OnChange(func(v string) { app.setStatus("align: " + v) }),
		).Gap(th.Spacing.M)),
	)
}

// --- Forms pages ---

func (app *CatalogApp) pageTextFields(c *ui.Ctx) ui.View {
	return app.pageShell(c, "Text fields",
		app.section("Plain", ui.TextField("tf-plain", app.textPlain).
			Placeholder("Type here…").
			OnChange(func(s string) { app.textPlain = s }).
			Grow(1)),
		app.section("With icon", ui.TextField("tf-search", app.textSearch).
			Placeholder("Search…").
			IconStart("search").
			OnChange(func(s string) { app.textSearch = s }).
			Grow(1)),
		app.section("Password", ui.TextField("tf-pass", app.textPassword).
			Placeholder("Enter password").
			Password(true).
			IconStart("lock").
			OnChange(func(s string) { app.textPassword = s }).
			Grow(1)),
	)
}

func (app *CatalogApp) pageSelection(c *ui.Ctx) ui.View {
	th := c.Theme()
	return app.pageShell(c, "Selection controls",
		app.section("Checkbox", ui.Column(
			ui.Checkbox("chk-a", "Enable notifications").Check(app.checkA).OnToggle(func(v bool) {
				app.checkA = v
				app.setStatus(fmt.Sprintf("notifications: %v", v))
			}),
			ui.Checkbox("chk-b", "Dark mode sync").Check(app.checkB).OnToggle(func(v bool) {
				app.checkB = v
				app.setStatus(fmt.Sprintf("dark sync: %v", v))
			}),
		).Gap(th.Spacing.S)),
		app.section("Radio", ui.Row(
			ui.Radio("rad-a", "Option A").Check(app.radio == 0).OnClick(func() {
				app.radio = 0
				app.setStatus("radio: A")
			}),
			ui.Radio("rad-b", "Option B").Check(app.radio == 1).OnClick(func() {
				app.radio = 1
				app.setStatus("radio: B")
			}),
			ui.Radio("rad-c", "Option C").Check(app.radio == 2).OnClick(func() {
				app.radio = 2
				app.setStatus("radio: C")
			}),
		).Gap(th.Spacing.M)),
		app.section("Switch", ui.Row(
			ui.Text("Airplane mode"),
			ui.Switch("sw-air").Check(app.switchOn).OnToggle(func(v bool) {
				app.switchOn = v
				app.setStatus(fmt.Sprintf("switch: %v", v))
			}),
		).Gap(th.Spacing.M)),
	)
}

func (app *CatalogApp) pageChoice(c *ui.Ctx) ui.View {
	return app.pageShell(c, "Choice fields",
		app.section("Select", ui.Select("sel-lang", []ui.SelectOption{
			{Label: "Go", Value: "go"},
			{Label: "Rust", Value: "rust"},
			{Label: "TypeScript", Value: "ts"},
		}).Width(240).Selected(selectIndex(app.selectV, []string{"go", "rust", "ts"})).OnChange(func(v string) {
			app.selectV = v
			app.setStatus("selected: " + v)
		})),
		app.section("Tag edit", ui.TagEdit("tags", app.tags).OnTags(func(tags []string) {
			app.tags = tags
			app.setStatus("tags: " + strings.Join(tags, ", "))
		}).Width(400)),
	)
}

func (app *CatalogApp) pageForm(c *ui.Ctx) ui.View {
	themes := make([]ui.SelectOption, 0, len(theme.Names()))
	for _, name := range theme.Names() {
		n := name
		themes = append(themes, ui.SelectOption{Label: n, Value: n})
	}
	return app.pageShell(c, "Form rows",
		ui.Form("form-settings",
			ui.FormSwitch("f-notify", "Notifications", "Show system alerts and toasts", app.formNotify, func(v bool) {
				app.formNotify = v
				app.setStatus(fmt.Sprintf("notify: %v", v))
			}),
			ui.FormSelect("f-theme", "Theme", "Application color scheme", themes,
				selectIndex(app.formTheme, themeNames()),
				func(v string) {
					theme.Use(v)
					app.formTheme = v
					app.theme = v
					app.setStatus("theme: " + v)
				}),
			ui.FormNumber("f-size", "Font size", "Editor font size in points", app.formSize, 10, 24, 1, func(v float64) {
				app.formSize = v
				app.setStatus(fmt.Sprintf("font size: %.0f", v))
			}),
			ui.FormText("f-file", "Default file", "Open this file on startup", app.formFile, func(v string) {
				app.formFile = v
				app.setStatus("default file: " + v)
			}),
		),
	)
}

// --- Navigation page ---

func (app *CatalogApp) pageNavigation(c *ui.Ctx) ui.View {
	th := c.Theme()
	leftPane := ui.Column(
		ui.Strong(app.splitA),
		ui.Muted("Drag the handle to resize"),
	).Gap(th.Spacing.S).Padding(th.Spacing.M).Grow(1).Background(ui.TokenChromeMuted)
	rightPane := ui.Column(
		ui.Strong(app.splitB),
		ui.Text("Flex pane fills remaining space."),
	).Gap(th.Spacing.S).Padding(th.Spacing.M).Grow(1).Background(ui.TokenChromeMuted)

	crumbs := []ui.BreadcrumbSegment{
		{Label: "Home", OnSelect: func() { app.crumb = 0; app.setStatus("crumb: Home") }},
		{Label: "Projects", OnSelect: func() { app.crumb = 1; app.setStatus("crumb: Projects") }},
		{Label: "Yoga UI"},
	}

	return app.pageShell(c, "Navigation",
		app.section("Vertical nav", ui.Nav("nav-v", ui.NavVertical, ui.NavIconLeft,
			ui.NavItem{ID: "home", Label: "Home", Icon: "home"},
			ui.NavItem{ID: "code", Label: "Code", Icon: "code"},
			ui.NavItem{ID: "settings", Label: "Settings", Icon: "settings"},
		).Selected(app.navVert).OnSelectItem(func(i int, id string) {
			app.navVert = i
			app.setStatus("nav: " + id)
		}).Width(200)),
		app.section("Horizontal nav", ui.Nav("nav-h", ui.NavHorizontal, ui.NavIconTop,
			ui.NavItem{ID: "new", Label: "New", Icon: "add"},
			ui.NavItem{ID: "open", Label: "Open", Icon: "folder_open"},
			ui.NavItem{ID: "save", Label: "Save", Icon: "save"},
		).Selected(app.navHoriz).OnSelectItem(func(i int, id string) {
			app.navHoriz = i
			app.setStatus("nav: " + id)
		})),
		app.section("Tabs", ui.Tabs("nav-tabs", []ui.TabModel{
			{Title: "main.go", Modified: true},
			{Title: "app.go"},
			{Title: "README.md", Badge: "2"},
		}).Selected(app.tabIdx).OnSelectItem(func(i int, _ string) {
			app.tabIdx = i
			app.setStatus(fmt.Sprintf("tab: %d", i))
		}).OnTabClose(func(i int) { app.setStatus(fmt.Sprintf("closed tab: %d", i)) })),
		app.section("Menus", ui.Row(
			ui.Dropdown("nav-dd", "Actions", []ui.MenuItem{
				{Label: "Copy", OnSelect: func() { app.setStatus("Copy") }},
				{Label: "Paste", OnSelect: func() { app.setStatus("Paste") }},
				{Label: "Delete", OnSelect: func() { app.setStatus("Delete") }},
			}),
		).Gap(th.Spacing.S)),
		app.section("Breadcrumb", ui.Breadcrumb("nav-crumb", crumbs...)),
		app.section("Splitter", ui.ViewOf(
			ui.Splitter("nav-split", ui.Horizontal, leftPane, rightPane).Sizes(200, 0),
		).Height(160)),
	)
}

// --- Data pages ---

func (app *CatalogApp) pageTable(c *ui.Ctx) ui.View {
	return app.pageShell(c, "Table",
		app.section("Editable table", ui.Column(
			ui.Row(
				ui.TextField("tbl-filter", app.kvFilter).
					Placeholder("Filter rows…").
					IconStart("search").
					OnChange(func(s string) {
						app.kvFilter = s
						app.kvTable.SetFilter(s)
					}),
				ui.Spacer(),
				ui.Button("tbl-add", ui.Text("Add Row")).OnClick(func() {
					id := fmt.Sprintf("r%d", time.Now().UnixNano())
					app.kvTable.AddRow(ui.TableRow{ID: id, Cells: map[string]string{"key": "", "val": ""}})
					app.setStatus("added row")
				}),
			).Gap(c.Theme().Spacing.S),
			ui.ViewOf(app.kvTable).Height(220),
		).Gap(c.Theme().Spacing.S)),
	)
}

func (app *CatalogApp) pageTree(c *ui.Ctx) ui.View {
	return app.pageShell(c, "Tree",
		app.section("File tree", ui.ViewOf(app.demoTree).Height(280).Grow(1)),
	)
}

// --- Feedback page ---

func (app *CatalogApp) pageFeedback(c *ui.Ctx) ui.View {
	th := c.Theme()
	return app.pageShell(c, "Feedback",
		app.section("Spinner", ui.Row(
			ui.Spinner("spin", 24),
			ui.Text("Loading indicator"),
		).Gap(th.Spacing.M)),
		app.section("Toasts", ui.Row(
			ui.Button("toast-info", ui.Text("Info")).OnClick(func() {
				c.Toasts().Show("Info toast message", ui.ToastInfo, 3*time.Second)
			}),
			ui.Button("toast-ok", ui.Text("Success")).OnClick(func() {
				c.Toasts().Show("Operation succeeded", ui.ToastSuccess, 3*time.Second)
			}),
			ui.Button("toast-warn", ui.Text("Warning")).OnClick(func() {
				c.Toasts().Show("Warning toast", ui.ToastWarning, 3*time.Second)
			}),
			ui.Button("toast-err", ui.Text("Error")).OnClick(func() {
				c.Toasts().Show("Error toast", ui.ToastError, 3*time.Second)
			}),
		).Gap(th.Spacing.S).Wrap()),
		app.section("Dialogs", ui.Row(
			ui.Button("dlg-info", ui.Text("Info dialog")).OnClick(func() {
				c.Dialogs().ShowInfo("Info", "Here is some helpful information.", func() {
					app.setStatus("info dialog dismissed")
				})
			}),
			ui.Button("dlg-warn", ui.Text("Warning dialog")).OnClick(func() {
				c.Dialogs().ShowWarning("Warning", "Check your input before continuing.", func() {
					app.setStatus("warning dialog dismissed")
				})
			}),
			ui.Button("dlg-err", ui.Text("Error dialog")).OnClick(func() {
				c.Dialogs().ShowError("Error", "Something failed unexpectedly.", func() {
					app.setStatus("error dialog dismissed")
				})
			}),
			ui.Button("dlg-action", ui.Text("Action dialog")).OnClick(func() {
				c.Dialogs().ShowAction("Delete file?", "This cannot be undone.",
					func() { app.setStatus("action: yes") },
					func() { app.setStatus("action: no") },
				)
			}),
			ui.Button("btn-dlg-input", ui.Text("Input dialog")).OnClick(func() {
				c.Dialogs().ShowInput("Rename", "New name…", func(v string) {
					app.setStatus("renamed to: " + v)
				}, nil)
			}),
			ui.Button("dlg-custom", ui.Text("Custom dialog")).OnClick(func() {
				app.showCustomDialog(c)
			}),
		).Gap(th.Spacing.S).Wrap()),
		app.section("File picker", ui.Row(
			ui.Button("fd-open", ui.Text("Open file")).OnClick(func() { app.showOpenFile(c) }),
			ui.Button("fd-folder", ui.Text("Select folder")).OnClick(func() { app.showFolder(c) }),
			ui.Button("fd-save", ui.Text("Save file")).OnClick(func() { app.showSaveFile(c) }),
		).Gap(th.Spacing.S).Wrap()),
	)
}

// --- Dialog helpers ---

func (app *CatalogApp) showOpenFile(c *ui.Ctx) {
	c.Files().Show(ui.FileDialogOpts{
		Mode: ui.FileDialogOpenFile,
		Filters: []ui.FileFilter{
			{Label: "Go files", Exts: []string{".go"}},
			{Label: "All files", Exts: nil},
		},
		OnConfirm: func(paths []string) {
			app.setStatus("opened: " + strings.Join(paths, ", "))
			c.Toasts().Show("Opened "+strings.Join(paths, ", "), ui.ToastSuccess, 3*time.Second)
		},
	})
}

func (app *CatalogApp) showSaveFile(c *ui.Ctx) {
	c.Files().Show(ui.FileDialogOpts{
		Mode:              ui.FileDialogSaveFile,
		AllowCreateFolder: true,
		OnConfirm: func(paths []string) {
			app.setStatus("save target: " + strings.Join(paths, ", "))
		},
	})
}

func (app *CatalogApp) showFolder(c *ui.Ctx) {
	c.Files().Show(ui.FileDialogOpts{
		Mode: ui.FileDialogOpenFolder,
		OnConfirm: func(paths []string) {
			app.setStatus("folder: " + strings.Join(paths, ", "))
		},
	})
}

func (app *CatalogApp) showCustomDialog(c *ui.Ctx) {
	c.Dialogs().Show(ui.DialogOpts{
		Title:  "Custom dialog",
		Width:  480,
		Height: 280,
		Body: func(c *ui.Ctx) ui.View {
			th := c.Theme()
			return ui.Column(
				ui.Text("This is a custom dialog body built with the Yoga UI DSL."),
				ui.Alert("Dialogs are window services — call c.Dialogs().Show from OnClick.", ui.AlertInfo),
			).Gap(th.Spacing.M).Padding(th.Spacing.M).Grow(1)
		},
		Actions: []ui.DialogAction{
			{Label: "Cancel", OnClick: func() { app.setStatus("dialog cancelled") }},
			{Label: "OK", Primary: true, OnClick: func() { app.setStatus("dialog confirmed") }},
		},
	})
}

func selectIndex(v string, values []string) int {
	for i, s := range values {
		if s == v {
			return i
		}
	}
	return 0
}

func themeNames() []string {
	return theme.Names()
}
