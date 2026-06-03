// UI shader: consumes screen-space pixel coordinates and converts them to
// WebGPU Normalized Device Coordinates (NDC) entirely on the GPU, using the
// framebuffer size supplied through a uniform buffer. This means the CPU side
// only ever works in intuitive top-left pixel space.

struct Uniforms {
    // .xy = framebuffer size in pixels; .zw = padding for 16-byte alignment.
    screen : vec4<f32>,
};

@group(0) @binding(0) var<uniform> u : Uniforms;
@group(0) @binding(1) var atlasTex : texture_2d<f32>;
@group(0) @binding(2) var atlasSampler : sampler;

struct VSOut {
    @builtin(position) clip : vec4<f32>,
    @location(0) uv : vec2<f32>,
    @location(1) color : vec4<f32>,
};

@vertex
fn vs_main(
    @location(0) pos : vec2<f32>,
    @location(1) uv : vec2<f32>,
    @location(2) color : vec4<f32>,
) -> VSOut {
    var out : VSOut;
    // Pixel -> NDC: x in [0,w] maps to [-1,1]; y is flipped because pixel space
    // grows downward while NDC grows upward.
    let ndcX = pos.x / u.screen.x * 2.0 - 1.0;
    let ndcY = 1.0 - pos.y / u.screen.y * 2.0;
    out.clip = vec4<f32>(ndcX, ndcY, 0.0, 1.0);
    out.uv = uv;
    out.color = color;
    return out;
}

@fragment
fn fs_main(in : VSOut) -> @location(0) vec4<f32> {
    // Negative UV is the sentinel for a flat-filled quad (rectangles, button
    // backgrounds, scrollbar thumbs, ...).
    if (in.uv.x < 0.0) {
        return in.color;
    }
    // Otherwise sample the atlas. Glyphs/icons are stored as single-channel
    // coverage in the red channel; the vertex color provides the tint, so the
    // final alpha is coverage * color.a.
    let coverage = textureSample(atlasTex, atlasSampler, in.uv).r;
    return vec4<f32>(in.color.rgb, in.color.a * coverage);
}
