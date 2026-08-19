//go:build !nogpu

package main

import (
	"github.com/mirzakhany/yoga"
)

func main() {
	cfg := yoga.Config{
		Title:  "Todos",
		Width:  520,
		Height: 640,
	}
	if err := yoga.Run(cfg, BuildTodoApp); err != nil {
		panic(err)
	}
}
