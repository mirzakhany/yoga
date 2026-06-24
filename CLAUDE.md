# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

Yoga is a from-scratch, cross-platform native UI framework written in Go (module `github.com/mirzakhany/yoga`, Go 1.26.2). It is demonstrated by a real code editor application: a dark-themed workspace with a file explorer, closable tabs, and syntax-highlighted editors over large files. The framework uses WebGPU + GLFW for rendering and a pure-Go layout engine (flex, grid, stack).

## Build & run

```bash
# Full GPU app (opens a window)
go run ./cmd/example

# Headless build: exercises the full pipeline without a window/GPU (useful for CI)
go run -tags nogpu ./cmd/example

# Sanity checks
go build ./...
go build -tags nogpu ./...
go vet ./...
go test ./...

# Run a single test package
go test ./layout/...
go test ./highlight/...
```

There is no Makefile. The `-tags nogpu` build tag is the only special flag; it swaps GPU-dependent files with no-op stubs.

## Architecture

The dependency direction is strictly **downward** — upper layers import lower ones, never the reverse.

```
cmd/example          (app: Workspace UI, file explorer, tabs, editor)
    │
yoga/                (runtime: Window, GLFW, WebGPU renderer, font atlas, input, event loop)
    │
components/          (Button, Editor, Tree, Splitter, Dialog, …; each exposes Layout(c))
    │
ui/                  (Ctx, View, FocusScope, VStack/HStack, fluent modifiers — the ergonomic surface)
    │
layout/  theme/      (Element tree, flex/grid/stack solvers, 2-pass pipeline; design tokens)
    │
render/              (DrawList, FontAtlas, WGSL shader, batched Renderer)
    │
input/  text/  highlight/   (leaves: mouse/kb state, piece-table, tree-sitter worker)
```

**`yoga` (runtime package)** — owns the GLFW window, WebGPU device, font atlas, and the event-driven frame loop (`WaitEvents`/`WaitEventsTimeout` for near-zero idle CPU). An app implements `yoga.App` (one method, `Body(c *ui.Ctx) *layout.Element`) and is started with `yoga.Run(cfg, build)`, where `build` constructs the app *after* the window and text engine exist (widgets measure text at construction). Optional capabilities are detected by type assertion: `Closer`, `KeyHook`, and a `ClearColor() render.Color` method. The concrete window-owner type is `yoga.Window`. Cgo (GLFW + WebGPU) lives entirely in `app_gpu.go`; `app_stub.go` is the headless counterpart.

**`ui` (ergonomic surface)** — the SwiftUI/Gio-inspired layer. `Ctx` is the per-frame build context (the "gtx"): `Invalidate` (wakes the loop via `glfw.PostEmptyEvent`), `Animate(d)` (schedules a repaint; the runtime takes the min across the frame), `Overlay(el)` (registers a portal — dropdown/menu/dialog — composed into a layer painted after the body), `Focus()` (a rebuild-aware `FocusScope` routing Tab/keys), plus `Mouse()/Keyboard()/Viewport()/Theme()/Text()`. `View` is the universal widget interface (`Layout(c) *layout.Element`); `ui.VStack/HStack/ZStack/Spacer` and the fluent `*Element` modifiers (`Gap`, `FlexGrow`, per-side `Padding*`/`Margin*`, …) live here. `BuildFrame(c, body, w, h, m, kb)` runs one build pass and is shared by the GPU runtime and every headless main/test.

**`render/`** — pure Go except `renderer.go` (the only Cgo file in the render layer). `draw.go` defines `DrawList`, `Vertex`, `DrawCmd`, and draw primitives (`AddRect`, `AddRoundedRect`, `AddTexQuad`, etc.). `PushClip`/`PopClip` segment the index stream into scissor-bounded `DrawCmd`s. The `solidUV = -1` sentinel tells the WGSL shader to emit a flat fill instead of sampling the atlas. `atlas.go` manages the glyph/icon atlas; text shaping is in `shape/` (HarfBuzz-quality via `go-text/typesetting`).

**`layout/`** — `Element` is the node type (Style, Children, Frame, Paint hook, OnMouse hook, Overlay, ScrollOffset, Clip). `New(style, children...)` is the declarative constructor. `Calculate` runs the two-pass pipeline: solve relative geometry → flatten to absolute `Frame`s. `Dispatch` delivers mouse events front-to-back (overlays first); set `m.Consumed = true` to stop propagation. `Style` uses NaN for unset/auto dimensions and has fluent builders (`Box()`, `FlexGrow`, `GridCols`, `Absolute`, etc.).

**`components/`** — high-level widgets. Each is a retained state struct that survives every rebuild and exposes `Layout(c *ui.Ctx) *layout.Element`: it returns its element and self-registers focus (`c.Focus().Add`), animation (`c.Animate`), and overlays (`c.Overlay`) through the context — no app-level wiring. The `Editor` uses piece-table storage (`text/piecetable.go`) and line-window virtualization to handle multi-MB files. Focus traversal is owned by `ui.FocusScope` (rebuilt each frame; `EnsureFocus` sets a default, `Route` applies text before keys so same-frame type+Enter works).

**`highlight/`** — async Tree-sitter syntax highlighting on a worker goroutine. Incremental reparsing via `UpdateEdit` + `tree.Edit` on each keystroke. Go and JSON grammars are wired; unknown file types fall back to plain text.

## Frame loop data flow

The body is rebuilt every drawn frame from `Ctx` (there is no host caching), so
overlays and animation requests are collected fresh. The event loop only
iterates on input or an `Animate` request, so this is cheap and idle CPU stays
near zero. Per drawn frame the body builds twice: once to hit-test input against
fresh geometry, then again so paint reflects state changed by that input.

```
GLFW callbacks → input.Mouse / input.Keyboard
yoga.Window.runApp (ui.BuildFrame each pass):
  Ctx.BeginFrame(w,h,m,kb)
  App.Body(c)            → components' Layout(c): build elements,
                           register focus / Animate / Overlay
                         → compose overlays into a synthetic root
                         → layout.Calculate → flatten to Frame
  layout.Dispatch(mouse) → OnMouse hooks (overlays hit-tested first)
  Ctx.Focus().Route(kb)  → focused widget HandleText then HandleKeys
  (components do per-frame Update via c.Mouse() inside Layout)
  layout.Paint(root)     → DrawList
  renderer.Render(dl)    → single buffer upload → one indexed draw per DrawCmd
  glfw.WaitEventsTimeout ← min Ctx.Animate(d) this frame; else WaitEvents (idle)
```

`Ctx.Invalidate()` (safe from any goroutine, e.g. a finished highlight/LSP
result) calls `glfw.PostEmptyEvent` to break the idle wait and force a repaint.

## Build tags

- `//go:build !nogpu` — real GPU/GLFW implementation (default)
- `//go:build nogpu`  — headless stubs; no Cgo, no window; used for CI and machines without a GPU stack

Files ending in `_gpu.go` and `_stub.go` follow this pattern throughout the codebase.

## Key design decisions

- Cgo is restricted to two locations: `yoga/app_gpu.go` (GLFW + WebGPU) and `render/renderer.go` (WebGPU device/pipeline). Everything else is pure Go.
- `render.DrawList` is plain Go memory; components and layout build it without touching Cgo.
- `cogentcore/webgpu` was chosen over `rajveermalviya/go-webgpu` for better maintenance; `go-tree-sitter` (official) over `smacker/go-tree-sitter`.
- The layout engine is intentionally swappable via the `layout.Engine` interface; the default is `customEngine` (flex/grid/stack in pure Go, no Cgo Yoga).
- Retained components + per-frame declarative rebuild (the SwiftUI `@State` analogue): component structs hold state across frames; `Body(c)` only re-wires elements. `Ctx`/`View` live in `ui` (not on `layout.Element`) so `layout` need not import `theme`/`ui` — the dependency direction stays downward; a bare `*layout.Element` is used as a `View` via `ui.Raw`.
- `yoga.Run[T App](cfg, build func() T)` builds the app *after* `New` creates the window + text engine, because component constructors measure text and need `yoga.Text()` live.
