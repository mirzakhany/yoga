package main

import (
	"strconv"
	"time"

	"github.com/mirzakhany/yoga"
	"github.com/mirzakhany/yoga/theme"
	"github.com/mirzakhany/yoga/ui"
)

const sidebarWidth = 240

type catalogPage struct {
	id, label, icon, group string
}

var catalogPages = []catalogPage{
	// Content
	{"typography", "Typography", "edit", "Content"},
	{"surfaces", "Surfaces", "grid", "Content"},
	{"icons", "Icons", "star", "Content"},
	// Actions
	{"buttons", "Buttons", "add", "Actions"},
	{"segmented", "Segmented", "list", "Actions"},
	// Forms
	{"text-fields", "Text fields", "search", "Forms"},
	{"selection", "Selection", "check_circle", "Forms"},
	{"choice", "Choice", "expand_more", "Forms"},
	{"form", "Form rows", "settings", "Forms"},
	{"slider", "Slider", "drag", "Forms"},
	// Navigation
	{"nav", "Navigation", "menu", "Navigation"},
	// Overlays
	{"overlays", "Overlays", "open_in_new", "Overlays"},
	// Data
	{"table", "Table", "list", "Data"},
	{"tree", "Tree", "folder", "Data"},
	// Feedback
	{"feedback", "Feedback", "notifications", "Feedback"},
	{"progress", "Progress", "refresh", "Feedback"},
}

// CatalogApp is the component gallery demo.
type CatalogApp struct {
	pageID string
	status string
	theme  string

	// Buttons page
	btnSeg int

	// Forms page
	textPlain    string
	textSearch   string
	textPassword string
	checkA       bool
	checkB       bool
	radio        int
	switchOn     bool
	selectV      string
	tags         []string
	formNotify   bool
	formTheme    string
	formSize     float64
	formFile     string

	// Slider / stepper
	sliderVal  float64
	stepperVal float64
	formVol    float64
	formCount  float64

	// Overlays
	popoverOpen bool

	// Accordion
	accordionOpen string

	// Progress
	progressVal float32

	// Navigation page
	navVert  int
	navHoriz int
	tabIdx   int
	segSplit int
	splitA   string
	splitB   string
	crumb    int

	// Data
	kvTable  *ui.Table
	kvFilter string
	demoTree *ui.Tree

	// Feedback
	alertDismissed bool
}

var _ yoga.App = (*CatalogApp)(nil)

func BuildCatalog() *CatalogApp {
	app := &CatalogApp{
		pageID:        "buttons",
		theme:         theme.Current().Name,
		checkB:        true,
		switchOn:      true,
		selectV:       "go",
		tags:          []string{"ui", "yoga"},
		formNotify:    true,
		formTheme:     "yoga-dark",
		formSize:      14,
		formFile:      "main.go",
		sliderVal:     40,
		stepperVal:    3,
		formVol:       50,
		formCount:     2,
		accordionOpen: "a",
		progressVal:   0.45,
		splitA:        "Left pane",
		splitB:        "Right pane",
	}
	app.kvTable = ui.NewTable([]ui.TableColumn{
		{ID: "sel", Label: "", Kind: ui.TableColCheckbox, Width: 36},
		{ID: "key", Label: "Key", Kind: ui.TableColEditable, Width: 0, Sortable: true},
		{ID: "val", Label: "Value", Kind: ui.TableColEditable, Width: 0, Sortable: true},
		{ID: "act", Label: "", Kind: ui.TableColActions, Width: 40, Locked: true},
	}, []ui.TableAction{{Icon: "delete", Tooltip: "Delete"}})
	app.kvTable.SetRows([]ui.TableRow{
		{ID: "r1", Cells: map[string]string{"key": "Content-Type", "val": "application/json"}},
		{ID: "r2", Cells: map[string]string{"key": "Authorization", "val": "Bearer token"}},
		{ID: "r3", Cells: map[string]string{"key": "Accept", "val": "text/html"}},
	})
	app.kvTable.Actions[0].OnClick = func(rowID string) {
		app.kvTable.RemoveRow(rowID)
		app.setStatus("deleted row " + rowID)
	}

	root := &ui.TreeNode{Label: "src", Data: "src"}
	app.demoTree = ui.NewTree(root)
	app.demoTree.Loader = func(n *ui.TreeNode) []*ui.TreeNode {
		switch n.Data {
		case "src":
			return []*ui.TreeNode{
				{Label: "main.go", Leaf: true, Data: "main.go"},
				{Label: "app.go", Leaf: true, Data: "app.go"},
				{Label: "ui", Data: "ui"},
			}
		case "ui":
			return []*ui.TreeNode{
				{Label: "button.go", Leaf: true, Data: "button.go"},
				{Label: "table.go", Leaf: true, Data: "table.go"},
			}
		default:
			return nil
		}
	}
	app.demoTree.SetRoot(root)
	app.demoTree.OnActivate = func(n *ui.TreeNode) {
		app.setStatus("tree: " + n.Label)
	}
	return app
}

func (app *CatalogApp) setStatus(s string) { app.status = s }

func (app *CatalogApp) registerCommands(c *ui.Ctx) {
	cmds := make([]*ui.Command, 0, len(catalogPages)+10)

	cmds = append(cmds, ui.Section("Recent"))
	recent := []struct{ name, path string }{
		{"app.go", "example/catalog/app.go"},
		{"pages.go", "example/catalog/pages.go"},
		{"commands.go", "ui/commands.go"},
	}
	for _, f := range recent {
		file := f
		cmds = append(cmds, ui.Item("recent."+file.path).
			Title(file.name).
			Detail(file.path).
			Icon("file").
			Run(func() {
				app.setStatus("open recent: " + file.path)
			}))
	}

	cmds = append(cmds, ui.Section("Commands"))
	for _, p := range catalogPages {
		page := p
		cmds = append(cmds, ui.Cmd("nav."+page.id).
			Title("Go to "+page.label).
			Group(page.group).
			Icon(page.icon).
			Run(func() {
				app.pageID = page.id
				app.setStatus("navigated to " + page.label)
			}))
	}
	cmds = append(cmds,
		ui.Cmd("view.theme.next").
			Title("Cycle Theme").
			Group("View").
			Icon("theme").
			Shortcut("⌘T").
			Run(func() {
				names := theme.Names()
				if len(names) == 0 {
					return
				}
				idx := themeIndex(app.theme)
				next := names[(idx+1)%len(names)]
				theme.Use(next)
				app.theme = next
				app.formTheme = next
				app.setStatus("theme: " + next)
			}),
		ui.Cmd("toast.info").
			Title("Show Info Toast").
			Group("Feedback").
			Icon("notifications").
			Run(func() {
				c.Toasts().Show("Info from command palette", ui.ToastInfo, 3*time.Second)
			}),
		ui.Cmd("toast.success").
			Title("Show Success Toast").
			Group("Feedback").
			Icon("check").
			Run(func() {
				c.Toasts().Show("Success from command palette", ui.ToastSuccess, 3*time.Second)
			}),
	)
	c.Commands().Register(cmds...)
}

func (app *CatalogApp) Body(c *ui.Ctx) ui.View {
	app.registerCommands(c)
	th := c.Theme()
	return ui.Column(
		app.topBar(c),
		ui.HLine(th.Stroke.Thin, th.Border),
		ui.Row(
			app.sidebar(c),
			ui.VLine(th.Stroke.Thin, th.Border),
			ui.Scroll("catalog-page-"+app.pageID, app.pageContent(c)).
				Grow(1).
				Background(ui.TokenSurface),
		).Align(ui.AlignStretch).Grow(1),
	).Grow(1).Background(ui.TokenSurface)
}

func (app *CatalogApp) topBar(c *ui.Ctx) ui.View {
	th := c.Theme()
	themes := make([]ui.SelectOption, 0, len(theme.Names()))
	for _, name := range theme.Names() {
		n := name
		themes = append(themes, ui.SelectOption{Label: n, Value: n})
	}
	return ui.Row(
		ui.Dropdown("menu-file", "File", []ui.MenuItem{
			{Label: "Open file…", OnSelect: func() { app.showOpenFile(c) }},
			{Label: "Save file…", OnSelect: func() { app.showSaveFile(c) }},
		}),
		ui.Dropdown("menu-go", "Go", app.goMenuItems()),
		ui.Dropdown("menu-help", "Help", []ui.MenuItem{
			{Label: "About Yoga", OnSelect: func() {
				c.Dialogs().ShowError("About Yoga", "Yoga component catalog demo.", nil)
			}},
		}),
		ui.Spacer(),
		ui.Button("cmd-palette", ui.Text("Commands")).Width(300).
			IconStart("search").
			Hint(c.Commands().ToggleLabel()).
			OnClick(func() { c.Commands().Show() }),
		ui.Spacer(),
		ui.Select("theme", themes).
			Width(180).
			Selected(themeIndex(app.theme)).
			OnChange(func(v string) {
				theme.Use(v)
				app.theme = v
				app.formTheme = v
			}),
	).Gap(th.Spacing.S).PaddingXY(th.Spacing.M, th.Spacing.S).
		Background(ui.TokenChrome).
		Shrink(0) // chrome must not compress when a page is taller than the window
}

func (app *CatalogApp) goMenuItems() []ui.MenuItem {
	items := make([]ui.MenuItem, 0, len(catalogPages))
	for _, p := range catalogPages {
		page := p
		items = append(items, ui.MenuItem{
			Label: page.label,
			OnSelect: func() {
				app.pageID = page.id
				app.setStatus("navigated to " + page.label)
			},
		})
	}
	return items
}

func (app *CatalogApp) sidebar(c *ui.Ctx) ui.View {
	th := c.Theme()
	header := ui.Row(
		ui.Icon("grid", th.Metrics.IconSizeMD, th.Accent),
		ui.Strong("Yoga Components"),
	).Gap(th.Spacing.S).PaddingXY(th.Spacing.M, th.Spacing.M).Shrink(0)

	var nav []ui.View
	prevGroup := ""
	for _, p := range catalogPages {
		if p.group != prevGroup {
			if prevGroup != "" {
				nav = append(nav, ui.Spacer().Height(th.Spacing.XS))
			}
			// Text respects PaddingLeft (border-box + paint inset).
			nav = append(nav, ui.Caption(p.group).
				Style(ui.Spec{}.TextColor(ui.TokenForegroundMuted)).
				PaddingLeft(th.Spacing.M).
				MarginTop(th.Spacing.S).
				MarginBottom(th.Spacing.XS))
			prevGroup = p.group
		}
		nav = append(nav, app.sidebarItem(c, p))
	}

	footer := ui.Row(
		ui.Muted("Component catalog"),
		ui.Spacer(),
		ui.Caption(formatCount(len(catalogPages))).
			Style(ui.Spec{}.TextColor(ui.TokenForegroundMuted)),
	).PaddingXY(th.Spacing.M, th.Spacing.M).Shrink(0)

	bg := th.Chrome
	// Fixed width only — Grow would share free space with the page and leave a gap.
	// Shrink(0) keeps the nav strip from compressing when page content is tall.
	// Scroll the page list between sticky header/footer so long catalogs fit.
	return ui.Column(
		header,
		ui.Scroll("catalog-sidebar", ui.Column(nav...)),
		footer,
	).Width(sidebarWidth).
		BackgroundColor(bg).
		Shrink(0)
}

func (app *CatalogApp) sidebarItem(c *ui.Ctx, p catalogPage) ui.View {
	th := c.Theme()
	selected := app.pageID == p.id
	label := ui.Row(
		ui.Icon(p.icon, th.Metrics.IconSizeSM, th.Foreground),
		ui.Text(p.label),
	).Gap(th.Spacing.S)

	var spec ui.Spec
	if selected {
		spec = ui.Background(ui.TokenListActive).
			Padding(th.Spacing.S).
			PaddingLeft(th.Spacing.M).
			PaddingRight(th.Spacing.M).
			Radius(th.Radius.Small).
			Cursor(ui.CursorPointer)
	} else {
		spec = ui.Background(ui.TokenUnset).
			When(ui.Hovered, ui.Background(ui.TokenListHover)).
			Padding(th.Spacing.S).
			PaddingLeft(th.Spacing.M).
			PaddingRight(th.Spacing.M).
			Radius(th.Radius.Small).
			Cursor(ui.CursorPointer)
	}

	pageID := p.id
	return ui.Button("cat-"+p.id, label).
		Subtle().
		Style(spec).
		OnClick(func() {
			app.pageID = pageID
			app.setStatus("")
		})
}

func (app *CatalogApp) pageContent(c *ui.Ctx) ui.View {
	switch app.pageID {
	case "typography":
		return app.pageTypography(c)
	case "surfaces":
		return app.pageSurfaces(c)
	case "icons":
		return app.pageIcons(c)
	case "buttons":
		return app.pageButtons(c)
	case "segmented":
		return app.pageSegmented(c)
	case "text-fields":
		return app.pageTextFields(c)
	case "selection":
		return app.pageSelection(c)
	case "choice":
		return app.pageChoice(c)
	case "form":
		return app.pageForm(c)
	case "slider":
		return app.pageSlider(c)
	case "nav":
		return app.pageNavigation(c)
	case "overlays":
		return app.pageOverlays(c)
	case "table":
		return app.pageTable(c)
	case "tree":
		return app.pageTree(c)
	case "feedback":
		return app.pageFeedback(c)
	case "progress":
		return app.pageProgress(c)
	default:
		return app.pageButtons(c)
	}
}

func themeIndex(name string) int {
	for i, n := range theme.Names() {
		if n == name {
			return i
		}
	}
	return 0
}

func formatCount(n int) string {
	return strconv.Itoa(n)
}
