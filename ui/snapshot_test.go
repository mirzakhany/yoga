package ui

import (
	"image"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
)

// A CPU rasterizer for DrawLists, so tests can save what a frame looks like.
// It is slow and approximate (no MSAA, nearest-texel sampling) and only for
// eyeballing layout: set YOGA_SNAPSHOT_DIR to write PNGs.

// rasterize paints dl at scale (device pixels per logical pixel).
func rasterize(dl *render.DrawList, text *shape.Engine, w, h int, scale float32, bg render.Color) *image.RGBA {
	W, H := int(float32(w)*scale), int(float32(h)*scale)
	img := image.NewRGBA(image.Rect(0, 0, W, H))
	buf := make([][4]float32, W*H)
	for i := range buf {
		buf[i] = [4]float32{bg.R, bg.G, bg.B, 1}
	}
	mono := text.Atlas.MonoPixels()
	mw, mh := text.Atlas.MonoSize()
	cpix := text.Atlas.ColorPixels()
	cw, ch := text.Atlas.ColorSize()
	sample := func(v render.Vertex, u, vv float32) [4]float32 {
		c := v.Col
		switch {
		case v.UV[0] < 0 || v.Page == 0:
			return c
		case v.Page == 1 && mw > 0:
			x := clampInt(int(u*float32(mw)), 0, mw-1)
			y := clampInt(int(vv*float32(mh)), 0, mh-1)
			c[3] *= float32(mono[y*mw+x]) / 255
		case v.Page == 2 && cw > 0:
			x := clampInt(int(u*float32(cw)), 0, cw-1)
			y := clampInt(int(vv*float32(ch)), 0, ch-1)
			p := cpix[(y*cw+x)*4:]
			a := float32(p[3]) / 255
			if a > 0 {
				c = [4]float32{c[0] * float32(p[0]) / 255 / a, c[1] * float32(p[1]) / 255 / a, c[2] * float32(p[2]) / 255 / a, c[3] * a}
			} else {
				c[3] = 0
			}
		}
		return c
	}
	for _, cmd := range dl.Commands {
		clip := image.Rect(0, 0, W, H)
		if cmd.Clip.W >= 0 {
			clip = clip.Intersect(image.Rect(int(cmd.Clip.X*scale), int(cmd.Clip.Y*scale),
				int((cmd.Clip.X+cmd.Clip.W)*scale), int((cmd.Clip.Y+cmd.Clip.H)*scale)))
		}
		for i := cmd.IndexStart; i+2 < cmd.IndexStart+cmd.IndexCount; i += 3 {
			a, b, c := dl.Vertices[dl.Indices[i]], dl.Vertices[dl.Indices[i+1]], dl.Vertices[dl.Indices[i+2]]
			ax, ay := a.Pos[0]*scale, a.Pos[1]*scale
			bx, by := b.Pos[0]*scale, b.Pos[1]*scale
			cx, cy := c.Pos[0]*scale, c.Pos[1]*scale
			area := (bx-ax)*(cy-ay) - (by-ay)*(cx-ax)
			if area == 0 {
				continue
			}
			x0 := max(clip.Min.X, int(math.Floor(float64(min(ax, bx, cx)))))
			x1 := min(clip.Max.X, int(math.Ceil(float64(max(ax, bx, cx)))))
			y0 := max(clip.Min.Y, int(math.Floor(float64(min(ay, by, cy)))))
			y1 := min(clip.Max.Y, int(math.Ceil(float64(max(ay, by, cy)))))
			for y := y0; y < y1; y++ {
				for x := x0; x < x1; x++ {
					px, py := float32(x)+0.5, float32(y)+0.5
					w0 := ((bx-px)*(cy-py) - (by-py)*(cx-px)) / area
					w1 := ((cx-px)*(ay-py) - (cy-py)*(ax-px)) / area
					w2 := 1 - w0 - w1
					if w0 < 0 || w1 < 0 || w2 < 0 {
						continue
					}
					u := w0*a.UV[0] + w1*b.UV[0] + w2*c.UV[0]
					v := w0*a.UV[1] + w1*b.UV[1] + w2*c.UV[1]
					col := sample(a, u, v)
					d := &buf[y*W+x]
					al := col[3]
					for k := 0; k < 3; k++ {
						d[k] = d[k]*(1-al) + col[k]*al
					}
				}
			}
		}
	}
	for i, p := range buf {
		img.Pix[i*4] = uint8(clampf(p[0], 0, 1) * 255)
		img.Pix[i*4+1] = uint8(clampf(p[1], 0, 1) * 255)
		img.Pix[i*4+2] = uint8(clampf(p[2], 0, 1) * 255)
		img.Pix[i*4+3] = 255
	}
	return img
}

// saveSnapshot writes img to $YOGA_SNAPSHOT_DIR/name, cropped to crop when
// it is non-empty.
func saveSnapshot(t *testing.T, name string, img *image.RGBA, crop image.Rectangle) {
	t.Helper()
	dir := os.Getenv("YOGA_SNAPSHOT_DIR")
	if dir == "" {
		return
	}
	var out image.Image = img
	if !crop.Empty() {
		out = img.SubImage(crop.Intersect(img.Bounds()))
	}
	f, err := os.Create(filepath.Join(dir, name))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, out); err != nil {
		t.Fatal(err)
	}
	t.Logf("wrote %s", f.Name())
}
