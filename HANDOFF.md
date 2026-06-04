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

- Batched WebGPU rendering (one indexed draw call per frame) with WGSL shaders.
- HiDPI/Retina-correct text via a baked Source Code Pro font atlas.
- Yoga-compatible flexbox layout (two-pass: solve → flatten to absolute frames).
- Components: Button, List, Scrollbar, Icon, Menu/Dropdown (overlay/z-order),
  TabBar, FileTree, and a virtualized Editor.
- **Editable** editor: caret, mouse click-to-place + drag selection, keyboard
  navigation (arrows/Home/End, Shift-extends), insert/Backspace/Delete,
  Enter/Tab, select-all, clipboard cut/copy/paste, and undo/redo with typing
  coalescing.
- Piece-table text storage + line-window virtualization (handles multi-MB files;
  see `bigdata.json`, ~8 MB, generated for stress testing).
- Async Tree-sitter syntax highlighting on a worker goroutine, pluggable by file
  extension. **Go** and **JSON** grammars are wired; unknown types are plain text.
- File explorer rooted at the launch directory (lazy `os.ReadDir`), tabs with a
  modified-dot and close box, and **Save to disk** (Cmd/Ctrl+S).
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
go test ./...                  # includes highlight/json_smoke_test.go
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

## 4. Architecture: three layers + leaves

```
┌───────────────────────────── cmd/example (the app) ─────────────────────────┐
│  main_gpu.go (GLFW+WebGPU)  /  main_headless.go (nogpu)  /  ui.go (Workspace) │
└───────────────▲───────────────────────────────────────────────▲─────────────┘
                │                                                 │
        Layer 3 │ components/      (Button, Editor, TabBar, FileTree, ...)
                │                                                 │
        Layer 1 │ layout/          (Element tree, flex engine, 2-pass pipeline)
                │                                                 │
        Layer 2 │ render/          (DrawList, FontAtlas, WGSL, batched Renderer)
                │
         leaves │ input/  text/ (piece table)  highlight/ (tree-sitter worker)
```

The dependency direction is strictly downward. `render` is pure Go **except** the
two build-tagged files (`renderer.go` / `renderer_stub.go`); everything above it
builds a `render.DrawList` (plain Go memory) with no Cgo.

### Data flow per frame (GPU build)

```
GLFW callbacks ──► input.Mouse / input.Keyboard
                         │
ws.Layout(w,h)  ─► flex CalculateLayout ─► flatten to absolute Element.Frame
                         │
ws.Update(m,kb) ─► route keys to active Editor ─► PieceTable Insert/Delete
                 ─► Editor.Update() polls highlight worker ─► per-byte colors
                 ─► layout.Dispatch(mouse) ─► widget OnMouse hooks
                 ─► Scrollbar.Update
                         │
layout.Paint(root) ─► every widget appends quads into one DrawList
                         │
renderer.Render(dl) ─► queue.WriteBuffer (single upload) ─► one indexed draw
```

---

## 5. Package-by-package reference

### `render/` — GPU layer
- `draw.go` (pure Go): `Color`, `Rect` (with `Contains`), `Vertex`
  (pos2 + uv2 + rgba4 = 32 bytes), and `DrawList` with `AddRect` / `AddTexQuad`.
  `solidUV = -1` is the sentinel telling the shader to emit a flat fill instead
  of sampling the atlas.
- `atlas.go` (pure Go): `FontAtlas` bakes ASCII glyphs (32–126) from embedded
  **Source Code Pro** into a grid, then rasterizes every registered SVG icon
  (see `icons.go`) and shelf-packs them into an icon region appended **below**
  the glyph grid — all in one R8 coverage texture. `NewMonoAtlasScale(scale)`
  bakes both glyphs and icons at device-pixel scale for HiDPI (icons stay crisp);
  `CellW/CellH` are reported in **logical** pixels. `GlyphUV`, `IconUV`,
  `Measure`, `DrawText`. **Only ASCII is baked** — non-ASCII renders as `?`.
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
  (`ensureCapacity`). `Render` uploads the whole DrawList with `queue.WriteBuffer`
  and issues one indexed draw. `UpdateAtlas(atlas)` re-uploads the atlas texture
  (which may have grown) and rebuilds the bind group — used when icons are
  registered after startup. `Destroy` releases all C objects.
- `renderer_stub.go` (`nogpu`): no-op `Renderer` with matching signatures
  (including `UpdateAtlas`).

### `layout/` — Layer 1 (flexbox tree)
- `style.go`: `Style` value type with fluent builders (`Box()`, `Direction`,
  `FlexGrow`, `W/H/Size`, `PaddingAll/PaddingXY`, `MarginAll`, `Absolute`,
  `AbsLeft/Top/Right/Bottom`, ...). Unset dimensions are NaN (= auto). `apply`
  writes the style onto a `*flex.Node`.
- `layout.go`: `Element` (Style, Children, `Frame`, `Paint`, `OnMouse`,
  `Overlay`, `ScrollOffset`, `Clip`, opaque `backend`). `New(style, children...)`
  is the declarative constructor. `Calculate` runs the **two-pass** pipeline via
  the swappable `Engine` (default `flexEngine` over kjk/flex). `MarkDirty` drops
  cached nodes for a structural rebuild; `ReapplyStyle` updates in place.
  `Paint` walks base tree then overlays (painter's algorithm). `Dispatch`
  delivers mouse front-to-back (overlays first); handlers set `m.Consumed` to
  stop propagation.

### `input/` — leaf
- `Mouse` (X/Y, Down/Pressed/Released, ScrollY, Consumed) + `EndFrame`.
- `Keyboard` (`Chars []rune`, `Keys []KeyEvent`) + `TypeRune`, `PressKey`,
  `EndFrame`. `Key` enum (editing subset), `Mod` bitflags with
  `Primary()` = Ctrl-or-Super (so shortcuts work cross-platform).
- `Clipboard` interface (`Get/Set`) + `MemClipboard` (headless/in-memory).
  GLFW-backed adapter lives in `cmd/example/main_gpu.go`.

### `text/` — leaf
- `PieceTable`: append-only, copy-free buffer (original + add buffers, ordered
  pieces). `Insert([]byte)`, `Delete`, `Len`, `Bytes` (cached flat view),
  `LineCount`, `Line`, `LineStart`. Line starts are recomputed in `rebuildCache`.

### `highlight/` — leaf (Tree-sitter worker)
- Async model: editor calls `Update(src)` (non-blocking, coalescing) and
  `Poll()` (non-blocking) for the latest `[]Token`. Results cross the goroutine
  boundary as a plain Go slice; the UI never touches a live C tree.
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
- `theme.go`: the semantic `Theme` palette (`Background`, `Panel`, `Text`,
  `Accent`, `Hover`, `Selection`, `Error/Warning/Success`, ... + a
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
- `components.go`: `Button`, `NewList`, `NewLabelRow`, `Scrollbar` (drives a
  `*float32` offset; wheel + thumb drag), `NewIcon`, `Menu`/`Dropdown`
  (overlay, z-ordered, painted manually). Constructors take `*theme.Theme`.
- `tabs.go`: `TabBar` — manually painted tabs with title, a Material `circle`
  modified dot, hover/active states, and a Material `close` icon. `Tabs
  []TabModel`, `Active`, `OnActivate`, `OnClose`. `layoutTabs()` computes per-tab
  extents shared by paint & hit-test.
- `filetree.go`: `FileTree` — recursive explorer rooted at a path. Lazy
  `os.ReadDir` per folder (dirs first, dotfiles hidden), flattened visible-row
  model, depth indentation, Material `folder`/`folder_open`/`file` icons.
  `OnOpenFile(path)`, `OnChange()`.
- `editor.go`: the big one. `Editor` owns a `PieceTable`, a `Highlighter`, a
  `Clipboard`, caret/selection state, an undo/redo stack, and a per-byte color
  table. Highlights:
  - **Virtualization**: `paint` only emits quads for the visible line window.
  - **Editing core**: every mutation funnels through `applyEdit(pos, delLen,
    ins, coalesceTyping)`, which records an inverse `editOp` (coalescing
    contiguous single-rune typing into one undo step), reparses, sets
    `modified`, and keeps the caret visible.
  - `HandleText(runes)` / `HandleKeys(keys)` for input; `onMouse` maps pixels →
    byte offset (honoring tab-stop expansion) for click/drag selection.
  - Geometry helpers: `lineOf`, `lineColOf`, `offsetOf`, `prevRune/nextRune`,
    `colToX`, `offsetAtPoint`, `ensureCaretVisible`.
  - API: `NewEditorFor(atlas, theme, path, content, clip)` (picks highlighter
    via `highlight.ForPath`), `Bytes()`, `Path`, `Modified()`, `MarkSaved()`,
    `Update()`, `Close()`.

### `cmd/example/` — the application
- `ui.go`: `Workspace` — owns `docs []*Editor`, `active int`, a `TabBar`, a
  `FileTree`, one shared `Scrollbar` rebound to the active doc, and an
  `editorHost` element that mounts the active editor. `openFile`/`closeTab`/
  `save`/`setActive`. `Layout(w,h)` records the viewport and runs the layout;
  `Update(m,kb)` routes keys to the active editor, handles global Cmd/Ctrl+S,
  dispatches mouse, and syncs tab modified flags. Structural changes call
  `relayout()` (MarkDirty + immediate re-solve at last viewport to avoid a
  one-frame glitch). The top bar has a **Theme** dropdown listing `theme.Names()`,
  each item calling `theme.Use(name)` — colors update instantly (no relayout).
  `Workspace.ClearColor()` returns `theme.Current().Background`; the yoga runtime
  reads it each frame (optional `interface{ ClearColor() render.Color }`
  capability) so the **window background** switches with the theme too.
- `main_gpu.go` (`!nogpu`): a ~12-line `main` using `yoga.New`/`SetScene`/`Run`;
  all GLFW/WebGPU/atlas/input boilerplate lives in `yoga` (`app_gpu.go`).
  `theme.Current()` seeds the initial clear color.
- `main_headless.go` (`nogpu`): same pipeline with a `MemClipboard`, no window;
  types a couple chars and prints geometry/metrics.
- `sample.go`: the seed `welcome.go` buffer content.

---

## 6. Key design decisions & rationale

1. **Strict layering with a tiny Cgo surface.** Only `render/renderer.go`,
   `highlight`, and the GLFW main file touch Cgo. Everything else is pure Go and
   builds under `-tags nogpu`. This keeps the GC/Cgo boundary auditable: geometry
   crosses to the GPU only as a flat `[]byte` via `queue.WriteBuffer` (copied
   immediately); no Go pointer is ever stored in a C struct.
2. **Single batched draw call.** All widgets append into one `DrawList`; the
   renderer uploads once and draws once per frame with growable buffers. No
   per-element draw calls.
3. **Two-pass layout.** `Calculate` solves flex constraints; `flatten`
   accumulates parent origins into absolute `Element.Frame` rectangles consumed
   by paint and hit-testing. Scrolling shifts children in `flatten`
   (`ScrollOffset`) without re-running the solver.
4. **Swappable engine/highlighter via interfaces.** `layout.Engine` and
   `highlight.Highlighter` allow replacing the pure-Go flex port with Cgo Yoga,
   or adding languages, without touching callers.
5. **Piece table over string/line-array.** Edits are O(edit), not O(file), so
   multi-MB files stay responsive. Combined with line-window virtualization in
   the editor's paint.
6. **Async highlighting.** Tree-sitter parses on a worker goroutine; the UI
   polls finished token sets, so a large-file parse never blocks input/scroll.
   Generalized to any grammar via `tsHighlighter` + `ForPath`.
7. **HiDPI correctness.** Atlas bakes at physical pixel scale; the surface
   renders at framebuffer size; the shader's uniform uses logical size, so the
   UI is authored in logical units yet rendered crisply.
8. **sRGB handling.** The renderer picks a non-sRGB surface format when
   available (`pickFormat`) to avoid washed-out colors from double gamma
   correction on macOS.
9. **Undo coalescing.** Contiguous single-rune typing merges into one undo step;
   structural ops (newline, paste, delete-selection, cut) are their own steps.
10. **Immediate relayout on structural change.** Opening/closing tabs or
    toggling folders calls `MarkDirty` + re-`Calculate` at the last viewport in
    the same frame, avoiding a visible one-frame glitch.

### Notable trade-offs / shortcuts (intentional)

- **ASCII-only atlas.** Non-ASCII glyphs render as `?` (tab titles truncate with
  `..`). Real i18n needs a dynamic/append glyph cache.
- **FileTree has no internal scroll** — it clips to the visible rows that fit.
- **Editor reparses the whole file on every edit** (full parse, not incremental
  `tree.Edit`). Fine to a few MB; see roadmap.
- **Per-byte color table** (`[]ColorClass` sized to document length) is simple
  but O(n) memory; fine for now, revisit for very large files.
- **Mouse focus is implicit**: keys always route to the active editor; there is
  no general focus manager.
- **No horizontal scrolling** in the editor; long lines extend past the
  viewport.

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
7. ✅ `cmd/example/ui.go` rework (multi-doc, tabs, explorer, rebindable
   scrollbar, save, keyboard routing, Cmd/Ctrl+S).
8. ✅ `main_gpu.go` char/key callbacks + GLFW clipboard.
9. ✅ `main_headless.go` updated; build/vet/headless/GPU verified.

Plus a follow-up: **generalized syntax highlighting** (generic `tsHighlighter`,
JSON grammar added, `ForPath` registry).

---

## 8. Suggested roadmap / next steps

Good next tasks for whoever continues (roughly ordered):

1. **More languages.** Add grammars (JS/TS, Rust, Python, YAML, Markdown,
   TOML). Mechanically: `go get` the grammar binding, write a `classifyXxx`, add
   a `case` to `highlight.ForPath`. Consider a small DSL/table for node-kind →
   class to reduce boilerplate.
2. **Incremental reparsing.** Feed `tree.Edit(InputEdit)` + previous tree into
   `parser.Parse` to avoid full reparses on every keystroke for big files.
3. **Horizontal scrolling + long-line handling** in the editor.
4. **FileTree scrolling** (reuse `Scrollbar`) and a file-watcher to refresh on
   disk changes; add context menu (new/rename/delete).
5. **Find/replace** (the Edit menu has placeholders).
6. **Unicode/i18n font support**: dynamic glyph atlas (append + repack), shaping.
7. **Focus manager** so non-editor widgets can receive keys; tab/shift-tab.
8. **Multiple cursors / word-wise movement (Ctrl/Alt+arrows), line operations.**
9. **Dirty/region rendering** if frame cost matters (currently full repaint).
10. **Native Cgo Yoga engine** behind `layout.Engine` if grid/advanced features
    are needed beyond kjk/flex.
11. **Tests**: extend beyond `highlight/json_smoke_test.go` — piece table,
    editor edit ops, undo/redo, offset/line-col mapping.

---

## 9. Gotchas for the next developer

- **Build tags**: anything touching wgpu must be `//go:build !nogpu`; provide a
  `nogpu` stub with matching signatures (see `renderer_stub.go`). Always run both
  `go build ./...` and `go build -tags nogpu ./...`.
- **`pt.Insert` takes `[]byte`**, not `string` (convert in editor ops).
- **`pt.Bytes()` returns a cached slice** reused after edits — copy out
  (`string(...)`) before mutating if you need to retain it (the undo stack does).
- **Atlas is ASCII-only** — don't rely on non-ASCII glyphs in UI chrome.
- **Highlighter is async** — colors lag the edit by a frame or two; that's
  expected. Always `Close()` editors (the workspace does) to stop worker
  goroutines and free C trees.
- **Layout caching**: after changing an Element's children, call `MarkDirty`
  (structural) or `ReapplyStyle` (style-only) or the change won't take effect.
- **`m.Consumed`** is how widgets stop event propagation; overlays are dispatched
  first.

---

## 10. File index

```
cmd/example/
  main_gpu.go        GPU entry (GLFW+WebGPU, input mapping, clipboard)   [!nogpu]
  main_headless.go   Headless entry (no window/GPU)                       [nogpu]
  ui.go              Workspace: tabs + explorer + multi-editor + save
  sample.go          Seed welcome buffer content
theme/
  theme.go           Palette + registry + live active theme (runtime switch)
  palettes.go        Builtin themes (dark, light, github, catppuccin, ...)
components/
  util.go            Package doc + float helpers
  components.go       Button, List, Scrollbar, Icon, Menu, Dropdown
  tabs.go            TabBar
  filetree.go        Recursive file explorer
  editor.go          Virtualized, editable, highlighted code editor
layout/
  style.go           Style + fluent builders (maps to flex)
  layout.go          Element tree, 2-pass pipeline, Paint/Dispatch, Engine
render/
  draw.go            Color/Rect/Vertex/DrawList (pure Go)
  atlas.go           Font atlas (Source Code Pro) + SVG icons, HiDPI baking
  icons.go           SVG icon pipeline + embedded Material set
  assets/icons/      Curated Material-style SVG icons (Apache-2.0)
  icon.go            SpriteSheet
  shader.wgsl        Vertex/fragment shaders
  renderer.go        Batched WebGPU renderer                              [!nogpu]
  renderer_stub.go   No-op renderer                                        [nogpu]
input/  input.go     Mouse, Keyboard, Key/Mod, Clipboard
text/   piecetable.go Piece-table text storage
highlight/
  highlight.go       Async Tree-sitter worker, Go+JSON, ForPath registry
  json_smoke_test.go JSON highlighting test
render/assets/SourceCodePro-Regular.ttf   Embedded font
bigdata.json         ~8 MB generated JSON for editor stress testing
```
