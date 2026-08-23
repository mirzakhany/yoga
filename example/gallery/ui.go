// Command example is a functional demo of the framework: a dark-themed coding
// workspace with a recursive file explorer, a closable tab bar, and multiple
// editable, syntax-highlighted code editors backed by piece tables.
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/mirzakhany/yoga"
	"github.com/mirzakhany/yoga/icons"
	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/lsp"
	"github.com/mirzakhany/yoga/shape"
	"github.com/mirzakhany/yoga/theme"
	"github.com/mirzakhany/yoga/ui"
)

// EditorPage is the code-editor workspace demo page.
type EditorPage struct {
	docs   []*ui.Editor
	active int
	tabs   []ui.TabModel
	tree   *ui.FileTree
	query  string

	// Editor font settings, applied live via yoga.SetFont.
	fontSize      float32
	letterSpacing float32
	lineHeight    float32

	status string
}

// Default editor font settings: roomier spacing and line height than the bare
// font metrics, which look cramped at small sizes.
const (
	defaultFontSize      = 14
	defaultLetterSpacing = 0.5
	defaultLineHeight    = 1.0
)

func buildEditorPage() *EditorPage {
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
	_ = yoga.SetFont(ws.fontConfig())

	welcome := ui.NewEditorFor("welcome.go", sampleSource)
	ws.docs = append(ws.docs, welcome)
	ws.tabs = append(ws.tabs, ui.TabModel{Title: "welcome.go"})
	ws.bindActive(0)

	cwd, err := os.Getwd()
	if err != nil || cwd == "" {
		cwd = "."
	}
	ws.tree = ui.NewFileTree(cwd)
	ws.tree.OnOpenFile = func(path string) { ws.openFile(path) }
	ws.tree.SetContextMenu(func(path string) []ui.MenuItem {
		if path == "" {
			return nil
		}
		return []ui.MenuItem{
			{Label: "Open", OnSelect: func() { ws.openFile(path) }},
			{Label: "Copy Path", OnSelect: func() {
				yoga.Clipboard().Set(path)
				ws.status = "copied: " + path
			}},
		}
	})
	ws.status = ws.statusText()
	return ws
}

func (ws *EditorPage) Layout(c *ui.Ctx) ui.View {
	if kb := c.Keyboard(); kb != nil {
		for _, ev := range kb.Keys {
			if ev.Mods.Primary() && ev.Key == input.KeyS {
				ws.save()
			}
		}
	}
	for i, d := range ws.docs {
		ws.tabs[i].Modified = d.Modified()
	}
	ws.status = ws.statusText()
	c.Focus().EnsureFocus(ws.activeDoc())

	th := c.Theme()
	var themeItems []ui.MenuItem
	for _, name := range theme.Names() {
		n := name
		themeItems = append(themeItems, ui.MenuItem{Label: n, OnSelect: func() {
			theme.Use(n)
			ws.status = "theme: " + n
		}})
	}
	top := ui.Row(
		ui.Dropdown("file-menu", "File", []ui.MenuItem{
			{Label: "Open…", OnSelect: func() { ws.openDialog(c) }},
			{Label: "Save", OnSelect: func() { ws.save() }},
			{Label: "Close Tab", OnSelect: func() { ws.closeTab(ws.active) }},
		}).Width(160),
		ui.Dropdown("edit-menu", "Edit", []ui.MenuItem{
			{Label: "Undo", OnSelect: func() { ws.activeDoc().Undo() }},
			{Label: "Redo", OnSelect: func() { ws.activeDoc().Redo() }},
		}).Width(160),
		ui.Dropdown("theme-menu", "Theme", themeItems).Width(180),
		ui.Dropdown("view-menu", "View", []ui.MenuItem{
			{Label: "Increase Font Size", OnSelect: func() { ws.adjustFontSize(1) }},
			{Label: "Decrease Font Size", OnSelect: func() { ws.adjustFontSize(-1) }},
			{Label: "Cycle Line Spacing", OnSelect: func() { ws.cycleLineSpacing() }},
			{Label: "Cycle Letter Spacing", OnSelect: func() { ws.cycleLetterSpacing() }},
			{Label: "Reset Font", OnSelect: func() { ws.resetFont() }},
		}).Width(200),
		ui.Spacer(),
		ui.Icon(icons.Circle, 12, th.Accent),
	).Height(36).PaddingXY(8, 0).Background(ui.TokenChrome)

	explorer := ui.Column(
		ui.Row(ui.Text("EXPLORER").Style(ui.Spec{}.TextColor(ui.TokenForegroundMuted))).
			Height(28).PaddingXY(12, 0),
		ui.TextField("file-search", ws.query).
			Placeholder("Search files...").
			IconStart(icons.Search).
			OnChange(func(q string) {
				ws.query = q
				ws.tree.SetFilter(q)
			}).
			PaddingXY(8, 4),
		ui.ViewOf(ws.tree).Grow(1),
	).Background(ui.TokenChrome)

	editorCol := ui.Column(
		ui.Tabs("editor-tabs", ws.tabs).
			Selected(ws.active).
			OnSelectItem(func(i int, _ string) { ws.setActive(i) }).
			OnTabClose(ws.closeTab),
		ui.ViewOf(ws.activeDoc()).Grow(1),
	).Grow(1)

	status := ui.Row(
		ui.Text(ws.status).Style(ui.Spec{}.TextColor(ui.TokenForegroundMuted)),
	).Height(22).PaddingXY(10, 0).Background(ui.TokenChromeMuted)

	return ui.Column(
		top,
		ui.Splitter("editor-split", ui.Horizontal, explorer, editorCol).Sizes(240, 0).Grow(1),
		status,
	).Grow(1).Background(ui.TokenSurface)
}

func (ws *EditorPage) activeDoc() *ui.Editor { return ws.docs[ws.active] }

func (ws *EditorPage) bindActive(i int) {
	for j, d := range ws.docs {
		if j != i {
			d.Blur()
		}
	}
	ws.active = i
}

func (ws *EditorPage) setActive(i int) {
	if i < 0 || i >= len(ws.docs) {
		return
	}
	ws.bindActive(i)
	ws.status = ws.statusText()
}

func (ws *EditorPage) openDialog(c *ui.Ctx) {
	dir := ""
	if ed := ws.activeDoc(); ed != nil && ed.Path != "" {
		dir = filepath.Dir(ed.Path)
	}
	c.Files().Show(ui.FileDialogOpts{
		Title: "Open File",
		Mode:  ui.FileDialogOpenFile,
		Dir:   dir,
		Filters: []ui.FileFilter{
			{Label: "Source", Exts: []string{".go", ".md", ".json", ".txt"}},
			{Label: "All files", Exts: nil},
		},
		OnConfirm: func(paths []string) {
			if len(paths) > 0 {
				ws.openFile(paths[0])
			}
		},
	})
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
	ed := ui.NewEditorFor(path, content)
	ws.docs = append(ws.docs, ed)
	ws.tabs = append(ws.tabs, ui.TabModel{Title: filepath.Base(path)})
	ws.setActive(len(ws.docs) - 1)
}

func (ws *EditorPage) closeTab(i int) {
	if i < 0 || i >= len(ws.docs) {
		return
	}
	ws.docs[i].Close()
	ws.docs = append(ws.docs[:i], ws.docs[i+1:]...)
	ws.tabs = append(ws.tabs[:i], ws.tabs[i+1:]...)
	if len(ws.docs) == 0 {
		scratch := ui.NewEditorFor("", nil)
		ws.docs = append(ws.docs, scratch)
		ws.tabs = append(ws.tabs, ui.TabModel{Title: "untitled"})
	}
	next := ws.active
	if next >= len(ws.docs) {
		next = len(ws.docs) - 1
	}
	ws.setActive(next)
}

func (ws *EditorPage) save() {
	ed := ws.activeDoc()
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

func (ws *EditorPage) applyFont() {
	if err := yoga.SetFont(ws.fontConfig()); err != nil {
		ws.status = "font error: " + err.Error()
		return
	}
	ws.status = fmt.Sprintf("font %.0fpx · letter %.1f · line %.1f×",
		ws.fontSize, ws.letterSpacing, ws.lineHeight)
}

func (ws *EditorPage) adjustFontSize(delta float32) {
	ws.fontSize = clampf(ws.fontSize+delta, 9, 32)
	ws.applyFont()
}

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
	ed := ws.activeDoc()
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
	return buildEditorPage()
}
