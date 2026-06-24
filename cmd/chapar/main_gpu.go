//go:build !nogpu

package main

import (
	"github.com/mirzakhany/yoga"
	"github.com/mirzakhany/yoga/theme"
)

func main() {
	// Select the theme before building Config so the GPU clear matches the color
	// the scene paints its root with.
	theme.Use("yoga-dark")

	cfg := yoga.Config{
		Title:      "Chapar",
		Width:      900,
		Height:     700,
		ClearColor: theme.Current().Background,
	}
	if err := yoga.Run(cfg, BuildChaparApp); err != nil {
		panic(err)
	}
}
