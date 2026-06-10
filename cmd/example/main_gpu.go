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

	// Resources (text engine, sprite sheet, clipboard) are registered inside
	// yoga.New and available via yoga.Text(), yoga.Icons(), yoga.Clipboard().
	app.SetScene(BuildApp())
	app.Run() // blocks until the window closes
}
