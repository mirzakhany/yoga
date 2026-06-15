//go:build !nogpu

package main

import (
	"github.com/mirzakhany/yoga"
	"github.com/mirzakhany/yoga/theme"
)

func main() {
	app, err := yoga.New(yoga.Config{
		Title:      "Yoga UI — API Test",
		Width:      900,
		Height:     700,
		ClearColor: theme.Current().Background,
	})
	if err != nil {
		panic(err)
	}
	defer app.Close()

	app.SetScene(BuildAPITestApp())
	app.Run()
}
