//go:build !nogpu

package main

import (
	"github.com/mirzakhany/yoga"
	"github.com/mirzakhany/yoga/theme"
)

func main() {
	cfg := yoga.Config{
		Title:      "Yoga UI — Coding Workspace",
		Width:      1100,
		Height:     720,
		ClearColor: theme.Current().Background,
	}
	// Resources (text engine, sprite sheet, clipboard) are created by Run before
	// it builds the app, so widget constructors can measure text.
	if err := yoga.Run(cfg, BuildApp); err != nil {
		panic(err)
	}
}
