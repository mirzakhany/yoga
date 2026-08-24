package main

import (
	"bytes"
	"embed"
	"image"
	"image/color"
	"image/png"
)

//go:embed testdata/sample.png
var samplePNG []byte

//go:embed testdata/logo.svg
var sampleSVG []byte

//go:embed testdata/mark.svg
var markSVG []byte

//go:embed testdata
var imageAssets embed.FS

// checkerPNG is a small 64×64 checkerboard generated once at init.
var checkerPNG []byte

func init() {
	checkerPNG = encodeCheckerPNG(64, 64)
}

func encodeCheckerPNG(w, h int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if (x/8+y/8)%2 == 0 {
				img.SetRGBA(x, y, color.RGBA{R: 240, G: 120, B: 60, A: 255})
			} else {
				img.SetRGBA(x, y, color.RGBA{R: 40, G: 40, B: 48, A: 255})
			}
		}
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}
