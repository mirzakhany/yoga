package main

import "strings"

// sampleSource is the Go code shown in the editor. It is repeated a few times so
// the document is taller than the viewport, demonstrating scrolling and line
// virtualization (only on-screen lines are turned into GPU vertices).
var sampleSource = buildSample()

func buildSample() []byte {
	const block = `package workspace

import (
	"fmt"
	"strings"
)

// Renderer batches every widget into a single draw call.
// Unicode: 日本語 مرحبا 🎉 ligatures: fi fl
type Renderer struct {
	vertices []Vertex // streamed to the GPU each frame
	indices  []uint32
}

// Frame uploads the draw list and issues one indexed draw.
func (r *Renderer) Frame(list *DrawList) error {
	if len(list.indices) == 0 {
		return nil
	}
	const greeting = "hello, webgpu"
	count := 0
	for i := 0; i < 42; i++ {
		count += i * 2
	}
	fmt.Println(strings.ToUpper(greeting), count)
	return nil
}
`
	var b strings.Builder
	for i := 0; i < 6; i++ {
		b.WriteString(block)
		b.WriteString("\n")
	}
	return []byte(b.String())
}
