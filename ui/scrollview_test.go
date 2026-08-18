package ui

import (
	"testing"

	"github.com/mirzakhany/yoga/layout"
)

func TestScrollViewContentHeight(t *testing.T) {
	block := layout.New(layout.Box().H(40))
	content := layout.New(layout.Box().Direction(layout.Column).Gap(8))
	for i := 0; i < 10; i++ {
		content.Children = append(content.Children, block)
	}
	sv := NewScrollView(content)
	root := layout.New(layout.Box().FlexGrow(1), sv.host)
	root.Calculate(400, 200)

	if content.Frame.H < 470 {
		t.Fatalf("scroll content height: got %v want >= 470", content.Frame.H)
	}
	if block.Frame.H < 39 {
		t.Fatalf("block height shrunk: got %v", block.Frame.H)
	}
}
