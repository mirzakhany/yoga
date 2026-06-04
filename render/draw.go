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

import "math"

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

// DrawCmd is a contiguous run of indices that share one scissor rectangle. The
// renderer issues one SetScissorRect + DrawIndexed per command. A command whose
// Clip.W is negative means "no clip" (draw to the whole surface).
type DrawCmd struct {
	Clip       Rect // logical-pixel scissor; Clip.W < 0 means full surface
	IndexStart int  // offset into DrawList.Indices
	IndexCount int  // number of indices in this run
}

// noClip is the sentinel "draw everywhere" rectangle (negative size).
var noClip = Rect{X: 0, Y: 0, W: -1, H: -1}

// DrawList is a per-frame, append-only batch of geometry. Every widget in the
// tree writes into one shared DrawList. In the common (unclipped) case the whole
// UI is one draw call; PushClip/PopClip split the stream into a few commands so
// scrolled content can be scissored to its viewport.
type DrawList struct {
	Vertices []Vertex
	Indices  []uint32

	// Commands segments Indices into scissor runs. When empty, the renderer
	// draws all indices with no scissor (the fast path).
	Commands  []DrawCmd
	clipStack []Rect
}

// Reset clears the lists while retaining their backing arrays, so a steady-state
// UI performs zero per-frame heap allocation for geometry.
func (d *DrawList) Reset() {
	d.Vertices = d.Vertices[:0]
	d.Indices = d.Indices[:0]
	d.Commands = d.Commands[:0]
	d.clipStack = d.clipStack[:0]
}

// PushClip restricts subsequent geometry to r (intersected with any clip already
// in effect), in logical pixels. Always pair with PopClip.
func (d *DrawList) PushClip(r Rect) {
	if n := len(d.clipStack); n > 0 {
		r = intersect(d.clipStack[n-1], r)
	}
	d.clipStack = append(d.clipStack, r)
}

// PopClip removes the most recent clip rectangle.
func (d *DrawList) PopClip() {
	if n := len(d.clipStack); n > 0 {
		d.clipStack = d.clipStack[:n-1]
	}
}

// curClip returns the active clip rectangle, or noClip if the stack is empty.
func (d *DrawList) curClip() Rect {
	if n := len(d.clipStack); n > 0 {
		return d.clipStack[n-1]
	}
	return noClip
}

// ensureCmd returns the command that new indices should extend, starting a new
// one whenever the active clip differs from the last command's clip. It must be
// called BEFORE the quad's indices are appended so a freshly started command
// captures the correct IndexStart (the current end of Indices).
func (d *DrawList) ensureCmd() *DrawCmd {
	clip := d.curClip()
	if n := len(d.Commands); n > 0 {
		last := &d.Commands[n-1]
		if last.Clip == clip {
			return last
		}
	}
	d.Commands = append(d.Commands, DrawCmd{Clip: clip, IndexStart: len(d.Indices)})
	return &d.Commands[len(d.Commands)-1]
}

// intersect returns the overlap of two rectangles (zero-size if disjoint).
func intersect(a, b Rect) Rect {
	x0 := f32maxr(a.X, b.X)
	y0 := f32maxr(a.Y, b.Y)
	x1 := f32minr(a.X+a.W, b.X+b.W)
	y1 := f32minr(a.Y+a.H, b.Y+b.H)
	if x1 < x0 {
		x1 = x0
	}
	if y1 < y0 {
		y1 = y0
	}
	return Rect{X: x0, Y: y0, W: x1 - x0, H: y1 - y0}
}

func f32maxr(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}

func f32minr(a, b float32) float32 {
	if a < b {
		return a
	}
	return b
}

// quad appends two triangles (4 vertices, 6 indices) describing a rectangle.
// uv values of solidUV select the flat-color path in the shader.
func (d *DrawList) quad(r Rect, uv Rect, c Color) {
	base := uint32(len(d.Vertices))
	col := [4]float32{c.R, c.G, c.B, c.A}

	// Reserve/extend the scissor command BEFORE appending indices so a newly
	// started command records the correct IndexStart (the offset of this quad's
	// first index). Doing it afterwards would push IndexStart 6 indices too far,
	// making the run over-read stale index-buffer data from a previous frame.
	cmd := d.ensureCmd()

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
	cmd.IndexCount += 6
}

// AddRect appends a flat-colored rectangle.
func (d *DrawList) AddRect(r Rect, c Color) {
	d.quad(r, Rect{solidUV, solidUV, 0, 0}, c)
}

const roundedSegments = 8

func clampRadius(r Rect, radius float32) float32 {
	maxR := r.W
	if r.H < maxR {
		maxR = r.H
	}
	maxR /= 2
	if radius > maxR {
		return maxR
	}
	if radius < 0 {
		return 0
	}
	return radius
}

// addSolidTriangle appends one filled triangle (flat color).
func (d *DrawList) addSolidTriangle(x0, y0, x1, y1, x2, y2 float32, c Color) {
	base := uint32(len(d.Vertices))
	col := [4]float32{c.R, c.G, c.B, c.A}
	uv := [2]float32{solidUV, solidUV}
	cmd := d.ensureCmd()
	d.Vertices = append(d.Vertices,
		Vertex{Pos: [2]float32{x0, y0}, UV: uv, Color: col},
		Vertex{Pos: [2]float32{x1, y1}, UV: uv, Color: col},
		Vertex{Pos: [2]float32{x2, y2}, UV: uv, Color: col},
	)
	d.Indices = append(d.Indices, base, base+1, base+2)
	cmd.IndexCount += 3
}

func (d *DrawList) addCornerFan(cx, cy, radius float32, start, end float32, c Color) {
	if radius <= 0 {
		return
	}
	for i := 0; i < roundedSegments; i++ {
		t0 := float32(i) / float32(roundedSegments)
		t1 := float32(i+1) / float32(roundedSegments)
		a0 := start + (end-start)*t0
		a1 := start + (end-start)*t1
		x0 := cx + radius*float32(math.Cos(float64(a0)))
		y0 := cy + radius*float32(math.Sin(float64(a0)))
		x1 := cx + radius*float32(math.Cos(float64(a1)))
		y1 := cy + radius*float32(math.Sin(float64(a1)))
		d.addSolidTriangle(cx, cy, x0, y0, x1, y1, c)
	}
}

// AddRoundedRect appends a flat-filled rectangle with rounded corners.
func (d *DrawList) AddRoundedRect(r Rect, radius float32, c Color) {
	radius = clampRadius(r, radius)
	if radius <= 0 {
		d.AddRect(r, c)
		return
	}
	x, y, w, h := r.X, r.Y, r.W, r.H
	d.quad(Rect{X: x + radius, Y: y + radius, W: w - 2*radius, H: h - 2*radius}, Rect{solidUV, solidUV, 0, 0}, c)
	d.quad(Rect{X: x + radius, Y: y, W: w - 2*radius, H: radius}, Rect{solidUV, solidUV, 0, 0}, c)
	d.quad(Rect{X: x + radius, Y: y + h - radius, W: w - 2*radius, H: radius}, Rect{solidUV, solidUV, 0, 0}, c)
	d.quad(Rect{X: x, Y: y + radius, W: radius, H: h - 2*radius}, Rect{solidUV, solidUV, 0, 0}, c)
	d.quad(Rect{X: x + w - radius, Y: y + radius, W: radius, H: h - 2*radius}, Rect{solidUV, solidUV, 0, 0}, c)
	const pi = float32(math.Pi)
	d.addCornerFan(x+radius, y+radius, radius, pi, 3*pi/2, c)
	d.addCornerFan(x+w-radius, y+radius, radius, 3*pi/2, 2*pi, c)
	d.addCornerFan(x+w-radius, y+h-radius, radius, 0, pi/2, c)
	d.addCornerFan(x+radius, y+h-radius, radius, pi/2, pi, c)
}

// AddRoundedRectBorder draws a rounded fill with a rounded border (inset fill).
func (d *DrawList) AddRoundedRectBorder(r Rect, radius, borderWidth float32, fill, border Color) {
	if borderWidth <= 0 {
		d.AddRoundedRect(r, radius, fill)
		return
	}
	d.AddRoundedRect(r, radius, border)
	inner := Rect{
		X: r.X + borderWidth, Y: r.Y + borderWidth,
		W: r.W - 2*borderWidth, H: r.H - 2*borderWidth,
	}
	innerRadius := radius - borderWidth
	if innerRadius < 0 {
		innerRadius = 0
	}
	if inner.W > 0 && inner.H > 0 {
		d.AddRoundedRect(inner, innerRadius, fill)
	}
}

// AddTexQuad appends a textured rectangle, sampling the atlas region described
// by uv (in 0..1 atlas space) and tinting it by c. Used for glyphs and icons.
func (d *DrawList) AddTexQuad(dst Rect, uv Rect, c Color) {
	d.quad(dst, uv, c)
}
