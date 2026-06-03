// Package render contains the GPU-facing layer of the framework: the shared
// CPU-side draw primitives (this file), the monospace font atlas, the icon
// sprite sheet, and the batched WebGPU renderer.
//
// IMPORTANT: only this package imports the wgpu Cgo bindings, and only in the
// build-tagged files (renderer.go / renderer_stub.go). Everything in this file
// is pure Go so that the layout, components, input and text packages can build
// a frame's geometry without ever touching Cgo. This keeps the GC/Cgo boundary
// small and well defined: components produce a DrawList (pure Go memory owned by
// the GC), and the renderer is the single place that copies that memory across
// into GPU buffers via queue.WriteBuffer.
package render

// Color is a straight (non-premultiplied) RGBA color in the 0..1 range.
type Color struct {
	R, G, B, A float32
}

// RGBA8 builds a Color from 0..255 components, the form most theme palettes use.
func RGBA8(r, g, b, a uint8) Color {
	return Color{float32(r) / 255, float32(g) / 255, float32(b) / 255, float32(a) / 255}
}

// Rect is an axis-aligned rectangle. It is used both for screen-space pixel
// geometry (X, Y in pixels from the top-left) and, separately, for normalized
// 0..1 texture coordinate regions in the atlas.
type Rect struct {
	X, Y, W, H float32
}

// Contains reports whether the pixel point (px, py) is inside the rectangle.
func (r Rect) Contains(px, py float32) bool {
	return px >= r.X && px <= r.X+r.W && py >= r.Y && py <= r.Y+r.H
}

// solidUV is the sentinel UV.x value that tells the fragment shader to emit a
// flat color instead of sampling the atlas texture. Any negative value works;
// the shader only checks the sign.
const solidUV = -1.0

// Vertex is the single interleaved vertex format streamed to the GPU. The
// layout (2+2+4 float32 = 32 bytes) is mirrored exactly by VertexBufferLayout
// in renderer.go and by the @location bindings in shader.wgsl.
type Vertex struct {
	Pos   [2]float32 // screen pixel coordinates (top-left origin)
	UV    [2]float32 // atlas texture coords, or {solidUV, solidUV} for a flat fill
	Color [4]float32 // straight RGBA, used as fill color or glyph tint
}

// DrawList is a per-frame, append-only batch of geometry. Every widget in the
// tree writes into one shared DrawList, so the whole UI is uploaded and drawn
// with a single indexed draw call rather than one draw per element.
type DrawList struct {
	Vertices []Vertex
	Indices  []uint32
}

// Reset clears the lists while retaining their backing arrays, so a steady-state
// UI performs zero per-frame heap allocation for geometry.
func (d *DrawList) Reset() {
	d.Vertices = d.Vertices[:0]
	d.Indices = d.Indices[:0]
}

// quad appends two triangles (4 vertices, 6 indices) describing a rectangle.
// uv values of solidUV select the flat-color path in the shader.
func (d *DrawList) quad(r Rect, uv Rect, c Color) {
	base := uint32(len(d.Vertices))
	col := [4]float32{c.R, c.G, c.B, c.A}

	d.Vertices = append(d.Vertices,
		Vertex{Pos: [2]float32{r.X, r.Y}, UV: [2]float32{uv.X, uv.Y}, Color: col},
		Vertex{Pos: [2]float32{r.X + r.W, r.Y}, UV: [2]float32{uv.X + uv.W, uv.Y}, Color: col},
		Vertex{Pos: [2]float32{r.X + r.W, r.Y + r.H}, UV: [2]float32{uv.X + uv.W, uv.Y + uv.H}, Color: col},
		Vertex{Pos: [2]float32{r.X, r.Y + r.H}, UV: [2]float32{uv.X, uv.Y + uv.H}, Color: col},
	)
	d.Indices = append(d.Indices,
		base, base+1, base+2,
		base, base+2, base+3,
	)
}

// AddRect appends a flat-colored rectangle.
func (d *DrawList) AddRect(r Rect, c Color) {
	d.quad(r, Rect{solidUV, solidUV, 0, 0}, c)
}

// AddTexQuad appends a textured rectangle, sampling the atlas region described
// by uv (in 0..1 atlas space) and tinting it by c. Used for glyphs and icons.
func (d *DrawList) AddTexQuad(dst Rect, uv Rect, c Color) {
	d.quad(dst, uv, c)
}
