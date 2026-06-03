//go:build nogpu

// Stub renderer compiled with `-tags nogpu`. It implements the same surface as
// the real renderer (minus the wgpu-typed constructor, which only the GPU
// example calls) so the rest of the framework builds and runs headless on
// machines where wgpu-native cannot be linked.

package render

// Renderer is a no-op stand-in for the WebGPU renderer.
type Renderer struct {
	ClearColor Color
}

// Render counts the geometry but performs no GPU work.
func (r *Renderer) Render(dl *DrawList) error { return nil }

// Resize is a no-op.
func (r *Renderer) Resize(fbW, fbH, logicalW, logicalH int) {}

// Destroy is a no-op.
func (r *Renderer) Destroy() {}
