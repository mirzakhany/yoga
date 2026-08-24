package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/mirzakhany/yoga/icons"
	"github.com/mirzakhany/yoga/icons/catalog"
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
	openIDs := []string{}
	if app.accordionOpen != "" {
		openIDs = []string{app.accordionOpen}
	}
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
		app.section("Badge", ui.Row(
			ui.Badge("3").Tone(ui.BadgeMuted),
			ui.Badge("New").Tone(ui.BadgeAccent),
			ui.Badge("OK").Tone(ui.BadgeSuccess),
			ui.Badge("Warn").Tone(ui.BadgeWarning),
			ui.Badge("Err").Tone(ui.BadgeError),
		).Gap(th.Spacing.S)),
		app.section("Kbd & Link", ui.Row(
			ui.Text("Press"),
			ui.Kbd("⌘S"),
			ui.Text("to save, or open"),
			ui.Link("link-docs", "the docs").OnClick(func() { app.setStatus("link clicked") }),
		).Gap(th.Spacing.S)),
		app.section("Accordion", ui.Accordion("acc",
			ui.AccordionItem{ID: "a", Title: "Getting started", Body: ui.Muted("Install Yoga and run the catalog demo.")},
			ui.AccordionItem{ID: "b", Title: "Theming", Body: ui.Muted("Call theme.Use to switch palettes at runtime.")},
			ui.AccordionItem{ID: "c", Title: "Overlays", Body: ui.Muted("Tooltip, Popover, and ContextMenu share placeAnchor.")},
		).OpenIDs(openIDs...).Exclusive().OnAccordionToggle(func(id string, open bool) {
			if open {
				app.accordionOpen = id
			} else if app.accordionOpen == id {
				app.accordionOpen = ""
			}
			app.setStatus(fmt.Sprintf("accordion %s open=%v", id, open))
		})),
	)
}

func (app *CatalogApp) pageIcons(c *ui.Ctx) ui.View {
	th := c.Theme()
	query := strings.ToLower(strings.TrimSpace(app.iconSearch))
	const maxIcons = 120

	matched := make([]icons.Icon, 0, maxIcons)
	for _, ic := range catalog.All {
		if len(matched) >= maxIcons {
			break
		}
		if query == "" || strings.Contains(strings.ToLower(ic.Name), query) {
			matched = append(matched, ic)
		}
	}

	var cells []ui.View
	for _, ic := range matched {
		icon := ic
		cells = append(cells, ui.Column(
			ui.Icon(icon, th.Metrics.IconSizeMD, th.Foreground),
			ui.Caption(icon.Name),
		).Gap(th.Spacing.XS).Align(ui.AlignCenter).Width(72))
	}

	sections := []ui.View{
		app.section("Search", ui.TextField("icon-search", app.iconSearch).
			Placeholder("Filter icons…").
			IconStart(icons.Search).
			OnChange(func(s string) { app.iconSearch = s }).
			Grow(1)),
	}
	if len(matched) == 0 {
		sections = append(sections, app.section("Results", ui.Muted("No icons match.")))
	} else {
		label := fmt.Sprintf("%d icons", len(matched))
		if len(matched) == maxIcons {
			label = fmt.Sprintf("First %d matches", maxIcons)
		}
		sections = append(sections, app.section(label, ui.Grid(6, cells...).Gap(th.Spacing.M)))
	}
	return app.pageShell(c, "Icons", sections...)
}

func (app *CatalogApp) pageImages(c *ui.Ctx) ui.View {
	th := c.Theme()
	return app.pageShell(c, "Images",
		app.section("From bytes", ui.Row(
			ui.Image("img-checker", checkerPNG).Width(96),
			ui.Column(
				ui.Text("PNG from in-memory bytes"),
				ui.Caption("Checkerboard generated at startup"),
			).Gap(th.Spacing.XS).Align(ui.AlignStart),
		).Gap(th.Spacing.L).Align(ui.AlignCenter)),
		app.section("From embed.FS", ui.Row(
			ui.ImageFS("img-embed", imageAssets, "testdata/sample.png").Width(96),
			ui.Column(
				ui.Text("PNG via embed.FS"),
				ui.Caption("example/catalog/testdata/sample.png"),
			).Gap(th.Spacing.XS).Align(ui.AlignStart),
		).Gap(th.Spacing.L).Align(ui.AlignCenter)),
		app.section("Fit modes", ui.Row(
			ui.Column(
				ui.Caption("Contain"),
				ui.Image("img-contain", samplePNG).Frame(120, 80).Fit(ui.FitContain),
			).Gap(th.Spacing.S).Align(ui.AlignCenter),
			ui.Column(
				ui.Caption("Cover"),
				ui.Image("img-cover", samplePNG).Frame(120, 80).Fit(ui.FitCover).Background(ui.TokenChrome),
			).Gap(th.Spacing.S).Align(ui.AlignCenter),
			ui.Column(
				ui.Caption("Fill"),
				ui.Image("img-fill", samplePNG).Frame(120, 80).Fit(ui.FitFill),
			).Gap(th.Spacing.S).Align(ui.AlignCenter),
		).Gap(th.Spacing.L).Wrap()),
	)
}

func (app *CatalogApp) pageSVG(c *ui.Ctx) ui.View {
	th := c.Theme()
	return app.pageShell(c, "SVG",
		app.section("From bytes", ui.Row(
			ui.SVG("svg-bytes", sampleSVG).Width(96),
			ui.Column(
				ui.Text("SVG from in-memory bytes"),
				ui.Caption("ui.SVG(id, data)"),
			).Gap(th.Spacing.XS).Align(ui.AlignStart),
		).Gap(th.Spacing.L).Align(ui.AlignCenter)),
		app.section("From embed.FS", ui.Row(
			ui.SVGFS("svg-embed", imageAssets, "testdata/logo.svg").Width(96),
			ui.Column(
				ui.Text("SVG via embed.FS"),
				ui.Caption("ui.SVGFS(id, fsys, name) — or ui.SVGFile(id, path) for the filesystem"),
			).Gap(th.Spacing.XS).Align(ui.AlignStart),
		).Gap(th.Spacing.L).Align(ui.AlignCenter)),
		app.section("currentColor", ui.Row(
			ui.Column(
				ui.Caption("Foreground"),
				ui.SVG("svg-fg", markSVG).Size(48),
			).Gap(th.Spacing.S).Align(ui.AlignCenter),
			ui.Column(
				ui.Caption("Accent"),
				ui.SVG("svg-accent", markSVG).Size(48).
					Style(ui.Spec{}.TextColor(ui.TokenAccent)),
			).Gap(th.Spacing.S).Align(ui.AlignCenter),
			ui.Column(
				ui.Caption("Success"),
				ui.SVG("svg-ok", markSVG).Size(48).
					Style(ui.Spec{}.TextColor(ui.TokenSuccess)),
			).Gap(th.Spacing.S).Align(ui.AlignCenter),
		).Gap(th.Spacing.L).Wrap()),
		app.section("Fit modes", ui.Row(
			ui.Column(
				ui.Caption("Contain"),
				ui.SVG("svg-contain", sampleSVG).Frame(120, 80).Fit(ui.FitContain),
			).Gap(th.Spacing.S).Align(ui.AlignCenter),
			ui.Column(
				ui.Caption("Cover"),
				ui.SVG("svg-cover", sampleSVG).Frame(120, 80).Fit(ui.FitCover).Background(ui.TokenChrome),
			).Gap(th.Spacing.S).Align(ui.AlignCenter),
			ui.Column(
				ui.Caption("Fill"),
				ui.SVG("svg-fill", sampleSVG).Frame(120, 80).Fit(ui.FitFill),
			).Gap(th.Spacing.S).Align(ui.AlignCenter),
		).Gap(th.Spacing.L).Wrap()),
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
			ui.Button("btn-ghost", ui.Text("Ghost")).Ghost().OnClick(func() { app.setStatus("Ghost clicked") }),
		).Gap(th.Spacing.S)),
		app.section("Footer style", ui.Row(
			ui.Button("btn-ln", ui.Caption("Ln 12, Col 4")).Ghost().
				IconStart(icons.Code).
				Tooltip("Go to line").
				OnClick(func() { app.setStatus("Go to line") }),
			ui.Button("btn-enc", ui.Caption("UTF-8")).Ghost().HoverFill().
				IconStart(icons.ChevronDown).
				Tooltip("Select encoding").
				OnClick(func() { app.setStatus("Select encoding") }),
			ui.Button("btn-notify", nil).Ghost().HoverFill().
				IconStart(icons.Bell).
				Tooltip("Notifications").
				OnClick(func() { app.setStatus("Notifications") }),
		).Gap(th.Spacing.M).Align(ui.AlignCenter)),
		app.section("Icon & hint", ui.Row(
			ui.Button("btn-icon", ui.Text("Save")).IconStart(icons.Save).Primary().Hint("⌘S").OnClick(func() { app.setStatus("Save clicked") }),
			ui.IconButton("btn-ib", icons.Settings).OnClick(func() { app.setStatus("Settings icon clicked") }),
			ui.IconButton("btn-add", icons.Plus).OnClick(func() { app.setStatus("Add icon clicked") }),
		).Gap(th.Spacing.S)),
		app.section("States", ui.Row(
			ui.Button("btn-disabled", ui.Text("Disabled")).Primary().Disabled(true),
			ui.Button("btn-loading", ui.Text("Loading")).Primary().IconStart(icons.RefreshCw).OnClick(func() {}),
		).Gap(th.Spacing.S)),
		app.section("Menu button", ui.Row(
			ui.MenuButton("btn-export", "Export", []ui.MenuItem{
				{Label: "CSV", OnSelect: func() { app.setStatus("Export CSV") }},
				{Label: "JSON", OnSelect: func() { app.setStatus("Export JSON") }},
			}).Primary().IconStart(icons.Download),
			ui.MenuButton("btn-save-split", "Save", []ui.MenuItem{
				{Label: "Save As…", OnSelect: func() { app.setStatus("Save As") }},
				{Label: "Save All", OnSelect: func() { app.setStatus("Save All") }},
			}).Primary().IconStart(icons.Save).OnClick(func() { app.setStatus("Save clicked") }),
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
			ui.SegmentItem{Icon: icons.LayoutPanelLeft, Value: "h"},
			ui.SegmentItem{Icon: icons.LayoutPanelTop, Value: "v"},
			ui.SegmentItem{Icon: icons.List, Value: "list"},
		).Selected(0).OnChange(func(v string) { app.setStatus("layout: " + v) })),
		app.section("Toolbar style", ui.Row(
			ui.Segmented("seg-fmt",
				ui.SegmentItem{Label: "B", Value: "bold"},
				ui.SegmentItem{Label: "I", Value: "italic"},
			).Selected(0).OnChange(func(v string) { app.setStatus("format: " + v) }),
			ui.Segmented("seg-align",
				ui.SegmentItem{Icon: icons.Menu, Value: "left"},
				ui.SegmentItem{Icon: icons.Ellipsis, Value: "center"},
				ui.SegmentItem{Icon: icons.EllipsisVertical, Value: "right"},
			).Selected(1).OnChange(func(v string) { app.setStatus("align: " + v) }),
		).Gap(th.Spacing.M)),
	)
}

// --- Forms pages ---

func (app *CatalogApp) pageTextFields(c *ui.Ctx) ui.View {
	th := c.Theme()
	return app.pageShell(c, "Text fields",
		app.section("Plain", ui.TextField("tf-plain", app.textPlain).
			Placeholder("Type here…").
			OnChange(func(s string) { app.textPlain = s }).
			Grow(1)),
		app.section("With icon", ui.TextField("tf-search", app.textSearch).
			Placeholder("Search…").
			IconStart(icons.Search).
			OnChange(func(s string) { app.textSearch = s }).
			Grow(1)),
		app.section("Password", ui.TextField("tf-pass", app.textPassword).
			Placeholder("Enter password").
			Password(true).
			IconStart(icons.Lock).
			OnChange(func(s string) { app.textPassword = s }).
			Grow(1)),
		app.section("Editable label", ui.Column(
			ui.EditableLabel("edit-title", app.editTitle).
				OnSave(func(s string) {
					app.editTitle = s
					app.setStatus("saved title: " + s)
				}),
			ui.EditableLabel("edit-name", app.editName).
				Placeholder("Click to name…").
				OnSave(func(s string) {
					app.editName = s
					app.setStatus("saved name: " + s)
				}),
		).Gap(th.Spacing.S)),
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
			ui.FormSlider("f-vol", "Volume", "Master output level", app.formVol, 0, 100, 1, func(v float64) {
				app.formVol = v
				app.setStatus(fmt.Sprintf("volume: %.0f", v))
			}),
			ui.FormStepper("f-count", "Retries", "Number of retry attempts", app.formCount, 0, 10, 1, func(v float64) {
				app.formCount = v
				app.setStatus(fmt.Sprintf("retries: %.0f", v))
			}),
		),
	)
}

func (app *CatalogApp) pageSlider(c *ui.Ctx) ui.View {
	th := c.Theme()
	return app.pageShell(c, "Slider & stepper",
		app.section("Slider", ui.Column(
			ui.Slider("sl-main", app.sliderVal).Min(0).Max(100).Step(1).Width(280).
				OnFloatChange(func(v float64) {
					app.sliderVal = v
					app.setStatus(fmt.Sprintf("slider: %.0f", v))
				}),
			ui.Muted(fmt.Sprintf("Value: %.0f", app.sliderVal)),
		).Gap(th.Spacing.S)),
		app.section("Number stepper", ui.Row(
			ui.NumberStepper("ns-main", app.stepperVal).Min(0).Max(20).Step(1).
				OnFloatChange(func(v float64) {
					app.stepperVal = v
					app.setStatus(fmt.Sprintf("stepper: %.0f", v))
				}),
			ui.Muted(fmt.Sprintf("Count: %.0f", app.stepperVal)),
		).Gap(th.Spacing.M)),
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
			ui.NavItem{ID: "home", Label: "Home", Icon: icons.House},
			ui.NavItem{ID: "code", Label: "Code", Icon: icons.Code},
			ui.NavItem{ID: "settings", Label: "Settings", Icon: icons.Settings},
		).Selected(app.navVert).OnSelectItem(func(i int, id string) {
			app.navVert = i
			app.setStatus("nav: " + id)
		}).Width(200)),
		app.section("Horizontal nav", ui.Nav("nav-h", ui.NavHorizontal, ui.NavIconTop,
			ui.NavItem{ID: "new", Label: "New", Icon: icons.Plus},
			ui.NavItem{ID: "open", Label: "Open", Icon: icons.FolderOpen},
			ui.NavItem{ID: "save", Label: "Save", Icon: icons.Save},
		).Selected(app.navHoriz).OnSelectItem(func(i int, id string) {
			app.navHoriz = i
			app.setStatus("nav: " + id)
		})),
		app.section("Tabs (closable)", ui.Tabs("nav-tabs", []ui.TabModel{
			{Title: "main.go", Modified: true},
			{Title: "app.go"},
			{Title: "README.md", Badge: "2"},
		}).Selected(app.tabIdx).OnSelectItem(func(i int, _ string) {
			app.tabIdx = i
			app.setStatus(fmt.Sprintf("tab: %d", i))
		}).OnTabClose(func(i int) { app.setStatus(fmt.Sprintf("closed tab: %d", i)) })),
		app.section("Tabs (section switcher)", ui.Tabs("nav-tabs-section", []ui.TabModel{
			{Title: "Body"},
			{Title: "Headers"},
		}).Selected(app.tabSectionIdx).OnSelectItem(func(i int, _ string) {
			app.tabSectionIdx = i
			app.setStatus(fmt.Sprintf("section tab: %d", i))
		}).Closable(false)),
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

func catalogDrawerEdge(i int) ui.Edge {
	switch i {
	case 0:
		return ui.EdgeLeft
	case 2:
		return ui.EdgeTop
	case 3:
		return ui.EdgeBottom
	default:
		return ui.EdgeRight
	}
}

func (app *CatalogApp) pageDrawer(c *ui.Ctx) ui.View {
	th := c.Theme()

	inspector := ui.Card("Inspector", "Slide-from-edge panel",
		ui.Column(
			ui.Muted("Resize from the inner edge. Toggle overlay vs push below."),
			ui.Form("drawer-form",
				ui.FormSwitch("drawer-notify", "Notifications", "Demo toggle", app.switchOn, func(v bool) {
					app.switchOn = v
				}),
			),
		).Gap(th.Spacing.S),
	).Grow(1)

	workspace := ui.Column(
		ui.Strong("Workspace"),
		ui.Muted("Main page content lives here. Open the drawer to inspect or edit."),
		ui.Spacer(),
		ui.Row(
			ui.Badge("demo").Tone(ui.BadgeMuted),
			ui.Link("drawer-link", "Learn more").OnClick(func() {
				app.setStatus("drawer: link clicked")
			}),
		).Gap(th.Spacing.S),
	).Gap(th.Spacing.S).Padding(th.Spacing.M).Grow(1).Background(ui.TokenSurface)

	singleDrawer := ui.Drawer("drawer-demo", inspector, workspace).
		Open(app.drawerOpen).
		Edge(catalogDrawerEdge(app.drawerEdge)).
		Size(280).
		Resizable(true).
		Swipe(app.drawerSwipe).
		OnOpenChange(func(v bool) {
			app.drawerOpen = v
			app.setStatus(fmt.Sprintf("drawer open=%v", v))
		}).
		Grow(1)
	if app.drawerPush {
		singleDrawer = singleDrawer.Push()
	} else {
		singleDrawer = singleDrawer.Overlay().Modal(app.drawerModal)
	}

	singleControls := ui.Column(
		ui.Row(
			ui.Button("drawer-toggle", ui.Text("Toggle drawer")).Primary().
				OnClick(func() {
					app.drawerOpen = !app.drawerOpen
					app.setStatus(fmt.Sprintf("drawer open=%v", app.drawerOpen))
				}),
			ui.Checkbox("drawer-push", "Push layout").Check(app.drawerPush).OnToggle(func(v bool) {
				app.drawerPush = v
			}),
			ui.Checkbox("drawer-modal", "Modal scrim").Check(app.drawerModal).OnToggle(func(v bool) {
				app.drawerModal = v
			}),
			ui.Checkbox("drawer-swipe", "Swipe").Check(app.drawerSwipe).OnToggle(func(v bool) {
				app.drawerSwipe = v
			}),
		).Gap(th.Spacing.S).Wrap(),
		ui.Segmented("drawer-edge",
			ui.SegmentItem{Label: "Left", Value: "left"},
			ui.SegmentItem{Label: "Right", Value: "right"},
			ui.SegmentItem{Label: "Top", Value: "top"},
			ui.SegmentItem{Label: "Bottom", Value: "bottom"},
		).Selected(app.drawerEdge).OnChange(func(v string) {
			switch v {
			case "left":
				app.drawerEdge = 0
			case "top":
				app.drawerEdge = 2
			case "bottom":
				app.drawerEdge = 3
			default:
				app.drawerEdge = 1
			}
		}),
	).Gap(th.Spacing.S)

	editor := ui.Column(
		ui.Strong("Editor"),
		ui.Muted("func main() {\n    yoga.Run(cfg, Build)\n}"),
	).Gap(th.Spacing.S).Padding(th.Spacing.M).Grow(1).Background(ui.TokenSurface)

	termPanel := ui.Card("Terminal", "Bottom push drawer",
		ui.Muted("$ go test ./...\nok  github.com/mirzakhany/yoga/ui"),
	).Grow(1)

	chatPanel := ui.Card("Chat", "Right push drawer",
		ui.Column(
			ui.Muted("Ask the assistant…"),
			ui.TextField("chat-in", "").Placeholder("Message").Grow(1),
		).Gap(th.Spacing.S).Grow(1),
	).Grow(1)

	nested := ui.Drawer("drawer-chat", chatPanel,
		ui.Drawer("drawer-term", termPanel, editor).
			Edge(ui.EdgeBottom).Push().Open(app.termOpen).Size(160).Resizable(true).
			OnOpenChange(func(v bool) {
				app.termOpen = v
				app.setStatus(fmt.Sprintf("terminal open=%v", v))
			}).Grow(1),
	).Edge(ui.EdgeRight).Push().Open(app.chatOpen).Size(260).Resizable(true).
		OnOpenChange(func(v bool) {
			app.chatOpen = v
			app.setStatus(fmt.Sprintf("chat open=%v", v))
		}).Grow(1)

	nestedChrome := ui.Column(
		ui.Row(
			ui.Button("term-toggle", ui.Text("Terminal")).IconStart(icons.Terminal).
				OnClick(func() {
					app.termOpen = !app.termOpen
					app.setStatus(fmt.Sprintf("terminal open=%v", app.termOpen))
				}),
			ui.Button("chat-toggle", ui.Text("Chat")).IconStart(icons.User).
				OnClick(func() {
					app.chatOpen = !app.chatOpen
					app.setStatus(fmt.Sprintf("chat open=%v", app.chatOpen))
				}),
			ui.Muted("VS Code-style: bottom terminal + right chat, both resizable."),
		).Gap(th.Spacing.S).Wrap(),
		ui.ViewOf(nested).Height(220).Grow(1),
	).Gap(th.Spacing.S)

	return app.pageShell(c, "Drawer",
		app.section("Single drawer", ui.Column(
			singleControls,
			ui.ViewOf(singleDrawer).Height(220),
		).Gap(th.Spacing.S)),
		app.section("Two drawers", nestedChrome),
	)
}

// --- Data pages ---

func (app *CatalogApp) pageTable(c *ui.Ctx) ui.View {
	return app.pageShell(c, "Table",
		app.section("Editable table", ui.Column(
			ui.Row(
				ui.TextField("tbl-filter", app.kvFilter).
					Placeholder("Filter rows…").
					IconStart(icons.Search).
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
		app.section("File tree", ui.Column(
			ui.Row(
				ui.Spacer(),
				ui.Button("tree-add", ui.Text("Add Item")).OnClick(func() {
					id := fmt.Sprintf("item-%d", time.Now().UnixNano())
					app.demoTree.AddChild(nil, &ui.TreeNode{Label: id, Leaf: true, Data: id})
					app.setStatus("added tree item")
				}),
			).Gap(c.Theme().Spacing.S),
			ui.ViewOf(app.demoTree).Height(280).Grow(1),
		).Gap(c.Theme().Spacing.S)),
	)
}

// --- Overlays page ---

func (app *CatalogApp) pageOverlays(c *ui.Ctx) ui.View {
	th := c.Theme()
	return app.pageShell(c, "Overlays",
		app.section("Tooltip", ui.Row(
			ui.Button("tip-save", ui.Text("Hover me")).Primary().Tooltip("Saves the current document"),
			ui.IconButton("tip-help", icons.CircleQuestionMark).Tooltip("Show help"),
			ui.Badge("β").Tone(ui.BadgeAccent).Tooltip("Beta feature"),
		).Gap(th.Spacing.S)),
		app.section("Popover", ui.Popover("pop-demo",
			ui.Button("pop-trig", ui.Text("Open popover")),
			ui.Column(
				ui.Strong("Quick settings"),
				ui.Muted("Anchored overlay without a scrim."),
				ui.Switch("pop-sw").Check(app.switchOn).OnToggle(func(v bool) {
					app.switchOn = v
				}),
			).Gap(th.Spacing.S),
		).Open(app.popoverOpen).OnOpenChange(func(v bool) {
			app.popoverOpen = v
			app.setStatus(fmt.Sprintf("popover open=%v", v))
		}).Width(260).Height(140).Placement(ui.PlacementBottom)),
		app.section("Context menu", ui.ContextMenu("ctx-card",
			ui.Card("Right-click me", "Context menu demo", ui.Muted("Use the secondary mouse button.")).Width(280),
			[]ui.MenuItem{
				{Label: "Copy", OnSelect: func() { app.setStatus("ctx: Copy") }},
				{Label: "Paste", OnSelect: func() { app.setStatus("ctx: Paste") }},
				{Label: "Delete", OnSelect: func() { app.setStatus("ctx: Delete") }},
			},
		)),
		app.section("Command palette", ui.Column(
			ui.Muted("Searchable command list with shortcuts. Press "+c.Commands().ToggleLabel()+" or use the button."),
			ui.Row(
				ui.Button("cmd-palette-page", ui.Text("Open command palette")).
					IconStart(icons.Search).
					Primary().
					Hint(c.Commands().ToggleLabel()).
					OnClick(func() { c.Commands().Show() }),
				ui.Kbd(c.Commands().ToggleLabel()),
			).Gap(th.Spacing.S),
		).Gap(th.Spacing.S)),
		app.section("Note", ui.Caption("Table row actions also show TableAction.Tooltip on hover.")),
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

func (app *CatalogApp) pageProgress(c *ui.Ctx) ui.View {
	th := c.Theme()
	return app.pageShell(c, "Progress",
		app.section("Progress bar", ui.Column(
			ui.ProgressBar("pb", app.progressVal).Width(280),
			ui.Row(
				ui.Button("pb-dec", ui.Text("−")).OnClick(func() {
					app.progressVal = clampf32(app.progressVal-0.1, 0, 1)
				}),
				ui.Button("pb-inc", ui.Text("+")).OnClick(func() {
					app.progressVal = clampf32(app.progressVal+0.1, 0, 1)
				}),
				ui.Muted(fmt.Sprintf("%.0f%%", app.progressVal*100)),
			).Gap(th.Spacing.S),
			ui.ProgressBar("pb-indet", 0).Width(280).Indeterminate(),
		).Gap(th.Spacing.S)),
		app.section("Progress ring", ui.Row(
			ui.ProgressRing("pr", app.progressVal),
			ui.ProgressRing("pr-indet", 0).Indeterminate(),
		).Gap(th.Spacing.L)),
		app.section("Skeleton", ui.Column(
			ui.Row(
				ui.Skeleton("sk-av").Circle(40),
				ui.Column(
					ui.Skeleton("sk-1").Width(200).Height(12),
					ui.Skeleton("sk-2").Width(160).Height(12),
				).Gap(th.Spacing.S),
			).Gap(th.Spacing.M),
		)),
		app.section("Empty state", ui.EmptyState("No results", "Try a different filter or create a new item.").
			EmptyIcon(icons.Search).
			Action(ui.Button("empty-add", ui.Text("Create")).Primary().OnClick(func() {
				app.setStatus("create from empty state")
			})).Height(200)),
	)
}

func clampf32(v, lo, hi float32) float32 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
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
