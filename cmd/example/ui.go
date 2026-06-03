// Command example is a functional demo of the framework: a dark-themed coding
// workspace with a recursive file explorer, a closable tab bar, and multiple
// editable, syntax-highlighted code editors backed by piece tables.
//
// This file (ui.go) contains the platform-independent UI construction and
// per-frame update logic, with no GLFW/WebGPU imports, so it compiles in both
// the GPU build (main_gpu.go) and the headless build (main_headless.go).
package main

import (
	"os"
	"path/filepath"

	"github.com/mirzakhany/yoga/components"
	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
)

// Workspace bundles the UI tree and the stateful widgets that need per-frame
// updates. It owns one Editor per open document; only the active document's
// Element is mounted into editorHost at a time, and the single shared Scrollbar
// is rebound to whichever document is active.
type Workspace struct {
	Root  *layout.Element
	theme components.Theme
	atlas *render.FontAtlas
	clip  input.Clipboard

	docs   []*components.Editor
	active int

	tabs       *components.TabBar
	tree       *components.FileTree
	scrollbar  *components.Scrollbar
	editorHost *layout.Element

	menus  []*components.Dropdown
	status string

	// lastW/H record the most recent viewport so structural changes (open/close
	// a tab, expand a folder) can re-solve the layout immediately, in the same
	// frame, avoiding a one-frame glitch.
	lastW, lastH float32
}

// BuildWorkspace assembles the whole UI declaratively. clip is the system
// clipboard adapter (GLFW-backed in the GPU build, in-memory headless).
func BuildWorkspace(atlas *render.FontAtlas, theme components.Theme, clip input.Clipboard) *Workspace {
	ws := &Workspace{theme: theme, atlas: atlas, clip: clip}
	sheet := render.NewSpriteSheet(atlas)

	// --- Tabs + editor host (active document is mounted here) ---
	ws.tabs = components.NewTabBar(atlas, theme)
	ws.tabs.OnActivate = func(i int) { ws.setActive(i) }
	ws.tabs.OnClose = func(i int) { ws.closeTab(i) }

	ws.editorHost = layout.New(layout.Box().Grow_(1))
	ws.scrollbar = components.NewScrollbar(theme, new(float32), new(float32), 12)

	// Seed one editable welcome buffer.
	welcome := components.NewEditorFor(atlas, theme, "welcome.go", sampleSource, clip)
	ws.docs = append(ws.docs, welcome)
	ws.tabs.Tabs = append(ws.tabs.Tabs, components.TabModel{Title: "welcome.go"})
	ws.bindActive(0)

	// --- File explorer rooted at the launch directory ---
	cwd, err := os.Getwd()
	if err != nil || cwd == "" {
		cwd = "."
	}
	ws.tree = components.NewFileTree(atlas, theme, sheet, cwd)
	ws.tree.OnOpenFile = func(path string) { ws.openFile(path) }
	ws.tree.OnChange = func() { ws.relayout() }

	explorer := layout.New(layout.Box().W(240),
		sidebarHeader(atlas, theme, "EXPLORER"),
		ws.tree.El,
	).WithBackground(theme.Panel)

	// --- Top menu bar with dropdowns + an icon ---
	fileMenu := components.NewDropdown(atlas, theme, "File", 160, []components.MenuItem{
		{Label: "Save", OnSelect: func() { ws.save() }},
		{Label: "Close Tab", OnSelect: func() { ws.closeTab(ws.active) }},
		{Label: "Quit", OnSelect: func() { ws.status = "Quit (close window)" }},
	})
	editMenu := components.NewDropdown(atlas, theme, "Edit", 160, []components.MenuItem{
		{Label: "Undo", OnSelect: func() { ws.active2().Undo() }},
		{Label: "Redo", OnSelect: func() { ws.active2().Redo() }},
	})
	ws.menus = []*components.Dropdown{fileMenu, editMenu}

	topBar := layout.New(
		layout.Box().Direction(layout.Row).H(36).AlignItems(layout.AlignCenter).PaddingXY(8, 0),
		fileMenu.Button.El,
		editMenu.Button.El,
		layout.New(layout.Box().Grow_(1)), // flexible spacer
		components.NewIcon(sheet, "circle", 12, theme.Accent),
	).WithBackground(theme.Panel)

	// --- Main content row: explorer | (tabs over editor host) ---
	editorColumn := layout.New(layout.Box().Direction(layout.Column).Grow_(1),
		ws.tabs.El,
		ws.editorHost,
	)
	mainRow := layout.New(layout.Box().Direction(layout.Row).Grow_(1),
		explorer,
		editorColumn,
	)

	// --- Status bar ---
	statusBar := layout.New(layout.Box().H(22))
	statusBar.Paint = func(dl *render.DrawList, a *render.FontAtlas) {
		dl.AddRect(statusBar.Frame, theme.PanelAlt)
		_, th := a.Measure(ws.status)
		a.DrawText(dl, ws.status, statusBar.Frame.X+10, statusBar.Frame.Y+(statusBar.Frame.H-th)/2, theme.TextDim)
	}

	// --- Root: vertical stack + overlay menus pinned at the root so their
	// absolute coordinates are screen-space. ---
	ws.Root = layout.New(layout.Box().Direction(layout.Column),
		topBar,
		mainRow,
		statusBar,
		fileMenu.Menu.El,
		editMenu.Menu.El,
	).WithBackground(theme.Background)

	ws.status = ws.statusText()
	return ws
}

func sidebarHeader(atlas *render.FontAtlas, theme components.Theme, title string) *layout.Element {
	el := layout.New(layout.Box().H(28))
	el.Paint = func(dl *render.DrawList, a *render.FontAtlas) {
		_, th := a.Measure(title)
		a.DrawText(dl, title, el.Frame.X+12, el.Frame.Y+(el.Frame.H-th)/2, theme.TextDim)
	}
	return el
}

// active2 returns the active editor (always valid: there is always >= 1 doc).
func (ws *Workspace) active2() *components.Editor { return ws.docs[ws.active] }

// bindActive mounts document i into editorHost and rebinds the shared scrollbar
// to it, without triggering a relayout (used during initial build).
func (ws *Workspace) bindActive(i int) {
	ws.active = i
	ws.tabs.Active = i
	ed := ws.docs[i]
	ws.scrollbar.Offset = &ed.ScrollPx
	ws.scrollbar.ContentHeight = &ed.ContentHeight
	ws.editorHost.Children = []*layout.Element{ed.El, ws.scrollbar.El}
}

// setActive switches the visible document and re-solves the layout now.
func (ws *Workspace) setActive(i int) {
	if i < 0 || i >= len(ws.docs) {
		return
	}
	ws.bindActive(i)
	ws.relayout()
	ws.status = ws.statusText()
}

// openFile opens path in a new tab (or activates it if already open).
func (ws *Workspace) openFile(path string) {
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
	ed := components.NewEditorFor(ws.atlas, ws.theme, path, content, ws.clip)
	ws.docs = append(ws.docs, ed)
	ws.tabs.Tabs = append(ws.tabs.Tabs, components.TabModel{Title: filepath.Base(path)})
	ws.setActive(len(ws.docs) - 1)
}

// closeTab closes document i, releasing its highlighter. A final tab close is
// replaced by a fresh empty scratch buffer so there is always something to edit.
func (ws *Workspace) closeTab(i int) {
	if i < 0 || i >= len(ws.docs) {
		return
	}
	ws.docs[i].Close()
	ws.docs = append(ws.docs[:i], ws.docs[i+1:]...)
	ws.tabs.Tabs = append(ws.tabs.Tabs[:i], ws.tabs.Tabs[i+1:]...)

	if len(ws.docs) == 0 {
		scratch := components.NewEditorFor(ws.atlas, ws.theme, "", nil, ws.clip)
		ws.docs = append(ws.docs, scratch)
		ws.tabs.Tabs = append(ws.tabs.Tabs, components.TabModel{Title: "untitled"})
	}

	next := ws.active
	if next >= len(ws.docs) {
		next = len(ws.docs) - 1
	}
	ws.setActive(next)
}

// save writes the active document to its path (no-op for an untitled buffer).
func (ws *Workspace) save() {
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

// relayout drops cached layout nodes and re-solves immediately at the last known
// viewport so structural edits take effect within the same frame.
func (ws *Workspace) relayout() {
	ws.Root.MarkDirty()
	if ws.lastW > 0 && ws.lastH > 0 {
		ws.Root.Calculate(ws.lastW, ws.lastH)
	}
}

func (ws *Workspace) statusText() string {
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

// Layout records the viewport and runs the two-pass layout pipeline. Call it
// once per frame before Update/Paint.
func (ws *Workspace) Layout(w, h float32) {
	ws.lastW, ws.lastH = w, h
	ws.Root.Calculate(w, h)
}

// Update runs one frame of state updates. It must be called AFTER Layout has
// computed Frames for this frame (the scrollbar and dispatch need geometry).
func (ws *Workspace) Update(m *input.Mouse, kb *input.Keyboard) {
	ed := ws.active2()

	if kb != nil {
		ws.handleShortcuts(kb)
		ed.HandleText(kb.Chars)
		ed.HandleKeys(kb.Keys)
	}
	ed.Update()

	// Mouse dispatch may switch the active document (tab click / file open),
	// which rebinds the scrollbar and re-solves the layout.
	layout.Dispatch(ws.Root, m)
	ed = ws.active2()
	ws.scrollbar.Update(m, ed.El.Frame)

	for i, d := range ws.docs {
		ws.tabs.Tabs[i].Modified = d.Modified()
	}
	ws.status = ws.statusText()
}

// handleShortcuts processes workspace-level keyboard shortcuts (currently
// Cmd/Ctrl+S to save). The active editor handles editing shortcuts itself.
func (ws *Workspace) handleShortcuts(kb *input.Keyboard) {
	for _, ev := range kb.Keys {
		if ev.Mods.Primary() && ev.Key == input.KeyS {
			ws.save()
		}
	}
}

// Close releases worker goroutines / native resources owned by the UI.
func (ws *Workspace) Close() {
	for _, d := range ws.docs {
		d.Close()
	}
}
