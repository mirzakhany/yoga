//go:build !nogpu

package main

import (
	"github.com/mirzakhany/yoga"
)

func main() {
	cfg := yoga.Config{
		Title:          "Yoga Components",
		Width:          1100,
		Height:         720,
		CustomTitleBar: true,
	}
	if err := yoga.Run(cfg, BuildCatalog); err != nil {
		panic(err)
	}
}
