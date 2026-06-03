//go:build !nogpu

// This file is the ONLY place in the framework that talks to the wgpu Cgo
// bindings. It is excluded by the `nogpu` build tag so the rest of the
// framework (and the headless example) can be compiled and tested on machines
// where the native wgpu-native libraries cannot be linked.
//
// Cgo / GC boundary notes:
//   - Every wgpu.* object (Instance, Adapter, Device, Buffer, Texture, ...)
//     wraps a C pointer that the Go GC does not track. We therefore own them
//     explicitly and release them in Destroy(), and Release()/defer the
//     short-lived ones (shader modules, views, encoders) as soon as we are done.
//   - We never store a Go pointer inside a C structure. Geometry crosses the
//     boundary only as a flat []byte (wgpu.ToBytes over a Go slice) passed to
//     queue.WriteBuffer, which copies it immediately, so the GC is free to move
//     or collect the Go slice afterwards.

package render

import (
	_ "embed"
	"fmt"
	"strings"
	"unsafe"

	"github.com/cogentcore/webgpu/wgpu"
)

//go:embed shader.wgsl
var shaderSrc string

// vertexBufferLayout mirrors the Vertex struct (draw.go) and the @location
// attributes in shader.wgsl. Keeping the three in sync is the contract of the
// renderer.
var vertexBufferLayout = wgpu.VertexBufferLayout{
	ArrayStride: uint64(unsafe.Sizeof(Vertex{})),
	StepMode:    wgpu.VertexStepModeVertex,
	Attributes: []wgpu.VertexAttribute{
		{Format: wgpu.VertexFormatFloat32x2, Offset: 0, ShaderLocation: 0},
		{Format: wgpu.VertexFormatFloat32x2, Offset: 2 * 4, ShaderLocation: 1},
		{Format: wgpu.VertexFormatFloat32x4, Offset: 4 * 4, ShaderLocation: 2},
	},
}

// Renderer owns the GPU device, surface, pipeline and the growable vertex/index
// buffers used for dynamic batching.
type Renderer struct {
	instance *wgpu.Instance
	adapter  *wgpu.Adapter
	surface  *wgpu.Surface
	device   *wgpu.Device
	queue    *wgpu.Queue
	config   *wgpu.SurfaceConfiguration
	pipeline *wgpu.RenderPipeline

	uniformBuf *wgpu.Buffer
	atlasTex   *wgpu.Texture
	atlasView  *wgpu.TextureView
	sampler    *wgpu.Sampler
	bindGroup  *wgpu.BindGroup

	vertexBuf *wgpu.Buffer
	indexBuf  *wgpu.Buffer
	vcap      int // capacity of vertexBuf in vertices
	icap      int // capacity of indexBuf in indices

	// ClearColor is the framebuffer clear color (the workspace background).
	ClearColor Color
}

// NewRenderer initializes a renderer for the given platform surface descriptor
// (obtain it with wgpuglfw.GetSurfaceDescriptor(window)) and uploads the font
// atlas as an R8 coverage texture.
//
// fbW/fbH are the framebuffer size in physical pixels (window.GetFramebufferSize)
// and the surface renders at that resolution. logicalW/logicalH are the window
// size in logical points (window.GetSize); the shader's pixel->NDC uniform uses
// them so the UI is authored in logical coordinates yet rasterized at full
// device resolution (crisp text on HiDPI).
func NewRenderer(sd *wgpu.SurfaceDescriptor, fbW, fbH, logicalW, logicalH int, atlas *FontAtlas) (r *Renderer, err error) {
	defer func() {
		if err != nil {
			r.Destroy()
			r = nil
		}
	}()
	r = &Renderer{ClearColor: RGBA8(24, 24, 29, 255)}

	r.instance = wgpu.CreateInstance(nil)
	r.surface = r.instance.CreateSurface(sd)

	r.adapter, err = r.instance.RequestAdapter(&wgpu.RequestAdapterOptions{
		CompatibleSurface: r.surface,
	})
	if err != nil {
		return r, err
	}

	r.device, err = r.adapter.RequestDevice(nil)
	if err != nil {
		return r, err
	}
	r.queue = r.device.GetQueue()

	caps := r.surface.GetCapabilities(r.adapter)
	r.config = &wgpu.SurfaceConfiguration{
		Usage:       wgpu.TextureUsageRenderAttachment,
		Format:      pickFormat(caps.Formats),
		Width:       uint32(fbW),
		Height:      uint32(fbH),
		PresentMode: wgpu.PresentModeFifo,
		AlphaMode:   caps.AlphaModes[0],
	}
	r.surface.Configure(r.adapter, r.device, r.config)

	if err = r.createPipeline(); err != nil {
		return r, err
	}
	if err = r.createAtlasTexture(atlas); err != nil {
		return r, err
	}
	if err = r.createUniformAndBindGroup(logicalW, logicalH); err != nil {
		return r, err
	}
	return r, nil
}

func (r *Renderer) createPipeline() error {
	shader, err := r.device.CreateShaderModule(&wgpu.ShaderModuleDescriptor{
		Label:          "ui.wgsl",
		WGSLDescriptor: &wgpu.ShaderModuleWGSLDescriptor{Code: shaderSrc},
	})
	if err != nil {
		return err
	}
	defer shader.Release() // the pipeline retains its own reference

	r.pipeline, err = r.device.CreateRenderPipeline(&wgpu.RenderPipelineDescriptor{
		Label: "ui-pipeline",
		Vertex: wgpu.VertexState{
			Module:     shader,
			EntryPoint: "vs_main",
			Buffers:    []wgpu.VertexBufferLayout{vertexBufferLayout},
		},
		Primitive: wgpu.PrimitiveState{
			Topology:  wgpu.PrimitiveTopologyTriangleList,
			FrontFace: wgpu.FrontFaceCCW,
			CullMode:  wgpu.CullModeNone,
		},
		Multisample: wgpu.MultisampleState{
			Count: 1,
			Mask:  0xFFFFFFFF,
		},
		Fragment: &wgpu.FragmentState{
			Module:     shader,
			EntryPoint: "fs_main",
			Targets: []wgpu.ColorTargetState{{
				Format:    r.config.Format,
				Blend:     &wgpu.BlendStateAlphaBlending, // text/icons need alpha
				WriteMask: wgpu.ColorWriteMaskAll,
			}},
		},
	})
	return err
}

func (r *Renderer) createAtlasTexture(atlas *FontAtlas) error {
	extent := wgpu.Extent3D{
		Width:              uint32(atlas.W),
		Height:             uint32(atlas.H),
		DepthOrArrayLayers: 1,
	}
	tex, err := r.device.CreateTexture(&wgpu.TextureDescriptor{
		Label:         "font-atlas",
		Size:          extent,
		MipLevelCount: 1,
		SampleCount:   1,
		Dimension:     wgpu.TextureDimension2D,
		Format:        wgpu.TextureFormatR8Unorm,
		Usage:         wgpu.TextureUsageTextureBinding | wgpu.TextureUsageCopyDst,
	})
	if err != nil {
		return err
	}
	r.atlasTex = tex

	r.atlasView, err = tex.CreateView(nil)
	if err != nil {
		return err
	}

	// Copy the Go-owned coverage bytes into the texture. WriteTexture copies
	// synchronously, so atlas.Pixels can be GC'd afterwards.
	r.queue.WriteTexture(
		tex.AsImageCopy(),
		atlas.Pixels,
		&wgpu.TextureDataLayout{
			Offset:       0,
			BytesPerRow:  uint32(atlas.W), // 1 byte per pixel (R8)
			RowsPerImage: uint32(atlas.H),
		},
		&extent,
	)

	r.sampler, err = r.device.CreateSampler(&wgpu.SamplerDescriptor{
		Label:        "atlas-sampler",
		AddressModeU: wgpu.AddressModeClampToEdge,
		AddressModeV: wgpu.AddressModeClampToEdge,
		AddressModeW: wgpu.AddressModeClampToEdge,
		MagFilter:     wgpu.FilterModeLinear,
		MinFilter:     wgpu.FilterModeLinear,
		MipmapFilter:  wgpu.MipmapFilterModeNearest,
		LodMaxClamp:   1,
		MaxAnisotropy: 1, // wgpu requires >= 1
	})
	return err
}

func (r *Renderer) createUniformAndBindGroup(width, height int) error {
	// 16 bytes: vec2 screen size padded to a vec4 (uniform alignment).
	u := [4]float32{float32(width), float32(height), 0, 0}
	var err error
	r.uniformBuf, err = r.device.CreateBufferInit(&wgpu.BufferInitDescriptor{
		Label:    "uniforms",
		Contents: wgpu.ToBytes(u[:]),
		Usage:    wgpu.BufferUsageUniform | wgpu.BufferUsageCopyDst,
	})
	if err != nil {
		return err
	}

	layout := r.pipeline.GetBindGroupLayout(0)
	defer layout.Release()

	r.bindGroup, err = r.device.CreateBindGroup(&wgpu.BindGroupDescriptor{
		Label:  "ui-bind-group",
		Layout: layout,
		Entries: []wgpu.BindGroupEntry{
			{Binding: 0, Buffer: r.uniformBuf, Size: wgpu.WholeSize},
			{Binding: 1, TextureView: r.atlasView},
			{Binding: 2, Sampler: r.sampler},
		},
	})
	return err
}

// Resize reconfigures the surface to the new framebuffer size and updates the
// screen-size uniform (in logical points) so the shader's pixel->NDC mapping
// stays correct.
func (r *Renderer) Resize(fbW, fbH, logicalW, logicalH int) {
	if fbW <= 0 || fbH <= 0 {
		return
	}
	r.config.Width = uint32(fbW)
	r.config.Height = uint32(fbH)
	r.surface.Configure(r.adapter, r.device, r.config)

	u := [4]float32{float32(logicalW), float32(logicalH), 0, 0}
	r.queue.WriteBuffer(r.uniformBuf, 0, wgpu.ToBytes(u[:]))
}

// ensureCapacity (re)creates the streaming buffers if the current frame needs
// more room than is allocated. Buffers only ever grow, so a stable UI reaches a
// steady state with no further allocation. This is the "dynamic batching"
// strategy: one big buffer, rewritten each frame, instead of per-element draws.
func (r *Renderer) ensureCapacity(nVerts, nIndices int) error {
	if nVerts > r.vcap {
		if r.vertexBuf != nil {
			r.vertexBuf.Release()
		}
		newCap := grow(r.vcap, nVerts)
		buf, err := r.device.CreateBuffer(&wgpu.BufferDescriptor{
			Label: "vertices",
			Size:  uint64(newCap) * uint64(unsafe.Sizeof(Vertex{})),
			Usage: wgpu.BufferUsageVertex | wgpu.BufferUsageCopyDst,
		})
		if err != nil {
			return err
		}
		r.vertexBuf = buf
		r.vcap = newCap
	}
	if nIndices > r.icap {
		if r.indexBuf != nil {
			r.indexBuf.Release()
		}
		newCap := grow(r.icap, nIndices)
		buf, err := r.device.CreateBuffer(&wgpu.BufferDescriptor{
			Label: "indices",
			Size:  uint64(newCap) * 4,
			Usage: wgpu.BufferUsageIndex | wgpu.BufferUsageCopyDst,
		})
		if err != nil {
			return err
		}
		r.indexBuf = buf
		r.icap = newCap
	}
	return nil
}

// pickFormat selects a non-sRGB surface format when one is offered. We author
// colors as straight 8-bit sRGB values and write them to the framebuffer
// unchanged; an sRGB-typed target would re-encode them on write and wash out the
// whole UI (dark theme -> mid gray). Falling back to the preferred format keeps
// things working if only sRGB formats are available.
func pickFormat(formats []wgpu.TextureFormat) wgpu.TextureFormat {
	for _, f := range formats {
		if !strings.HasSuffix(strings.ToLower(f.String()), "srgb") {
			return f
		}
	}
	return formats[0]
}

func grow(cur, need int) int {
	n := cur
	if n == 0 {
		n = 1024
	}
	for n < need {
		n *= 2
	}
	return n
}

// Render uploads the frame's DrawList into the GPU buffers and draws the entire
// UI with a single indexed draw call.
func (r *Renderer) Render(dl *DrawList) error {
	surfaceTex, err := r.surface.GetCurrentTexture()
	if err != nil {
		return err
	}
	view, err := surfaceTex.CreateView(nil)
	if err != nil {
		return err
	}
	defer view.Release()

	encoder, err := r.device.CreateCommandEncoder(nil)
	if err != nil {
		return err
	}
	defer encoder.Release()

	pass := encoder.BeginRenderPass(&wgpu.RenderPassDescriptor{
		ColorAttachments: []wgpu.RenderPassColorAttachment{{
			View:    view,
			LoadOp:  wgpu.LoadOpClear,
			StoreOp: wgpu.StoreOpStore,
			ClearValue: wgpu.Color{
				R: float64(r.ClearColor.R),
				G: float64(r.ClearColor.G),
				B: float64(r.ClearColor.B),
				A: float64(r.ClearColor.A),
			},
		}},
	})

	if n := len(dl.Indices); n > 0 {
		if err := r.ensureCapacity(len(dl.Vertices), n); err != nil {
			pass.End()
			pass.Release()
			return err
		}
		r.queue.WriteBuffer(r.vertexBuf, 0, wgpu.ToBytes(dl.Vertices))
		r.queue.WriteBuffer(r.indexBuf, 0, wgpu.ToBytes(dl.Indices))

		pass.SetPipeline(r.pipeline)
		pass.SetBindGroup(0, r.bindGroup, nil)
		pass.SetVertexBuffer(0, r.vertexBuf, 0, wgpu.WholeSize)
		pass.SetIndexBuffer(r.indexBuf, wgpu.IndexFormatUint32, 0, wgpu.WholeSize)
		pass.DrawIndexed(uint32(n), 1, 0, 0, 0)
	}

	pass.End()
	pass.Release()

	cmd, err := encoder.Finish(nil)
	if err != nil {
		return err
	}
	defer cmd.Release()

	r.queue.Submit(cmd)
	r.surface.Present()
	return nil
}

// Destroy releases every GPU resource. Safe to call on a partially constructed
// renderer (used by NewRenderer's error path).
func (r *Renderer) Destroy() {
	if r == nil {
		return
	}
	if r.indexBuf != nil {
		r.indexBuf.Release()
		r.indexBuf = nil
	}
	if r.vertexBuf != nil {
		r.vertexBuf.Release()
		r.vertexBuf = nil
	}
	if r.bindGroup != nil {
		r.bindGroup.Release()
		r.bindGroup = nil
	}
	if r.sampler != nil {
		r.sampler.Release()
		r.sampler = nil
	}
	if r.atlasView != nil {
		r.atlasView.Release()
		r.atlasView = nil
	}
	if r.atlasTex != nil {
		r.atlasTex.Release()
		r.atlasTex = nil
	}
	if r.uniformBuf != nil {
		r.uniformBuf.Release()
		r.uniformBuf = nil
	}
	if r.pipeline != nil {
		r.pipeline.Release()
		r.pipeline = nil
	}
	if r.queue != nil {
		r.queue.Release()
		r.queue = nil
	}
	if r.device != nil {
		r.device.Release()
		r.device = nil
	}
	if r.adapter != nil {
		r.adapter.Release()
		r.adapter = nil
	}
	if r.surface != nil {
		r.surface.Release()
		r.surface = nil
	}
	if r.instance != nil {
		r.instance.Release()
		r.instance = nil
	}
}

// compile-time guard that DrawList carries 32-byte vertices as the shader expects.
var _ = func() bool {
	if unsafe.Sizeof(Vertex{}) != 32 {
		panic(fmt.Sprintf("Vertex must be 32 bytes, got %d", unsafe.Sizeof(Vertex{})))
	}
	return true
}()
