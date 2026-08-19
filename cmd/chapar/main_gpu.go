//go:build !nogpu

package main

import (
	"github.com/mirzakhany/yoga"
	"github.com/mirzakhany/yoga/theme"
)

func main() {
	// Select the theme before Run so the first frame matches the scene.
	theme.Use("github-dark")

	cfg := yoga.Config{
		Title:  "Chapar",
		Width:  900,
		Height: 700,
	}
	if err := yoga.Run(cfg, BuildChaparApp); err != nil {
		panic(err)
	}
}
