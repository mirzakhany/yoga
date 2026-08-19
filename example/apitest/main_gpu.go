//go:build !nogpu

package main

import (
	"github.com/mirzakhany/yoga"
	"github.com/mirzakhany/yoga/theme"
)

func main() {
	// Select the theme before Run so the first frame matches the scene.
	theme.Use("yoga-midnight")

	cfg := yoga.Config{
		Title:  "Yoga UI — API Test",
		Width:  1100,
		Height: 720,
	}
	if err := yoga.Run(cfg, BuildAPITestApp); err != nil {
		panic(err)
	}
}
