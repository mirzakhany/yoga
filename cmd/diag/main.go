package main

import (
	"fmt"
	"os"

	"github.com/mirzakhany/yoga"
	"github.com/mirzakhany/yoga/input"
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
	inner1 := ui.NewCard("Sample Card", "A surfaced container", galleryCardBody())
	inner2 := ui.NewCard("Elevated Card", "Stronger drop shadow", galleryCardBody()).Elevated()
	alert1 := ui.NewAlert("This is an informational alert.", ui.AlertInfo)
	alert2 := ui.NewAlert("Warning: unsaved changes.", ui.AlertWarning)
	alert3 := ui.NewAlert("Error: could not connect.", ui.AlertError)
	alert4 := ui.NewAlert("Success: file saved.", ui.AlertSuccess)
	surf := ui.NewCard("Surfaces", "Cards and alerts", ui.Column(
		ui.ViewOf(inner1),
		ui.ViewOf(inner2),
		ui.ViewOf(alert1),
		ui.ViewOf(alert2),
		ui.ViewOf(alert3),
		ui.ViewOf(alert4),
	).Gap(th.Spacing.S))

	c := ui.New(text, ui.NewFocusScope(), nil)
	c.SetIcons(sheet)
	mouse := &input.Mouse{}
	kb := &input.Keyboard{}
	_ = ui.BuildFrame(c, func(_ *ui.Ctx) ui.View {
		return ui.Column(ui.ViewOf(surf)).Padding(th.Spacing.L)
	}, 1200, 2000, mouse, kb)

	fmt.Printf("surfCard:     Y=%.1f H=%.1f (bottom=%.1f)\n", surf.El.Frame.Y, surf.El.Frame.H, surf.El.Frame.Y+surf.El.Frame.H)
	fmt.Printf("innerCard1:   Y=%.1f H=%.1f (bottom=%.1f)\n", inner1.El.Frame.Y, inner1.El.Frame.H, inner1.El.Frame.Y+inner1.El.Frame.H)
	fmt.Printf("innerCard2:   Y=%.1f H=%.1f (bottom=%.1f)\n", inner2.El.Frame.Y, inner2.El.Frame.H, inner2.El.Frame.Y+inner2.El.Frame.H)
	fmt.Printf("alert1(info): Y=%.1f H=%.1f (bottom=%.1f)\n", alert1.El.Frame.Y, alert1.El.Frame.H, alert1.El.Frame.Y+alert1.El.Frame.H)
	fmt.Printf("alert2(warn): Y=%.1f H=%.1f (bottom=%.1f)\n", alert2.El.Frame.Y, alert2.El.Frame.H, alert2.El.Frame.Y+alert2.El.Frame.H)
	fmt.Printf("alert3(err):  Y=%.1f H=%.1f (bottom=%.1f)\n", alert3.El.Frame.Y, alert3.El.Frame.H, alert3.El.Frame.Y+alert3.El.Frame.H)
	fmt.Printf("alert4(succ): Y=%.1f H=%.1f (bottom=%.1f)\n", alert4.El.Frame.Y, alert4.El.Frame.H, alert4.El.Frame.Y+alert4.El.Frame.H)
}

func galleryCardBody() ui.View {
	return ui.Text("Card body content goes here.").Padding(theme.Current().Spacing.S)
}
