package main

import (
	"fmt"
	"os"

	"github.com/mirzakhany/yoga"
	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
	"github.com/mirzakhany/yoga/theme"
	"github.com/mirzakhany/yoga/ui"
)

func main() {
	text, err := shape.NewEngine(1, false)
	if err != nil {
		fmt.Println("engine error:", err)
		os.Exit(1)
	}
	sheet := render.NewSpriteSheet(text.Atlas)
	yoga.SetResources(text, sheet, nil)

	th := theme.Current()
	c := ui.New(text, ui.NewFocusScope(), nil)
	c.SetIcons(sheet)
	mouse := &input.Mouse{}
	kb := &input.Keyboard{}
	root := ui.BuildFrame(c, func(_ *ui.Ctx) ui.View {
		return ui.Column(
			ui.Card("Surfaces", "Cards and alerts", ui.Column(
				ui.Card("Sample Card", "A surfaced container", galleryCardBody()),
				ui.Card("Elevated Card", "Stronger drop shadow", galleryCardBody()).Elevated(),
				ui.Alert("This is an informational alert.", ui.AlertInfo),
				ui.Alert("Warning: unsaved changes.", ui.AlertWarning),
				ui.Alert("Error: could not connect.", ui.AlertError),
				ui.Alert("Success: file saved.", ui.AlertSuccess),
			).Gap(th.Spacing.S)),
		).Padding(th.Spacing.L)
	}, 1200, 2000, mouse, kb)

	surf := root.Children[0]
	body := surf.Children[len(surf.Children)-1]
	printEl("surfCard", surf)
	for i, name := range []string{"innerCard1", "innerCard2", "alert1(info)", "alert2(warn)", "alert3(err)", "alert4(succ)"} {
		if i >= len(body.Children) {
			fmt.Printf("%s: missing\n", name)
			continue
		}
		printEl(name, body.Children[i])
	}
}

func printEl(name string, el *layout.Element) {
	fmt.Printf("%-14s Y=%.1f H=%.1f (bottom=%.1f)\n", name+":", el.Frame.Y, el.Frame.H, el.Frame.Y+el.Frame.H)
}

func galleryCardBody() ui.View {
	return ui.Text("Card body content goes here.").Padding(theme.Current().Spacing.S)
}
