//go:build !nogpu

package main

import (
	"github.com/mirzakhany/yoga"
	"github.com/mirzakhany/yoga/theme"
)

func main() {
	cfg := yoga.Config{
		Title:      "Todos",
		Width:      520,
		Height:     640,
		ClearColor: theme.Current().Background,
	}
	if err := yoga.Run(cfg, BuildTodoApp); err != nil {
		panic(err)
	}
}
