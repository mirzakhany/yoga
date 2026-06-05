# Yoga UI Framework — Project Handoff & Design Document

> A portable, cross-platform native UI framework written in Go, targeting desktop
> via WebGPU + GLFW, with a Yoga-style flexbox layout engine and Tree-sitter
> syntax highlighting. This document is a self-contained handoff so another
> developer (or AI tool) can continue the work without prior context.

Module path: `github.com/mirzakhany/yoga` · Go `1.26.2`

---

## 1. What this project is

A from-scratch immediate-ish-mode native UI toolkit demonstrated by a small but
real **code editor** application: a dark-themed coding workspace with a file
explorer, closable tabs, and multiple editable, syntax-highlighted editors over
large files.

The codebase is intentionally layered so each concern is swappable and the
Cgo/GC boundary is tiny and explicit.

### Current feature set (working today)

- Batched WebGPU rendering with WGSL shaders: one `DrawList` upload per frame,
  one indexed draw per scissor segment (a single unclipped UI is one draw).
- HiDPI/Retina-correct text via a baked Source Code Pro font atlas.
- Yoga-compatible flexbox layout (two-pass: solve → flatten to absolute frames).
- **`yoga` runtime package**: GLFW window, WebGPU renderer, font atlas, input
  wiring, OS cursors, clipboard, and an **event-driven** frame loop
  (`WaitEvents` / `WaitEventsTimeout`) so idle CPU stays near zero; scenes may
  implement `Animator` (`AnimationWait`) for caret blink and highlight cadence.
- Components: Button, List, Scrollbar (vertical + horizontal), Icon,
  Menu/Dropdown (overlay/z-order), TabBar, generic **Tree**, **FileTree**
  (thin wrapper over `Tree`), **Splitter**, **TextField**, and a virtualized
  Editor with built-in scrollbars.
- **Editable** editor: caret, mouse click-to-place + drag selection, keyboard
  navigation (arrows/Home/End, Shift-extends), insert/Backspace/Delete,
  Enter/Tab, select-all, clipboard cut/copy/paste, undo/redo with typing
  coalescing, **vertical and horizontal scrolling** (own v/h scrollbars).
- Piece-table text storage + line-window virtualization (handles multi-MB files;
  see `bigdata.json`, ~8 MB, generated for stress testing).
- Async Tree-sitter syntax highlighting on a worker goroutine, pluggable by file
  extension. **Go** and **JSON** grammars are wired; unknown types are plain text.
  **Incremental** reparsing via `UpdateEdit` + `tree.Edit` on each keystroke.
- File explorer rooted at the launch directory (lazy `os.ReadDir` via `Tree.Loader`),
  **scrollable** tree panel (v+h scrollbars), **name filter** via a search
  `TextField`, **right-click context menus** on files, tabs with a modified-dot
  and close box, and **Save to disk** (Cmd/Ctrl+S).
- **Resizable** explorer/editor split (`Splitter`); top menu bar (File/Edit/Theme).
- Dual build: full GPU app (`go run ./cmd/example`) and a **headless** CPU-only
  build (`-tags nogpu`) that exercises the whole pipeline without a window/GPU
  (useful for CI and machines without a working GPU stack).

---

## 2. How to build & run

```bash
# Full GPU app (opens a window)
go run ./cmd/example

# Headless: runs layout + paint + an edit, prints geometry/metrics, no window
go run -tags nogpu ./cmd/example

# Sanity checks
go build ./...                 # GPU build
go build -tags nogpu ./...     # headless build
go vet ./...
go test ./...                  # highlight/json_smoke_test.go, incremental_test.go
```

Generate a large test file (already present as `bigdata.json`):

```bash
python3 -c "import json,random,string;open('bigdata.json','w').write(json.dumps([{'id':i,'name':''.join(random.choices(string.ascii_letters,k=12))} for i in range(20000)],indent=2))"
```

### Key bindings in the demo

- Type to insert; Enter/Tab as expected; Backspace/Delete.
- Arrows / Home / End to move; hold **Shift** to extend selection.
- **Cmd/Ctrl**: A (select all), C/X/V (copy/cut/paste), Z (undo), Shift+Z (redo),
  **S (save active file)**.
- Click a file in the explorer to open it in a new tab; click a folder to
  expand/collapse; click a tab to activate, click its `x` to close.
- Right-click a file in the explorer for a context menu (Open, Copy Path).
- Click the search field to filter files by name; click elsewhere to blur it.
- Drag the splitter handle between explorer and editor to resize panes.

---

## 3. Tech stack & dependency decisions

| Concern | Library | Why this one |
|---|---|---|
| Graphics (WebGPU) | `github.com/cogentcore/webgpu` | Maintained, builds cleanly; chosen over the originally-suggested `rajveermalviya/go-webgpu` which was less maintained. Only this package touches the wgpu Cgo bindings. |
| Windowing/input | `github.com/go-gl/glfw/v3.3/glfw` | Standard, stable Cgo GLFW binding. |
| Layout | `github.com/kjk/flex` | Pure-Go port of Facebook Yoga. Chosen for a clean `go build` (no Cgo) while matching the Yoga flexbox API the project targets. The `layout.Engine` interface allows swapping in a Cgo-native Yoga binding later. |
| Syntax highlighting | `github.com/tree-sitter/go-tree-sitter` + grammars `tree-sitter-go`, `tree-sitter-json` | Official, maintained Tree-sitter Go bindings. |
| Font rasterization | `golang.org/x/image/font/opentype` | High-quality TTF rasterization for the atlas. |

**Original request vs. actual choices:** the initial brief named
`rajveermalviya/go-webgpu`, `jackwakefield/yogoa` (Cgo Yoga), and
`smacker/go-tree-sitter`. During scoping the user chose **`compile_check`**
(must build cleanly) and **`pragmatic`** (most-maintained) bindings, so the
stack above was substituted accordingly. The architecture keeps these behind
interfaces/wrappers so reverting to Cgo-native equivalents is localized.

---

## 4. Architecture: runtime + three layers + leaves

```
┌───────────────────────────── cmd/example (the app) ─────────────────────────┐
│  main_gpu.go (~12 lines)  /  main_headless.go (nogpu)  /  ui.go (Workspace) │
└───────────────▲───────────────────────────────────────────────▲─────────────┘
                │ implements yoga.Scene                         │
        Runtime │ yoga/          (App, GLFW, WebGPU, atlas, input, event loop)
                │                                                 │
        Layer 3 │ components/  theme/   (Button, Editor, Tree, Splitter, ...)
                │                                                 │
        Layer 1 │ layout/          (Element tree, flex engine, 2-pass pipeline)
                │                                                 │
        Layer 2 │ render/          (DrawList, FontAtlas, WGSL, batched Renderer)
                │
         leaves │ input/  text/ (piece table)  highlight/ (tree-sitter worker)
```

The dependency direction is strictly downward. `yoga` imports `render`, `layout`,
and `input` but **not** `components` or `theme` — the app wires those together.
`render` is pure Go **except** the two build-tagged files (`renderer.go` /
`renderer_stub.go`); everything above it builds a `render.DrawList` (plain Go
memory) with no Cgo. GLFW/WebGPU Cgo lives in `yoga/app_gpu.go`, not in the app.

### Data flow per frame (GPU build)

```
GLFW callbacks ──► input.Mouse / input.Keyboard  (wired in yoga.App)
                         │
app.Run loop:  scene.Layout(w,h) ─► flex CalculateLayout ─► flatten to Frame
                         │
               scene.Update(m,kb) ─► route keys (editor or search TextField)
                 ─► Editor.Update() ─► PieceTable + highlight.UpdateEdit
                 ─► layout.Dispatch(mouse) ─► widget OnMouse hooks
                 ─► Tree/Editor scrollbars, Splitter drag, cursor requests
                         │
               layout.Paint(root) ─► one DrawList (PushClip where needed)
                         │
               renderer.Render(dl) ─► single buffer upload ─► indexed draw(s)
                         │              (one per DrawCmd scissor segment)
               glfw.WaitEvents(Timeout) ◄── AnimationWait() when scene animates
```

---

## 5. Package-by-package reference

### `render/` — GPU layer
- `draw.go` (pure Go): `Color`, `Rect` (with `Contains`), `Vertex`
  (pos2 + uv2 + rgba4 = 32 bytes), `DrawCmd` (scissor + index range), and
  `DrawList` with `AddRect` / `AddTexQuad` / `AddRoundedRect` /
  `AddRoundedRectBorder`. `PushClip` / `PopClip` segment the index stream into
  per-viewport scissor commands (`Clip.W < 0` = draw everywhere). `solidUV = -1`
  is the sentinel telling the shader to emit a flat fill instead of sampling
  the atlas.
- `atlas.go` (pure Go): dynamic **glyph atlas** with mono R8 + color RGBA pages, lazy rasterization via go-text, and SVG icons shelf-packed in the mono page. Shaping lives in `shape/` (HarfBuzz-class via go-text/typesetting): bidi, font fallback (`fontscan`), per-line cache, proportional/bidi-aware editor caret and selection.
- `icons.go` (pure Go): SVG icon pipeline. Embeds a curated **Material-style
  icon set** (`assets/icons/*.svg`, Pictogrammers MDI, Apache-2.0) via
  `//go:embed`. `RegisterIcon(name, svg)` adds/overrides an icon (call **before**
  baking the atlas; for post-startup additions, re-bake + `Renderer.UpdateAtlas`).
  `rasterizeIcon` uses `srwiley/oksvg` + `srwiley/rasterx` to render an SVG to an
  alpha coverage mask; the mask is the R8 coverage and is tinted by the vertex
  color at draw time, so a single monochrome icon renders in **any theme color**.
- `icon.go`: `SpriteSheet` resolving named icons to atlas UVs (unchanged).
- `shader.wgsl`: vertex shader converts pixel coords → NDC via a `screen`
  uniform; fragment shader branches on `uv.x < 0` (flat color) vs. atlas sample
  (glyph/icon alpha tinted by vertex color).
- `renderer.go` (`!nogpu`, the ONLY Cgo-in-render file): owns instance/adapter/
  device/queue/surface/pipeline, a uniform buffer (logical screen size), the
  atlas texture/sampler/bind group, and **growable** vertex/index buffers
  (`ensureCapacity`). `Render` uploads the whole DrawList once, then issues one
  indexed draw per `DrawCmd` (setting `SetScissorRect` when clipped). When
  `Commands` is empty, a single full-surface draw is used. `ClearColor`,
  `Resize`, `UpdateAtlas(atlas)` (re-upload atlas + rebuild bind group).
  `Destroy` releases all C objects.
- `renderer_stub.go` (`nogpu`): no-op `Renderer` with matching signatures
  (including `UpdateAtlas`).

### `layout/` — Layer 1 (flexbox tree)
- `style.go`: `Style` value type with fluent builders (`Box()`, `Direction`,
  `FlexGrow`, `W/H/Size`, `PaddingAll/PaddingXY`, `MarginAll`, `Absolute`,
  `AbsLeft/Top/Right/Bottom`, ...). Unset dimensions are NaN (= auto). `apply`
  writes the style onto a `*flex.Node`.
- `layout.go`: `Element` (Style, Children, `Frame`, `Paint`, `OnMouse`,
  `Overlay`, `ScrollOffset`, `Clip`, opaque `backend`). `New(style, children...)`
  is the declarative constructor. `WithBackground` / `WithBackgroundPtr` attach
  a solid fill hook (pointer variant tracks live theme colors). `Calculate` runs
  the **two-pass** pipeline via the swappable `Engine` (default `flexEngine` over
  kjk/flex). `MarkDirty` drops cached nodes for a structural rebuild;
  `ReapplyStyle` updates in place. `Paint` walks base tree then overlays
  (painter's algorithm). `Dispatch` delivers mouse front-to-back (overlays
  first); handlers set `m.Consumed` to stop propagation.

### `input/` — leaf
- `Mouse` (X/Y, Down/Pressed/Released, right-button edges, `ScrollX`/`ScrollY`,
  `Consumed`, `Cursor`) + `SetPos`, `SetButton`, `SetRightButton`, `AddScroll`,
  `AddScrollX`, `SetCursor`, `EndFrame`. `Cursor` enum: default, resize EW/NS.
- `Keyboard` (`Chars []rune`, `Keys []KeyEvent`) + `TypeRune`, `PressKey`,
  `EndFrame`. `Key` enum (editing subset), `Mod` bitflags with
  `Primary()` = Ctrl-or-Super (so shortcuts work cross-platform).
- `Clipboard` interface (`Get/Set`) + `MemClipboard` (headless/in-memory).
  GLFW-backed adapter lives in `yoga/app_gpu.go` (`glfwClipboard`).

### `text/` — leaf
- `PieceTable`: append-only, copy-free buffer (original + add buffers, ordered
  pieces). `Insert([]byte)`, `Delete`, `Len`, `Bytes` (cached flat view),
  `LineCount`, `Line`, `LineStart`. Line starts are recomputed in `rebuildCache`.

### `highlight/` — leaf (Tree-sitter worker)
- Async model: `Update(src)` requests a **full** reparse (initial load); `UpdateEdit(src, edit)`
  requests an **incremental** reparse (`tree.Edit` + `Parse(src, prev)`). `Poll()`
  returns the latest `[]Token` non-blocking. Pending jobs accumulate in order so
  the edit chain stays consistent. Results cross the goroutine boundary as a
  plain Go slice; the UI never touches a live C tree.
- `Edit` / `Pt` describe byte/range mutations for incremental parsing.
- `Highlighter` interface: `Update`, `UpdateEdit`, `Poll`, `Close`.
- `tsHighlighter` is **generic**: parameterized by a grammar (`langFn`) and a
  `classifyFunc`. Cgo tree lifecycle (close previous tree on new parse, close on
  exit) is handled once here.
- `NewGo()` / `NewJSON()` are one-liners over it; `classifyGo` / `classifyJSON`
  map node kinds → `ColorClass`. JSON colors object **keys** (`ClassType`)
  distinctly from string values.
- **`ForPath(path)`** is the single registry mapping extension → highlighter
  (`.go`, `.json`, else `Noop`). **Add a language here.**
- `Token{Start,End,Class}`, `ColorClass` (Default/Keyword/String/Comment/
  Number/Type), `Noop` highlighter.

### `theme/` — palette + runtime themes
- `theme.go`: the semantic `Theme` palette (`Background`, `Panel`, `PanelAlt`,
  `Text`, `TextDim`, `Accent`, `AccentText`, `Hover`, `Active`, `Border`,
  `Selection`, `ScrollTrack`, `ScrollThumb`, `ScrollThumbHover`,
  `Error/Warning/Success`, ... + a
  `Syntax map[highlight.ColorClass]render.Color`) and `SyntaxColor(class)`.
- **Runtime switching model**: there is exactly one live `*Theme`
  (`theme.Current()`). Every widget stores that pointer. `Use(name)` overwrites
  the live instance's fields **in place**, so the next paint uses the new colors
  with **zero rebuild**. `Register(t)`, `Get(name)`, `Names()` (sorted) manage the
  registry; `init()` registers all builtins and selects `dark`.
- `palettes.go`: builtin themes — `dark`, `light`, `github-dark`,
  `github-light`, `catppuccin`, `dracula`, `nord`, `solarized-dark`.
- Layering: `theme` imports only `render` + `highlight`; `components` imports
  `theme` (never the reverse).

### `components/` — Layer 3 (widgets)
Each widget owns an `El *layout.Element` and attaches `Paint`/`OnMouse` hooks.
Widgets hold a `*theme.Theme` (the live `theme.Current()`), so a theme switch is
reflected on the next paint with no rebuild.
- `util.go`: package doc + small float helpers (`f32max/f32min/clampf`).
- `components.go`: `Button`, `NewList`, `NewLabelRow`, `Scrollbar` / `NewScrollbarAxis`
  (`Axis` vertical/horizontal; drives `*float32` offset + content extent; wheel +
  thumb drag), `NewIcon`, `Menu`/`Dropdown` (overlay, z-ordered, painted manually).
  `Menu`: `OpenAt`, `SetItems`, `Close`. Constructors take `*theme.Theme`.
- `tabs.go`: `TabBar` — manually painted tabs with title, a Material `circle`
  modified dot, hover/active states, and a Material `close` icon. `Tabs
  []TabModel`, `Active`, `OnActivate`, `OnClose`. `layoutTabs()` computes per-tab
  extents shared by paint & hit-test.
- `tree.go`: generic `Tree` — scrollable expandable tree (`TreeNode` hierarchy,
  lazy `Loader`, `IconFor`, `ContextMenu`, `SetFilter`, v+h scrollbars, row
  virtualization via scroll offset, right-click menus). `OnActivate`, `OnToggle`.
- `filetree.go`: `FileTree` — thin wrapper over `Tree` for directory browsing.
  Supplies a lazy `os.ReadDir` `Loader` (dirs first, dotfiles hidden), maps
  `OnOpenFile` / `OnChange`, `SetContextMenu`, `SetFilter`. `El()`, `MenuEl()`,
  `Update(m)`.
- `splitter.go`: `Splitter` — draggable handles between panes along horizontal or
  vertical axis; fixed-size sections (`SplitSection.Size`) vs flex-fill (`Size==0`);
  resize cursor via `input.Cursor`.
- `textfield.go`: `TextField` — single-line input (placeholder, password mask,
  start/end icons, rounded border, caret blink). `HandleText`/`HandleKeys`,
  `Focused`/`Blur`, `OnChange`. Used in the demo for file-name search.
- `editor.go`: the big one. `Editor` owns a `PieceTable`, a `Highlighter`, a
  `Clipboard`, caret/selection state, an undo/redo stack, built-in v/h scrollbars,
  and a per-byte color table. Highlights:
  - **Virtualization**: `paint` only emits quads for the visible line window;
    `PushClip` on the text viewport.
  - **Scrolling**: `ScrollPx`/`ScrollX`, `ContentHeight`/`ContentWidth`, own
    `Scrollbar` pair; `ensureCaretVisible` scrolls both axes.
  - **Editing core**: every mutation funnels through `applyEdit(pos, delLen,
    ins, coalesceTyping)`, which records an inverse `editOp` (coalescing
    contiguous single-rune typing into one undo step), calls
    `highlight.UpdateEdit`, sets `modified`, and keeps the caret visible.
  - `HandleText(runes)` / `HandleKeys(keys)` for input; `onMouse` maps pixels →
    byte offset (honoring tab-stop expansion) for click/drag selection.
  - `Undo()` / `Redo()` public; `AnimationWait()` for caret blink + highlight poll.
  - Geometry helpers: `lineOf`, `lineColOf`, `offsetOf`, `prevRune/nextRune`,
    `colToX`, `offsetAtPoint`, `ensureCaretVisible`.
  - API: `NewEditorFor(atlas, theme, path, content, clip)` (picks highlighter
    via `highlight.ForPath`), `Bytes()`, `Path`, `Modified()`, `MarkSaved()`,
    `Update()`, `Close()`.

### `yoga/` — application runtime (GPU + headless stub)
- `yoga.go`: `Config` (title, size, `ClearColor`), `Scene` interface
  (`Root`, `Layout`, `Update`, `Close`), optional `Animator` (`AnimationWait`).
- `app_gpu.go` (`!nogpu`): `App` — `New` (GLFW window, HiDPI atlas bake,
  WebGPU renderer, input callbacks, standard cursors), `SetScene`, blocking `Run`
  (layout → update → paint → render → `WaitEvents`/`WaitEventsTimeout`),
  `Atlas()`, `Clipboard()`, `Close()`. Discovers `ClearColor()` and
  `AnimationWait()` on the scene via interface assertions. `glfwClipboard` adapter.
- `app_stub.go` (`nogpu`): minimal stubs if needed for non-GPU tooling.

### `cmd/example/` — the application
- `ui.go`: `Workspace` implements `yoga.Scene` — owns `docs []*Editor`, `active`,
  `TabBar`, `FileTree`, search `TextField`, `Splitter` (explorer | editor column),
  `editorHost` (mounts active editor only). `openFile`/`closeTab`/`save`/
  `setActive`. `Layout`/`Update`/`Close`; `AnimationWait` delegates to active editor.
  Keys route to the search field when focused, else the active editor; blur search
  on click outside. File tree: `SetFilter` from search, `SetContextMenu` (Open,
  Copy Path). Top bar: File/Edit/Theme dropdowns. `relayout()` on structural change.
  `ClearColor()` for runtime theme background sync.
- `main_gpu.go` (`!nogpu`): ~12-line `main` — `yoga.New`, `SetScene(BuildWorkspace(...))`,
  `Run()`.
- `main_headless.go` (`nogpu`): layout + paint + edit without window/GPU.
- `sample.go`: the seed `welcome.go` buffer content.

---

## 6. Key design decisions & rationale

1. **Strict layering with a tiny Cgo surface.** Only `render/renderer.go`,
   `highlight`, and `yoga/app_gpu.go` touch Cgo. Everything else is pure Go and
   builds under `-tags nogpu`. This keeps the GC/Cgo boundary auditable: geometry
   crosses to the GPU only as a flat `[]byte` via `queue.WriteBuffer` (copied
   immediately); no Go pointer is ever stored in a C struct.
2. **Single batched geometry upload.** All widgets append into one `DrawList`;
   the renderer uploads once, then issues one indexed draw per scissor `DrawCmd`
   (typically one draw when nothing is clipped). No per-widget GPU objects.
3. **Event-driven frame loop.** `App.Run` repaints on input/resize or animation
   deadlines (`AnimationWait`), then blocks in GLFW — not a busy vsync spin.
4. **Two-pass layout.** `Calculate` solves flex constraints; `flatten`
   accumulates parent origins into absolute `Element.Frame` rectangles consumed
   by paint and hit-testing. Scrolling shifts children in `flatten`
   (`ScrollOffset`) without re-running the solver.
5. **Swappable engine/highlighter via interfaces.** `layout.Engine` and
   `highlight.Highlighter` allow replacing the pure-Go flex port with Cgo Yoga,
   or adding languages, without touching callers.
6. **Piece table over string/line-array.** Edits are O(edit), not O(file), so
   multi-MB files stay responsive. Combined with line-window virtualization in
   the editor's paint.
7. **Async + incremental highlighting.** Tree-sitter parses on a worker goroutine;
   edits use `UpdateEdit` + `tree.Edit` when a previous tree exists. The UI
   polls finished token sets, so input/scroll never block on parse.
8. **HiDPI correctness.** Atlas bakes at physical pixel scale; the surface
   renders at framebuffer size; the shader's uniform uses logical size, so the
   UI is authored in logical units yet rendered crisply.
9. **sRGB handling.** The renderer picks a non-sRGB surface format when
   available (`pickFormat`) to avoid washed-out colors from double gamma
   correction on macOS.
10. **Undo coalescing.** Contiguous single-rune typing merges into one undo step;
    structural ops (newline, paste, delete-selection, cut) are their own steps.
11. **Immediate relayout on structural change.** Opening/closing tabs or
    toggling folders calls `MarkDirty` + re-`Calculate` at the last viewport in
    the same frame, avoiding a visible one-frame glitch.

### Notable trade-offs / shortcuts (intentional)

- **Shaping cost.** Per-line HarfBuzz shaping + lazy glyph bake is heavier than
  the old `CellW` grid; mitigated by the per-line shape cache in `shape/`.
- **Per-byte color table** (`[]ColorClass` sized to document length) is simple
  but O(n) memory; fine for now, revisit for very large files.
- **Partial focus routing**: the demo routes keys to the active editor or the
  search `TextField` when focused; there is no general tab-order focus manager.
- **Tree filter + lazy load**: `SetFilter` only searches loaded branches; deep
  unexpanded folders are invisible to the filter until expanded.
- **Full repaint every frame** that runs (no dirty regions); mitigated by the
  event-driven loop when idle.

---

## 7. The original plan (implemented)

The "Editable Editor, Tabs, and File Explorer" plan turned the read-only demo
into a working editor. All of its to-dos are complete:

1. ✅ `input`: `Key`/`Mod` enums, `KeyEvent`, `Keyboard`, `Clipboard`.
2. ✅ Editor caret/selection: click-to-place, drag-select, blinking caret,
   selection rects, `ensureCaretVisible`.
3. ✅ Editor editing: insert/Enter/Tab/Backspace/Delete, arrows/Home/End
   (Shift-extends), select-all, single `applyEdit` path.
4. ✅ Clipboard cut/copy/paste, undo/redo with coalescing, `modified` flag,
   `NewEditorFor`, `Bytes/Path/MarkSaved`.
5. ✅ `components/tabs.go` (TabBar).
6. ✅ `components/filetree.go` (recursive lazy explorer).
7. ✅ `cmd/example/ui.go` rework (multi-doc, tabs, explorer, save, keyboard routing,
   Cmd/Ctrl+S); later: splitter, search field, tree context menus.
8. ✅ `main_gpu.go` slim entry; GLFW/WebGPU/input in `yoga/app_gpu.go`.
9. ✅ `main_headless.go` updated; build/vet/headless/GPU verified.

Plus follow-ups: **generalized syntax highlighting** (generic `tsHighlighter`,
JSON grammar, `ForPath` registry); **incremental reparsing** (`UpdateEdit`);
**editor horizontal scroll**; **generic `Tree`** with scroll + context menus;
**`yoga` runtime** extraction; **Splitter** + **TextField** + file search.

---

## 8. Suggested roadmap / next steps

Good next tasks for whoever continues (roughly ordered):

1. **More languages.** Add grammars (JS/TS, Rust, Python, YAML, Markdown,
   TOML). Mechanically: `go get` the grammar binding, write a `classifyXxx`, add
   a `case` to `highlight.ForPath`. Consider a small DSL/table for node-kind →
   class to reduce boilerplate.
2. ~~**Incremental reparsing.**~~ Done (`UpdateEdit` + `tree.Edit` in worker).
3. ~~**Horizontal scrolling** in the editor.~~ Done (built-in h-scrollbar).
4. **File explorer polish.** File-watcher refresh; context menu actions beyond
   Open/Copy Path (new/rename/delete); filter across unexpanded lazy branches.
5. **Find/replace** in the editor (Edit menu has Undo/Redo only today).
6. ~~**Unicode/i18n font support**~~ Done: `shape/` + dynamic glyph atlas (mono +
   color pages), `fontscan` system fallback, bidi-aware editor caret/selection.
7. **Focus manager** — tab/shift-tab between all focusable widgets.
8. **Multiple cursors / word-wise movement (Ctrl/Alt+arrows), line operations.**
9. **Dirty/region rendering** if frame cost matters (currently full repaint per
   wake; idle CPU is already low via event loop).
10. **Native Cgo Yoga engine** behind `layout.Engine` if grid/advanced features
    are needed beyond kjk/flex.
11. **Tests**: extend beyond `highlight/*_test.go` — piece table, editor edit
    ops, undo/redo, offset/line-col mapping, splitter/tree interaction.

---

## 9. Gotchas for the next developer

- **Build tags**: anything touching wgpu must be `//go:build !nogpu`; provide a
  `nogpu` stub with matching signatures (see `renderer_stub.go`). Always run both
  `go build ./...` and `go build -tags nogpu ./...`.
- **`pt.Insert` takes `[]byte`**, not `string` (convert in editor ops).
- **`pt.Bytes()` returns a cached slice** reused after edits — copy out
  (`string(...)`) before mutating if you need to retain it (the undo stack does).
- **Text engine** lives in `shape/`; widgets take `*shape.Engine` (fonts + atlas +
  shaper). `App.Text()` returns it; `App.Atlas()` is legacy (atlas only).
- **Highlighter is async** — colors lag the edit by a frame or two; that's
  expected. Always `Close()` editors (the workspace does) to stop worker
  goroutines and free C trees.
- **Layout caching**: after changing an Element's children, call `MarkDirty`
  (structural) or `ReapplyStyle` (style-only) or the change won't take effect.
- **`m.Consumed`** is how widgets stop event propagation; overlays are dispatched
  first.
- **`App.Run` blocks on the main goroutine** (GLFW OS-thread lock in `yoga` init);
  background work is fine, but window/input/render must stay on main.
- **Scissor commands**: if you add clipped scrolling, pair `PushClip`/`PopClip`;
  a mismatched stack corrupts `DrawCmd` index ranges.

---

## 10. File index

```
yoga.go              Config, Scene, Animator interfaces
app_gpu.go           App: GLFW, WebGPU, input, event loop, clipboard        [!nogpu]
app_stub.go          Runtime stubs                                           [nogpu]
cmd/example/
  main_gpu.go        GPU entry (~12 lines: yoga.New + SetScene + Run)       [!nogpu]
  main_headless.go   Headless entry (no window/GPU)                         [nogpu]
  ui.go              Workspace: tabs, splitter, tree, search, multi-editor
  sample.go          Seed welcome buffer content
theme/
  theme.go           Palette + registry + live active theme (runtime switch)
  palettes.go        Builtin themes (dark, light, github, catppuccin, ...)
components/
  util.go            Package doc + float helpers
  components.go      Button, List, Scrollbar (v/h), Icon, Menu, Dropdown
  tabs.go            TabBar
  tree.go            Generic scrollable tree (lazy load, filter, context menu)
  filetree.go        Directory explorer (wraps Tree)
  splitter.go        Draggable multi-pane splitter
  textfield.go       Single-line text input
  editor.go          Virtualized editor + v/h scrollbars + incremental HL
layout/
  style.go           Style + fluent builders (maps to flex)
  layout.go          Element tree, 2-pass pipeline, Paint/Dispatch, Engine
render/
  draw.go            Color/Rect/Vertex/DrawList, clip + rounded rects
  atlas.go           Dynamic glyph atlas (mono R8 + color RGBA) + icons
  icons.go           SVG icon pipeline + embedded Material set
shape/
  fonts.go           FontSystem (Source Code Pro + fontscan fallback)
  line.go            Shaper, shaped Line (bidi, tabs, hit-test)
  cache.go           Per-line shape cache
  engine.go          Engine: DrawString, DrawLineGlyphs, FlushAtlas
  assets/icons/      Curated Material-style SVG icons (Apache-2.0)
  icon.go            SpriteSheet
  shader.wgsl        Vertex/fragment shaders
  renderer.go        Batched WebGPU renderer (scissor per DrawCmd)          [!nogpu]
  renderer_stub.go   No-op renderer                                          [nogpu]
input/  input.go     Mouse (incl. right btn, scroll X, cursor), Keyboard, Clipboard
text/   piecetable.go Piece-table text storage
highlight/
  highlight.go       Async Tree-sitter worker, incremental edits, ForPath
  json_smoke_test.go JSON highlighting test
  incremental_test.go Incremental reparse tests
render/assets/SourceCodePro-Regular.ttf   Embedded font
bigdata.json         ~8 MB generated JSON for editor stress testing
```
