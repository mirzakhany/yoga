//go:build !nogpu

package main

import (
	"github.com/mirzakhany/yoga"
)

func main() {
	cfg := yoga.Config{
		Title:  "Yoga UI — Coding Workspace",
		Width:  1100,
		Height: 720,
	}
	// Resources (text engine, sprite sheet, clipboard) are created by Run before
	// it builds the app, so widget constructors can measure text.
	if err := yoga.Run(cfg, BuildApp); err != nil {
		panic(err)
	}
}
