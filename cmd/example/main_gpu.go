//go:build !nogpu

package main

import (
	"github.com/mirzakhany/yoga"
	"github.com/mirzakhany/yoga/theme"
)

func main() {
	app, err := yoga.New(yoga.Config{
		Title:      "Yoga UI — Coding Workspace",
		Width:      1100,
		Height:     720,
		ClearColor: theme.Current().Background,
	})
	if err != nil {
		panic(err)
	}
	defer app.Close()

	// The scene reads colors from the live active theme and reports the current
	// clear color each frame, so switching themes at runtime is immediate.
	app.SetScene(BuildApp(app.Text(), app.Clipboard()))
	app.Run() // blocks until the window closes
}
