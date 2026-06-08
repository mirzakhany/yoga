// Command example is a functional demo of the framework: a dark-themed coding
// workspace with a recursive file explorer, a closable tab bar, and multiple
// editable, syntax-highlighted code editors backed by piece tables.
package main

import (
	"os"
	"path/filepath"
	"time"

	"github.com/mirzakhany/yoga/components"
	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
	"github.com/mirzakhany/yoga/theme"
)

// EditorPage is the code-editor workspace demo page.
type EditorPage struct {
	root  *layout.Element
	theme *theme.Theme
	text  *shape.Engine
	clip  input.Clipboard

	docs   []*components.Editor
	active int

	tabs       *components.TabBar
	tree       *components.FileTree
	search     *components.TextField
	editorHost *layout.Element

	menus []*components.Dropdown
	focus *components.FocusManager
	status string
	lastW, lastH float32
}

func buildEditorPage(text *shape.Engine, clip input.Clipboard, sheet *render.SpriteSheet, _ *components.DialogHost, _ *components.ToastHost) *EditorPage {
	th := theme.Current()
	ws := &EditorPage{theme: th, text: text, clip: clip}

	ws.tabs = components.NewTabBar(text, th)
	ws.tabs.OnActivate = func(i int) { ws.setActive(i) }
	ws.tabs.OnClose = func(i int) { ws.closeTab(i) }
	ws.editorHost = layout.New(layout.Box().FlexGrow(1))

	welcome := components.NewEditorFor(text, th, "welcome.go", sampleSource, clip)
	ws.docs = append(ws.docs, welcome)
	ws.tabs.Tabs = append(ws.tabs.Tabs, components.TabModel{Title: "welcome.go"})
	ws.bindActive(0)

	cwd, err := os.Getwd()
	if err != nil || cwd == "" {
		cwd = "."
	}
	ws.tree = components.NewFileTree(text, th, sheet, cwd)
	ws.tree.OnOpenFile = func(path string) { ws.openFile(path) }
	ws.tree.OnChange = func() { ws.relayout() }
	ws.tree.SetContextMenu(func(path string) []components.MenuItem {
		if path == "" {
			return nil
		}
		return []components.MenuItem{
			{Label: "Open", OnSelect: func() { ws.openFile(path) }},
			{Label: "Copy Path", OnSelect: func() {
				ws.clip.Set(path)
				ws.status = "copied: " + path
			}},
		}
	})

	ws.search = components.NewTextField(text, th, sheet, clip, components.TextFieldConfig{
		Placeholder: "Search files...",
		IconStart:   "search",
	})
	ws.search.OnChange = func(q string) {
		ws.tree.SetFilter(q)
		ws.relayout()
	}

	explorer := layout.New(layout.Box(),
		sidebarHeader(th, "EXPLORER"),
		ws.search.El,
		ws.tree.El(),
	).WithBackgroundPtr(&th.Panel)

	fileMenu := components.NewDropdown(text, th, "File", 160, []components.MenuItem{
		{Label: "Save", OnSelect: func() { ws.save() }},
		{Label: "Close Tab", OnSelect: func() { ws.closeTab(ws.active) }},
	})
	editMenu := components.NewDropdown(text, th, "Edit", 160, []components.MenuItem{
		{Label: "Undo", OnSelect: func() { ws.active2().Undo() }},
		{Label: "Redo", OnSelect: func() { ws.active2().Redo() }},
	})
	var themeItems []components.MenuItem
	for _, name := range theme.Names() {
		name := name
		themeItems = append(themeItems, components.MenuItem{Label: name, OnSelect: func() {
			theme.Use(name)
			ws.status = "theme: " + name
		}})
	}
	themeMenu := components.NewDropdown(text, th, "Theme", 180, themeItems)
	ws.menus = []*components.Dropdown{fileMenu, editMenu, themeMenu}

	topBar := layout.New(
		layout.Box().Direction(layout.Row).H(36).AlignItems(layout.AlignCenter).PaddingXY(8, 0),
		fileMenu.Button.El,
		editMenu.Button.El,
		themeMenu.Button.El,
		layout.New(layout.Box().FlexGrow(1)),
		components.NewIcon(sheet, "circle", 12, th.Accent),
	).WithBackgroundPtr(&th.Panel)

	editorColumn := layout.New(layout.Box().Direction(layout.Column).FlexGrow(1),
		ws.tabs.El,
		ws.editorHost,
	)
	split := components.NewSplitter(th, components.Horizontal,
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

	ws.focus = components.NewFocusManager()
	ws.focus.Add(ws.search, ws.tree, ws.tabs, ws.editorFocus())
	ws.focus.Focus(ws.editorFocus())
	ws.status = ws.statusText()
	return ws
}

func (ws *EditorPage) overlayEls() []*layout.Element {
	out := []*layout.Element{ws.tree.MenuEl()}
	for _, m := range ws.menus {
		out = append(out, m.Menu.El)
	}
	return out
}

type editorFocusProxy struct{ ws *EditorPage }

func (ws *EditorPage) editorFocus() components.Focusable {
	return editorFocusProxy{ws: ws}
}

func (p editorFocusProxy) editor() *components.Editor { return p.ws.active2() }
func (p editorFocusProxy) Focus()                     { p.editor().Focus() }
func (p editorFocusProxy) Blur()                      { p.editor().Blur() }
func (p editorFocusProxy) Focused() bool              { return p.editor().Focused() }
func (p editorFocusProxy) HandleText(r []rune)        { p.editor().HandleText(r) }
func (p editorFocusProxy) HandleKeys(k []input.KeyEvent) { p.editor().HandleKeys(k) }
func (p editorFocusProxy) CapturesTab() bool          { return true }
func (p editorFocusProxy) FocusOnClick() bool         { return true }
func (p editorFocusProxy) FocusEl() *layout.Element { return p.editor().El }

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
	if ws.focus != nil {
		ws.focus.Focus(ws.editorFocus())
	}
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
	ed := components.NewEditorFor(ws.text, ws.theme, path, content, ws.clip)
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
		scratch := components.NewEditorFor(ws.text, ws.theme, "", nil, ws.clip)
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

func (ws *EditorPage) relayout() {
	ws.root.MarkDirty()
	if ws.lastW > 0 && ws.lastH > 0 {
		ws.root.Calculate(ws.lastW, ws.lastH)
	}
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

func (ws *EditorPage) update(m *input.Mouse, kb *input.Keyboard) {
	ws.lastW = ws.root.Frame.W
	ws.lastH = ws.root.Frame.H
	layout.Dispatch(ws.root, m)
	if ws.focus != nil {
		ws.focus.HandleMouse(m)
	}
	if kb != nil {
		for _, ev := range kb.Keys {
			if ev.Mods.Primary() && ev.Key == input.KeyS {
				ws.save()
			}
		}
		if ws.focus != nil {
			ws.focus.Route(kb)
		}
	}
	ws.active2().Update(m)
	ws.search.Update(m)
	ws.tree.Update(m)
	for i, d := range ws.docs {
		ws.tabs.Tabs[i].Modified = d.Modified()
	}
	ws.status = ws.statusText()
}

func (ws *EditorPage) animationWait() (time.Duration, bool) {
	return ws.active2().AnimationWait()
}

func (ws *EditorPage) close() {
	for _, d := range ws.docs {
		d.Close()
	}
}

func (ws *EditorPage) Root() *layout.Element { return ws.root }

func (ws *EditorPage) Layout(w, h float32) {
	ws.lastW, ws.lastH = w, h
	ws.root.Calculate(w, h)
}

func (ws *EditorPage) Update(m *input.Mouse, kb *input.Keyboard) { ws.update(m, kb) }

func (ws *EditorPage) Close() { ws.close() }

// BuildWorkspace preserves the headless entry API.
func BuildWorkspace(text *shape.Engine, clip input.Clipboard) *EditorPage {
	sheet := render.NewSpriteSheet(text.Atlas)
	return buildEditorPage(text, clip, sheet, nil, nil)
}
