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
yoga/                (runtime: App, GLFW window, WebGPU renderer, font atlas, input, event loop)
    │
components/ theme/   (Button, Editor, Tree, Splitter, Dialog, FocusManager, …)
    │
layout/              (Element tree, flex/grid/stack solvers, 2-pass pipeline)
    │
render/              (DrawList, FontAtlas, WGSL shader, batched Renderer)
    │
input/  text/  highlight/   (leaves: mouse/kb state, piece-table, tree-sitter worker)
```

**`yoga` (runtime package)** — owns the GLFW window, WebGPU device, font atlas, and the event-driven frame loop (`WaitEvents`/`WaitEventsTimeout` for near-zero idle CPU). Apps implement the `yoga.Scene` interface. Cgo (GLFW + WebGPU) lives entirely in `app_gpu.go`; `app_stub.go` is the headless counterpart.

**`render/`** — pure Go except `renderer.go` (the only Cgo file in the render layer). `draw.go` defines `DrawList`, `Vertex`, `DrawCmd`, and draw primitives (`AddRect`, `AddRoundedRect`, `AddTexQuad`, etc.). `PushClip`/`PopClip` segment the index stream into scissor-bounded `DrawCmd`s. The `solidUV = -1` sentinel tells the WGSL shader to emit a flat fill instead of sampling the atlas. `atlas.go` manages the glyph/icon atlas; text shaping is in `shape/` (HarfBuzz-quality via `go-text/typesetting`).

**`layout/`** — `Element` is the node type (Style, Children, Frame, Paint hook, OnMouse hook, Overlay, ScrollOffset, Clip). `New(style, children...)` is the declarative constructor. `Calculate` runs the two-pass pipeline: solve relative geometry → flatten to absolute `Frame`s. `Dispatch` delivers mouse events front-to-back (overlays first); set `m.Consumed = true` to stop propagation. `Style` uses NaN for unset/auto dimensions and has fluent builders (`Box()`, `FlexGrow`, `GridCols`, `Absolute`, etc.).

**`components/`** — high-level widgets. All components build a `layout.Element` tree and expose state structs. The `FocusManager` routes Tab/Shift-Tab and Ctrl+Tab between registered focusable widgets. The `Editor` component uses piece-table storage (`text/piecetable.go`) and line-window virtualization to handle multi-MB files.

**`highlight/`** — async Tree-sitter syntax highlighting on a worker goroutine. Incremental reparsing via `UpdateEdit` + `tree.Edit` on each keystroke. Go and JSON grammars are wired; unknown file types fall back to plain text.

## Frame loop data flow

```
GLFW callbacks → input.Mouse / input.Keyboard
app.Run:
  scene.Layout(w,h)  → layout.Calculate → flatten to Frame
  scene.Update(m,kb) → layout.Dispatch(mouse) → OnMouse hooks
                     → FocusManager.Route(kb) → focused widget HandleKeys
                     → Editor.Update() → PieceTable + highlight.UpdateEdit
  layout.Paint(root) → DrawList
  renderer.Render(dl) → single buffer upload → one indexed draw per DrawCmd
  glfw.WaitEvents(Timeout)  ← AnimationWait() when scene animates (caret blink, etc.)
```

## Build tags

- `//go:build !nogpu` — real GPU/GLFW implementation (default)
- `//go:build nogpu`  — headless stubs; no Cgo, no window; used for CI and machines without a GPU stack

Files ending in `_gpu.go` and `_stub.go` follow this pattern throughout the codebase.

## Key design decisions

- Cgo is restricted to two locations: `yoga/app_gpu.go` (GLFW + WebGPU) and `render/renderer.go` (WebGPU device/pipeline). Everything else is pure Go.
- `render.DrawList` is plain Go memory; components and layout build it without touching Cgo.
- `cogentcore/webgpu` was chosen over `rajveermalviya/go-webgpu` for better maintenance; `go-tree-sitter` (official) over `smacker/go-tree-sitter`.
- The layout engine is intentionally swappable via the `layout.Engine` interface; the default is `customEngine` (flex/grid/stack in pure Go, no Cgo Yoga).
