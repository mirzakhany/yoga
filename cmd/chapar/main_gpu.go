//go:build !nogpu

package main

import (
	"github.com/mirzakhany/yoga"
	"github.com/mirzakhany/yoga/theme"
)

func main() {
	// Select the theme before reading ClearColor so the GPU clear matches the
	// color the scene paints its root with (otherwise the window clears with the
	// default theme's background and the header band reads a different shade).
	theme.Use("yoga-dark")

	app, err := yoga.New(yoga.Config{
		Title:      "Chapar",
		Width:      900,
		Height:     700,
		ClearColor: theme.Current().Background,
	})
	if err != nil {
		panic(err)
	}
	defer app.Close()

	app.SetScene(BuildChaparApp())
	app.Run()
}
