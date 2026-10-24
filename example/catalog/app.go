package main

import (
	"strconv"
	"time"

	"github.com/mirzakhany/yoga"
	"github.com/mirzakhany/yoga/icons"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/theme"
	"github.com/mirzakhany/yoga/ui"
)

const sidebarWidth = 240

type catalogPage struct {
	id, label string
	icon      icons.Icon
	group     string
}

var catalogPages = []catalogPage{
	// Content
	{"typography", "Typography", icons.Pencil, "Content"},
	{"surfaces", "Surfaces", icons.LayoutGrid, "Content"},
	{"icons", "Icons", icons.Star, "Content"},
	{"images", "Images", icons.Image, "Content"},
	{"svg", "SVG", icons.Shapes, "Content"},
	// Actions
	{"buttons", "Buttons", icons.Plus, "Actions"},
	{"segmented", "Segmented", icons.List, "Actions"},
	// Forms
	{"text-fields", "Text fields", icons.Search, "Forms"},
	{"selection", "Selection", icons.CircleCheck, "Forms"},
	{"choice", "Choice", icons.ChevronDown, "Forms"},
	{"form", "Form rows", icons.Settings, "Forms"},
	{"slider", "Slider", icons.GripVertical, "Forms"},
	// Navigation
	{"nav", "Navigation", icons.Menu, "Navigation"},
	{"drawer", "Drawer", icons.LayoutPanelLeft, "Navigation"},
	// Overlays
	{"overlays", "Overlays", icons.ExternalLink, "Overlays"},
	// Data
	{"table", "Table", icons.List, "Data"},
	{"tree", "Tree", icons.Folder, "Data"},
	// Feedback
	{"feedback", "Feedback", icons.Bell, "Feedback"},
	{"progress", "Progress", icons.RefreshCw, "Feedback"},
}

// CatalogApp is the component gallery demo.
type CatalogApp struct {
	pageID string
	status string
	theme  string

	paletteCompact bool // the request switcher uses compact rows and a hint footer

	// Tree page: a request collection
	reqTree   *ui.Tree
	reqTarget *ui.TreeNode

	// Input group demo
	reqMethod, reqURL string

	// Buttons page
	btnSeg int

	// Forms page
	textPlain    string
	textSearch   string
	textPassword string
	editTitle    string
	editName     string
	checkA       bool
	checkB       bool
	radio        int
	switchOn     bool
	selectV      string
	tags         []string
	protoPaths   []string
	formNotify   bool
	formTheme    string
	formSize     float64
	formFile     string
	formCert     string

	// Slider / stepper
	sliderVal  float64
	stepperVal float64
	formVol    float64
	formCount  float64

	// Overlays
	popoverOpen bool

	// Accordion
	accordionOpen string

	// Icons page
	iconSearch string

	// Progress
	progressVal float32

	// Navigation page
	navVert            int
	navHoriz           int
	tabIdx             int
	docTabs            []ui.TabModel
	tabSectionIdx      int
	splitA             string
	splitB             string
	splitHandleOnHover bool
	crumb              int

	// Drawer page
	drawerOpen  bool
	drawerEdge  int
	drawerPush  bool
	drawerModal bool
	drawerSwipe bool
	termOpen    bool
	chatOpen    bool

	// Data
	kvTable          *ui.Table
	kvFilter         string
	tblEditable      bool
	tblHighlight     bool
	tblOpaque        bool
	tblCollapseEmpty bool
	tblBg            render.Color
	demoTree         *ui.Tree

	// Feedback
	alertDismissed bool
}

var _ yoga.App = (*CatalogApp)(nil)

func BuildCatalog() *CatalogApp {
	app := &CatalogApp{
		pageID:   "buttons",
		theme:    theme.Current().Name,
		checkB:   true,
		switchOn: true,
		selectV:  "go",
		tags:     []string{"ui", "yoga"},
		protoPaths: []string{
			"/Users/me/src/service/api/v1/service.proto",
			"/Users/me/src/service/vendor/google/api/annotations.proto",
		},
		formNotify:         true,
		formTheme:          "yoga-dark",
		formSize:           14,
		formFile:           "main.go",
		sliderVal:          40,
		stepperVal:         3,
		formVol:            50,
		formCount:          2,
		accordionOpen:      "a",
		progressVal:        0.45,
		splitA:             "Left pane",
		splitB:             "Right pane",
		splitHandleOnHover: true,
		editTitle:          "Project name",
		editName:           "",
		drawerEdge:         1,
		tblEditable:        true,
		tblHighlight:       true,
	}
	app.kvTable = ui.NewTable([]ui.TableColumn{
		{ID: "sel", Label: "", Kind: ui.TableColCheckbox, Width: 36},
		{ID: "key", Label: "Key", Kind: ui.TableColEditable, Width: 0, Sortable: true},
		{ID: "val", Label: "Value", Kind: ui.TableColEditable, Width: 0, Sortable: true},
		{ID: "act", Label: "", Kind: ui.TableColActions, Width: 40, Locked: true},
	}, []ui.TableAction{{Icon: icons.Trash2, Tooltip: "Delete"}})
	app.kvTable.SetRows([]ui.TableRow{
		{ID: "r1", Cells: map[string]string{"key": "Content-Type", "val": "application/json"}},
		{ID: "r2", Cells: map[string]string{"key": "Authorization", "val": "Bearer token"}},
		{ID: "r3", Cells: map[string]string{"key": "Accept", "val": "text/html"}},
	})
	app.kvTable.Actions[0].OnClick = func(rowID string) {
		app.kvTable.RemoveRow(rowID)
		app.setStatus("deleted row " + rowID)
	}

	root := &ui.TreeNode{
		Label: "src",
		Data:  "src",
		Children: []*ui.TreeNode{
			{Label: "main.go", Leaf: true, Data: "main.go"},
			{Label: "app.go", Leaf: true, Data: "app.go"},
			{
				Label: "ui",
				Data:  "ui",
				Children: []*ui.TreeNode{
					{Label: "button.go", Leaf: true, Data: "button.go"},
					{Label: "table.go", Leaf: true, Data: "table.go"},
				},
			},
		},
	}
	app.demoTree = ui.NewTree(root)
	app.demoTree.ContextMenu = func(n *ui.TreeNode) []ui.MenuItem {
		if n == nil {
			return nil
		}
		return []ui.MenuItem{{
			Label: "Remove",
			OnSelect: func() {
				app.demoTree.Remove(n)
				app.setStatus("removed " + n.Label)
			},
		}}
	}
	app.demoTree.OnActivate = func(n *ui.TreeNode) {
		app.setStatus("tree: " + n.Label)
	}
	app.buildRequestTree()
	return app
}

// buildRequestTree makes a collection tree whose requests show method
// badges and whose folders show how many requests they hold.
func (app *CatalogApp) buildRequestTree() {
	req := func(method, name string) *ui.TreeNode {
		return &ui.TreeNode{Label: name, Leaf: true, Data: method}
	}
	folder := func(name string, kids ...*ui.TreeNode) *ui.TreeNode {
		return &ui.TreeNode{Label: name, Children: kids}
	}
	app.reqTarget = req("DELETE", "Delete todo")
	root := &ui.TreeNode{Children: []*ui.TreeNode{
		folder("Users", req("GET", "List users"), req("GET", "Get user"), req("POST", "Create user")),
		folder("Auth", req("POST", "Sign in"), req("POST", "Refresh token")),
		folder("Todos",
			req("GET", "List todos"), req("POST", "Create todo"),
			folder("Admin", req("PUT", "Update todo"), app.reqTarget)),
	}}
	app.reqTree = ui.NewTree(root)
	app.reqTree.Leading = func(n *ui.TreeNode) ui.View {
		method, ok := n.Data.(string)
		if !ok {
			return nil // folders keep their icon
		}
		return ui.Row(ui.Badge(method).Hue(methodHue(method)).Mono()).Width(64)
	}
	app.reqTree.Trailing = func(n *ui.TreeNode) ui.View {
		if n.Leaf {
			return nil
		}
		return ui.Caption(strconv.Itoa(countRequests(n))).Style(ui.Spec{}.TextColor(ui.TokenForegroundMuted))
	}
}

func countRequests(n *ui.TreeNode) int {
	total := 0
	for _, c := range n.Children {
		if c.Leaf {
			total++
		} else {
			total += countRequests(c)
		}
	}
	return total
}

func (app *CatalogApp) setStatus(s string) { app.status = s }

func (app *CatalogApp) setDensity(d theme.Density, name string) {
	theme.UseDensity(d)
	app.setStatus("density: " + name)
}

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
			Icon(icons.File).
			Run(func() {
				app.setStatus("open recent: " + file.path)
			}))
	}

	// A compact "Go to request" list: method badges, a trailing collection,
	// and ⌘↵ to open and send at once.
	requests := []struct {
		method, name, collection string
		hue                      ui.Hue
	}{
		{"GET", "List todos", "Todos", ui.HueGreen},
		{"POST", "Create todo", "Todos", ui.HueBlue},
		{"PUT", "Update todo", "Todos", ui.HueOrange},
		{"DELETE", "Delete todo", "Todos", ui.HueRed},
		{"GET", "List users", "Users", ui.HueGreen},
		{"GET", "Get user", "Users", ui.HueGreen},
		{"POST", "Sign in", "Auth", ui.HueBlue},
		{"POST", "Refresh token", "Auth", ui.HueBlue},
		{"GET", "Health check", "Ops", ui.HueGreen},
		{"GET", "Metrics", "Ops", ui.HueGreen},
		{"DELETE", "Purge cache", "Ops", ui.HueRed},
	}
	cmds = append(cmds, ui.Section("Todos").Scope("requests"))
	section := "Todos"
	for _, r := range requests {
		req := r
		if req.collection != section {
			section = req.collection
			cmds = append(cmds, ui.Section(section).Scope("requests"))
		}
		cmds = append(cmds, ui.Item("req."+req.name).Scope("requests").
			Title(req.name).
			// A fixed-width slot keeps the titles lined up.
			Leading(ui.Row(ui.Badge(req.method).Hue(req.hue).Mono()).Width(64)).
			Trailing(req.collection).
			Run(func() { app.setStatus("open request: " + req.name) }).
			Alt("⌘↵", func() { app.setStatus("open and send: " + req.name) }).
			AltLabel("Open and send"))
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
			Icon(icons.Sun).
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
		ui.Cmd("view.density.compact").Title("Density: Compact").Group("View").Icon(icons.Minimize2).
			Run(func() { app.setDensity(theme.DensityCompact, "compact") }),
		ui.Cmd("view.density.default").Title("Density: Default").Group("View").Icon(icons.Square).
			Run(func() { app.setDensity(theme.DensityDefault, "default") }),
		ui.Cmd("view.density.comfortable").Title("Density: Comfortable").Group("View").Icon(icons.Maximize2).
			Run(func() { app.setDensity(theme.DensityComfortable, "comfortable") }),
		ui.Cmd("view.scrollbars").Title("Toggle Overlay Scrollbars").Group("View").Icon(icons.PanelRight).
			Run(func() {
				mode := theme.ScrollOverlay
				if theme.Current().Metrics.ScrollbarMode == theme.ScrollOverlay {
					mode = theme.ScrollAlways
				}
				theme.UseScrollbarMode(mode)
				app.setStatus("overlay scrollbars: " + strconv.FormatBool(mode == theme.ScrollOverlay))
			}),
		ui.Cmd("toast.info").
			Title("Show Info Toast").
			Group("Feedback").
			Icon(icons.Bell).
			Run(func() {
				c.Toasts().Show("Info from command palette", ui.ToastInfo, 3*time.Second)
			}),
		ui.Cmd("toast.success").
			Title("Show Success Toast").
			Group("Feedback").
			Icon(icons.Check).
			Run(func() {
				c.Toasts().Show("Success from command palette", ui.ToastSuccess, 3*time.Second)
			}),
	)
	c.Commands().Register(cmds...)
	density := ui.DensityDefault
	if app.paletteCompact {
		density = ui.DensityCompact
	}
	c.Commands().Density(density).HintFooter(app.paletteCompact)
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
	themes := make([]ui.SelectOption, 0, len(theme.Names()))
	for _, name := range theme.Names() {
		n := name
		themes = append(themes, ui.SelectOption{Label: n, Value: n})
	}
	return ui.TitleBar(
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
			IconStart(icons.Search).
			Hint(c.Commands().ToggleLabel()).
			OnClick(func() {
				app.paletteCompact = false
				c.Commands().Show()
			}),
		ui.Spacer(),
		ui.Select("theme", themes).
			Width(180).
			Selected(themeIndex(app.theme)).
			OnChange(func(v string) {
				theme.Use(v)
				app.theme = v
				app.formTheme = v
			}),
	)
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
		ui.Icon(icons.Yoga, th.Metrics.IconSizeMD, th.Accent),
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
	case "images":
		return app.pageImages(c)
	case "svg":
		return app.pageSVG(c)
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
	case "drawer":
		return app.pageDrawer(c)
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
