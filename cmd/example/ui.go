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
	"time"

	"github.com/mirzakhany/yoga"
	"github.com/mirzakhany/yoga/components"
	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
	"github.com/mirzakhany/yoga/theme"
)

// Workspace implements yoga.Scene so it can be driven directly by the runtime.
var _ yoga.Scene = (*Workspace)(nil)

// Workspace bundles the UI tree and the stateful widgets that need per-frame
// updates. It owns one Editor per open document; only the active document's
// Element is mounted into editorHost at a time, and the single shared Scrollbar
// is rebound to whichever document is active.
type Workspace struct {
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

	menus  []*components.Dropdown
	status string

	// lastW/H record the most recent viewport so structural changes (open/close
	// a tab, expand a folder) can re-solve the layout immediately, in the same
	// frame, avoiding a one-frame glitch.
	lastW, lastH float32
}

// BuildWorkspace assembles the whole UI declaratively. clip is the system
// clipboard adapter (GLFW-backed in the GPU build, in-memory headless). The UI
// reads its colors from the live active theme (theme.Current()), so a runtime
// theme switch is reflected immediately with no rebuild.
func BuildWorkspace(text *shape.Engine, clip input.Clipboard) *Workspace {
	th := theme.Current()
	ws := &Workspace{theme: th, text: text, clip: clip}
	sheet := render.NewSpriteSheet(text.Atlas)

	// --- Tabs + editor host (active document is mounted here) ---
	ws.tabs = components.NewTabBar(text, th)
	ws.tabs.OnActivate = func(i int) { ws.setActive(i) }
	ws.tabs.OnClose = func(i int) { ws.closeTab(i) }

	ws.editorHost = layout.New(layout.Box().FlexGrow(1))

	// Seed one editable welcome buffer.
	welcome := components.NewEditorFor(text, th, "welcome.go", sampleSource, clip)
	ws.docs = append(ws.docs, welcome)
	ws.tabs.Tabs = append(ws.tabs.Tabs, components.TabModel{Title: "welcome.go"})
	ws.bindActive(0)

	// --- File explorer rooted at the launch directory ---
	cwd, err := os.Getwd()
	if err != nil || cwd == "" {
		cwd = "."
	}
	ws.tree = components.NewFileTree(text, th, sheet, cwd)
	ws.tree.OnOpenFile = func(path string) { ws.openFile(path) }
	ws.tree.OnChange = func() { ws.relayout() }
	// Demonstrate per-item customization: a right-click context menu on files.
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
		Radius:      4,
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

	// --- Top menu bar with dropdowns + an icon ---
	fileMenu := components.NewDropdown(text, th, "File", 160, []components.MenuItem{
		{Label: "Save", OnSelect: func() { ws.save() }},
		{Label: "Close Tab", OnSelect: func() { ws.closeTab(ws.active) }},
		{Label: "Quit", OnSelect: func() { ws.status = "Quit (close window)" }},
	})
	editMenu := components.NewDropdown(text, th, "Edit", 160, []components.MenuItem{
		{Label: "Undo", OnSelect: func() { ws.active2().Undo() }},
		{Label: "Redo", OnSelect: func() { ws.active2().Redo() }},
	})
	// Theme switcher: one item per registered theme; selecting one switches the
	// live active theme instantly.
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
		layout.New(layout.Box().FlexGrow(1)), // flexible spacer
		components.NewIcon(sheet, "circle", 12, th.Accent),
	).WithBackgroundPtr(&th.Panel)

	// --- Main content row: explorer | (tabs over editor host) ---
	editorColumn := layout.New(layout.Box().Direction(layout.Column).FlexGrow(1),
		ws.tabs.El,
		ws.editorHost,
	)
	split := components.NewSplitter(th, components.Horizontal,
		components.SplitSection{El: explorer, Size: 240},
		components.SplitSection{El: editorColumn, Size: 0},
	)
	mainRow := split.El

	// --- Status bar ---
	statusBar := layout.New(layout.Box().H(22))
	statusBar.Paint = func(dl *render.DrawList, text *shape.Engine) {
		dl.AddRect(statusBar.Frame, th.PanelAlt)
		_, sh := text.Measure(ws.status)
		text.DrawStringTop(dl, ws.status, statusBar.Frame.X+10, statusBar.Frame.Y+(statusBar.Frame.H-sh)/2, th.TextDim)
	}

	// --- Root: vertical stack + overlay menus pinned at the root so their
	// absolute coordinates are screen-space. ---
	ws.root = layout.New(layout.Box().Direction(layout.Column),
		topBar,
		mainRow,
		statusBar,
		fileMenu.Menu.El,
		editMenu.Menu.El,
		themeMenu.Menu.El,
		ws.tree.MenuEl(),
	).WithBackgroundPtr(&th.Background)

	ws.status = ws.statusText()
	return ws
}

func sidebarHeader(th *theme.Theme, title string) *layout.Element {
	el := layout.New(layout.Box().H(28))
	el.Paint = func(dl *render.DrawList, text *shape.Engine) {
		_, sh := text.Measure(title)
		text.DrawStringTop(dl, title, el.Frame.X+12, el.Frame.Y+(el.Frame.H-sh)/2, th.TextDim)
	}
	return el
}

// active2 returns the active editor (always valid: there is always >= 1 doc).
func (ws *Workspace) active2() *components.Editor { return ws.docs[ws.active] }

// bindActive mounts document i into editorHost (editor owns its scrollbars).
func (ws *Workspace) bindActive(i int) {
	ws.active = i
	ws.tabs.Active = i
	ed := ws.docs[i]
	ws.editorHost.Children = []*layout.Element{ed.El}
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
	ed := components.NewEditorFor(ws.text, ws.theme, path, content, ws.clip)
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
	ws.root.MarkDirty()
	if ws.lastW > 0 && ws.lastH > 0 {
		ws.root.Calculate(ws.lastW, ws.lastH)
	}
}

// Root returns the root element of the UI tree (satisfies yoga.Scene).
func (ws *Workspace) Root() *layout.Element { return ws.root }

// ClearColor reports the framebuffer clear color for this frame. The yoga
// runtime reads it each frame, so a theme switch updates the window background
// immediately. (Optional Scene capability discovered via interface assertion.)
func (ws *Workspace) ClearColor() render.Color { return theme.Current().Background }

// AnimationWait implements yoga.Animator so the runtime can sleep between frames
// instead of busy-rendering. It delegates to the active editor (caret blink and
// post-edit highlight cadence); there is always exactly one active editor.
func (ws *Workspace) AnimationWait() (time.Duration, bool) { return ws.active2().AnimationWait() }

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
	ws.root.Calculate(w, h)
}

// Update runs one frame of state updates. It must be called AFTER Layout has
// computed Frames for this frame (the scrollbar and dispatch need geometry).
func (ws *Workspace) Update(m *input.Mouse, kb *input.Keyboard) {
	ed := ws.active2()

	// Mouse dispatch may switch the active document (tab click / file open).
	layout.Dispatch(ws.root, m)

	if m.Pressed && !ws.search.El.Frame.Contains(m.X, m.Y) {
		ws.search.Blur()
	}

	if ws.search.Focused() {
		if kb != nil {
			ws.search.HandleText(kb.Chars)
			ws.search.HandleKeys(kb.Keys)
		}
	} else {
		if kb != nil {
			ws.handleShortcuts(kb)
			ed.HandleText(kb.Chars)
			ed.HandleKeys(kb.Keys)
		}
	}

	ed = ws.active2()
	ed.Update(m)
	ws.search.Update(m)
	ws.tree.Update(m) // drive the file tree's own scrollbars

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
