// Command example is a functional demo of the framework: a dark-themed coding
// workspace with a recursive file explorer, a closable tab bar, and multiple
// editable, syntax-highlighted code editors backed by piece tables.
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/mirzakhany/yoga"
	"github.com/mirzakhany/yoga/components"
	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/lsp"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
	"github.com/mirzakhany/yoga/theme"
	"github.com/mirzakhany/yoga/ui"
)

// EditorPage is the code-editor workspace demo page.
type EditorPage struct {
	root *layout.Element

	docs   []*components.Editor
	active int

	tabs       *components.TabBar
	tree       *components.FileTree
	search     *components.TextField
	editorHost *layout.Element
	lspOverlay *layout.Element // hosts the active editor's completion/hover UI

	menus []*components.Dropdown

	// Editor font settings, applied live via yoga.SetFont.
	fontSize      float32
	letterSpacing float32
	lineHeight    float32

	status       string
	lastW, lastH float32
}

// Default editor font settings: roomier spacing and line height than the bare
// font metrics, which look cramped at small sizes.
const (
	defaultFontSize      = 14
	defaultLetterSpacing = 0.5
	defaultLineHeight    = 1.0
)

func buildEditorPage(_ *components.DialogHost, _ *components.ToastHost) *EditorPage {
	th := theme.Current()

	// gopls (.go) is built into the lsp package; register any extra servers the
	// demo wants here. The JSON server ships in the npm package
	// vscode-langservers-extracted (`npm i -g vscode-langservers-extracted`).
	// If its binary is not on PATH, .json files simply open without LSP.
	lsp.Register(".json", lsp.ServerConfig{
		LanguageID: "json",
		Command:    "vscode-json-language-server",
		Args:       []string{"--stdio"},
	})

	ws := &EditorPage{
		fontSize:      defaultFontSize,
		letterSpacing: defaultLetterSpacing,
		lineHeight:    defaultLineHeight,
	}
	// Apply the editor font before constructing editors so they cache the
	// correct line metrics from the first frame.
	_ = yoga.SetFont(ws.fontConfig())

	ws.tabs = components.NewTabBar()
	ws.tabs.OnActivate = func(i int) { ws.setActive(i) }
	ws.tabs.OnClose = func(i int) { ws.closeTab(i) }
	ws.editorHost = layout.New(layout.Box().FlexGrow(1))

	welcome := components.NewEditorFor("welcome.go", sampleSource)
	ws.docs = append(ws.docs, welcome)
	ws.tabs.Tabs = append(ws.tabs.Tabs, components.TabModel{Title: "welcome.go"})
	ws.bindActive(0)

	cwd, err := os.Getwd()
	if err != nil || cwd == "" {
		cwd = "."
	}
	ws.tree = components.NewFileTree(cwd)
	ws.tree.OnOpenFile = func(path string) { ws.openFile(path) }
	ws.tree.OnChange = func() { ws.relayout() }
	ws.tree.SetContextMenu(func(path string) []components.MenuItem {
		if path == "" {
			return nil
		}
		return []components.MenuItem{
			{Label: "Open", OnSelect: func() { ws.openFile(path) }},
			{Label: "Copy Path", OnSelect: func() {
				yoga.Clipboard().Set(path)
				ws.status = "copied: " + path
			}},
		}
	})

	ws.search = components.NewTextField(components.TextFieldConfig{Placeholder: "Search files..."}).
		WithIconStart("search").
		Changed(func(q string) {
			ws.tree.SetFilter(q)
			ws.relayout()
		})

	explorer := layout.New(layout.Box(),
		sidebarHeader(th, "EXPLORER"),
		ws.search.El,
		ws.tree.El(),
	).WithBackgroundPtr(&th.Panel)

	fileMenu := components.NewDropdown("File", 160, []components.MenuItem{
		{Label: "Save", OnSelect: func() { ws.save() }},
		{Label: "Close Tab", OnSelect: func() { ws.closeTab(ws.active) }},
	})
	editMenu := components.NewDropdown("Edit", 160, []components.MenuItem{
		{Label: "Undo", OnSelect: func() { ws.active2().Undo() }},
		{Label: "Redo", OnSelect: func() { ws.active2().Redo() }},
	})
	var themeItems []components.MenuItem
	for _, name := range theme.Names() {
		themeItems = append(themeItems, components.MenuItem{Label: name, OnSelect: func() {
			theme.Use(name)
			ws.status = "theme: " + name
		}})
	}
	themeMenu := components.NewDropdown("Theme", 180, themeItems)
	viewMenu := components.NewDropdown("View", 200, []components.MenuItem{
		{Label: "Increase Font Size", OnSelect: func() { ws.adjustFontSize(1) }},
		{Label: "Decrease Font Size", OnSelect: func() { ws.adjustFontSize(-1) }},
		{Label: "Cycle Line Spacing", OnSelect: func() { ws.cycleLineSpacing() }},
		{Label: "Cycle Letter Spacing", OnSelect: func() { ws.cycleLetterSpacing() }},
		{Label: "Reset Font", OnSelect: func() { ws.resetFont() }},
	})
	ws.menus = []*components.Dropdown{fileMenu, editMenu, themeMenu, viewMenu}

	topBar := layout.New(
		layout.Box().Direction(layout.Row).H(36).AlignItems(layout.AlignCenter).PaddingXY(8, 0),
		fileMenu.Button.El,
		editMenu.Button.El,
		themeMenu.Button.El,
		viewMenu.Button.El,
		layout.Spacer(),
		components.NewIcon("circle", 12, th.Accent),
	).WithBackgroundPtr(&th.Panel)

	editorColumn := layout.New(layout.Box().Direction(layout.Column).FlexGrow(1),
		ws.tabs.El,
		ws.editorHost,
	)
	split := components.NewSplitter(components.Horizontal,
		components.SplitSection{El: explorer, Size: 240},
		components.SplitSection{El: editorColumn, Size: 0},
	)

	statusBar := layout.New(layout.Box().H(22))
	statusBar.Paint = func(dl *render.DrawList, eng *shape.Engine) {
		dl.AddRect(statusBar.Frame, th.PanelAlt)
		_, sh := eng.Measure(ws.status)
		eng.DrawStringTop(dl, ws.status, statusBar.Frame.X+10, statusBar.Frame.Y+(statusBar.Frame.H-sh)/2, th.TextDim)
	}

	ws.root = layout.New(layout.Box().Direction(layout.Column).FlexGrow(1),
		topBar,
		split.El,
		statusBar,
	).WithBackgroundPtr(&th.Background)

	// A single overlay element delegates to whichever editor is active, so the
	// completion popup and hover tooltip paint on top of (and outside) the
	// clipped editor viewport. Mounted once at the app root via overlayEls.
	ws.lspOverlay = layout.New(layout.Box())
	ws.lspOverlay.Overlay = true
	ws.lspOverlay.Paint = func(dl *render.DrawList, eng *shape.Engine) {
		ws.active2().PaintLSPOverlay(dl, eng)
	}
	ws.lspOverlay.OnMouse = func(el *layout.Element, m *input.Mouse) {
		ws.active2().LSPOverlayMouse(el, m)
	}

	ws.status = ws.statusText()
	return ws
}

// Layout is the ui.View entry point for the editor page. It registers the
// page's focusables and overlays through the frame context, advances per-frame
// component work, handles the Ctrl+S shortcut, and returns the retained tree
// (which the runtime re-solves every frame).
func (ws *EditorPage) Layout(c *ui.Ctx) *ui.Element {
	m := c.Mouse()

	// Calling each component's Layout(c) registers focus (and the editor's
	// caret-blink animation / the file tree's context-menu overlay).
	ws.search.Layout(c)
	ws.tree.Layout(c)
	ws.tabs.Layout(c)
	ws.active2().Layout(c)
	c.Focus().EnsureFocus(ws.active2())

	// Per-frame component work (drag continuation, highlight polling, caret).
	ws.active2().Update(m)
	ws.search.Update(m)
	ws.tree.Update(m)

	// Dropdown menus self-register their overlays; the lsp popup mounts last so
	// it paints above the menus.
	for _, mn := range ws.menus {
		mn.Layout(c)
	}
	c.Overlay(ws.lspOverlay)

	if kb := c.Keyboard(); kb != nil {
		for _, ev := range kb.Keys {
			if ev.Mods.Primary() && ev.Key == input.KeyS {
				ws.save()
			}
		}
	}
	for i, d := range ws.docs {
		ws.tabs.Tabs[i].Modified = d.Modified()
	}
	ws.status = ws.statusText()
	return ws.root
}

func sidebarHeader(th *theme.Theme, title string) *layout.Element {
	el := layout.New(layout.Box().H(28))
	el.Paint = func(dl *render.DrawList, text *shape.Engine) {
		_, sh := text.Measure(title)
		text.DrawStringTop(dl, title, el.Frame.X+12, el.Frame.Y+(el.Frame.H-sh)/2, th.TextDim)
	}
	return el
}

func (ws *EditorPage) active2() *components.Editor { return ws.docs[ws.active] }

func (ws *EditorPage) bindActive(i int) {
	for j, d := range ws.docs {
		if j != i {
			d.Blur()
		}
	}
	ws.active = i
	ws.tabs.Active = i
	ws.editorHost.Children = []*layout.Element{ws.docs[i].El}
}

func (ws *EditorPage) setActive(i int) {
	if i < 0 || i >= len(ws.docs) {
		return
	}
	ws.bindActive(i)
	// Focus follows the active editor automatically via EnsureFocus in Layout.
	ws.relayout()
	ws.status = ws.statusText()
}

func (ws *EditorPage) openFile(path string) {
	for i, d := range ws.docs {
		if d.Path == path {
			ws.setActive(i)
			return
		}
	}
	content, err := os.ReadFile(path)
	if err != nil {
		ws.status = "open failed: " + err.Error()
		return
	}
	ed := components.NewEditorFor(path, content)
	ws.docs = append(ws.docs, ed)
	ws.tabs.Tabs = append(ws.tabs.Tabs, components.TabModel{Title: filepath.Base(path)})
	ws.setActive(len(ws.docs) - 1)
}

func (ws *EditorPage) closeTab(i int) {
	if i < 0 || i >= len(ws.docs) {
		return
	}
	ws.docs[i].Close()
	ws.docs = append(ws.docs[:i], ws.docs[i+1:]...)
	ws.tabs.Tabs = append(ws.tabs.Tabs[:i], ws.tabs.Tabs[i+1:]...)
	if len(ws.docs) == 0 {
		scratch := components.NewEditorFor("", nil)
		ws.docs = append(ws.docs, scratch)
		ws.tabs.Tabs = append(ws.tabs.Tabs, components.TabModel{Title: "untitled"})
	}
	next := ws.active
	if next >= len(ws.docs) {
		next = len(ws.docs) - 1
	}
	ws.setActive(next)
}

func (ws *EditorPage) save() {
	ed := ws.active2()
	if ed.Path == "" {
		ws.status = "cannot save: untitled buffer"
		return
	}
	if err := os.WriteFile(ed.Path, ed.Bytes(), 0o644); err != nil {
		ws.status = "save failed: " + err.Error()
		return
	}
	ed.MarkSaved()
	ws.status = "saved " + ed.Path
}

// relayout marks the tree dirty; the runtime re-solves it on the next frame
// (the build loop calls Calculate every frame), so no manual solve is needed.
func (ws *EditorPage) relayout() { ws.root.MarkDirty() }

// fontConfig builds the engine font configuration from the current settings.
func (ws *EditorPage) fontConfig() shape.FontConfig {
	return shape.FontConfig{
		UI: shape.FaceConfig{Size: defaultFontSize},
		Mono: shape.FaceConfig{
			Size:          ws.fontSize,
			LetterSpacing: ws.letterSpacing,
			LineHeight:    ws.lineHeight,
		},
		TabWidth: 4,
	}
}

// applyFont pushes the current settings to the engine; editors refresh on their
// next Update via the engine's font generation counter.
func (ws *EditorPage) applyFont() {
	if err := yoga.SetFont(ws.fontConfig()); err != nil {
		ws.status = "font error: " + err.Error()
		return
	}
	ws.status = fmt.Sprintf("font %.0fpx · letter %.1f · line %.1f×",
		ws.fontSize, ws.letterSpacing, ws.lineHeight)
	ws.relayout()
}

func (ws *EditorPage) adjustFontSize(delta float32) {
	ws.fontSize = clampf(ws.fontSize+delta, 9, 32)
	ws.applyFont()
}

// cycleLineSpacing steps the line-height multiplier through a few presets.
func (ws *EditorPage) cycleLineSpacing() {
	switch {
	case ws.lineHeight < 1.45:
		ws.lineHeight = 1.6
	case ws.lineHeight < 1.75:
		ws.lineHeight = 2.0
	default:
		ws.lineHeight = 1.0
	}
	ws.applyFont()
}

// cycleLetterSpacing steps the per-glyph tracking through a few presets.
func (ws *EditorPage) cycleLetterSpacing() {
	switch {
	case ws.letterSpacing < 0.25:
		ws.letterSpacing = 0.5
	case ws.letterSpacing < 0.75:
		ws.letterSpacing = 1.5
	default:
		ws.letterSpacing = 1.2
	}
	ws.applyFont()
}

func (ws *EditorPage) resetFont() {
	ws.fontSize = defaultFontSize
	ws.letterSpacing = defaultLetterSpacing
	ws.lineHeight = defaultLineHeight
	ws.applyFont()
}

func clampf(v, lo, hi float32) float32 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func (ws *EditorPage) statusText() string {
	ed := ws.active2()
	name := ed.Path
	if name == "" {
		name = "untitled"
	}
	mark := ""
	if ed.Modified() {
		mark = " *"
	}
	return name + mark + "  —  Go  —  UTF-8"
}

func (ws *EditorPage) close() {
	for _, d := range ws.docs {
		d.Close()
	}
}

func (ws *EditorPage) Close() { ws.close() }

// BuildWorkspace preserves the headless entry API.
func BuildWorkspace() *EditorPage {
	return buildEditorPage(nil, nil)
}
