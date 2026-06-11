package main

import (
	"fmt"
	"os"

	"github.com/mirzakhany/yoga"
	"github.com/mirzakhany/yoga/components"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
	"github.com/mirzakhany/yoga/theme"
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

	// Build Surfaces section like gallery.go does
	innerCard1 := components.NewCard("Sample Card", "A surfaced container", galleryCardBody(th)).El
	innerCard2 := components.NewCard("Elevated Card", "Stronger drop shadow", galleryCardBody(th)).Elevated().El

	alert1 := components.NewAlert("This is an informational alert.", components.AlertInfo).El
	alert2 := components.NewAlert("Warning: unsaved changes.", components.AlertWarning).El
	alert3 := components.NewAlert("Error: could not connect.", components.AlertError).El
	alert4 := components.NewAlert("Success: file saved.", components.AlertSuccess).El

	// Reproduce galleryCard
	col := layout.VStack(th.Spacing.S, innerCard1, innerCard2, alert1, alert2, alert3, alert4)
	surfCard := components.NewCard("Surfaces", "Cards and alerts", col)

	// Wrap in sections-like container
	root := layout.New(layout.Box().Direction(layout.Column).PaddingAll(th.Spacing.L),
		surfCard.El,
	)
	root.Calculate(1200, 2000)

	fmt.Printf("surfCard.El:  Y=%.1f H=%.1f (bottom=%.1f)\n", surfCard.El.Frame.Y, surfCard.El.Frame.H, surfCard.El.Frame.Y+surfCard.El.Frame.H)
	fmt.Printf("innerCard1:   Y=%.1f H=%.1f (bottom=%.1f)\n", innerCard1.Frame.Y, innerCard1.Frame.H, innerCard1.Frame.Y+innerCard1.Frame.H)
	fmt.Printf("innerCard2:   Y=%.1f H=%.1f (bottom=%.1f)\n", innerCard2.Frame.Y, innerCard2.Frame.H, innerCard2.Frame.Y+innerCard2.Frame.H)
	fmt.Printf("alert1(info): Y=%.1f H=%.1f (bottom=%.1f)\n", alert1.Frame.Y, alert1.Frame.H, alert1.Frame.Y+alert1.Frame.H)
	fmt.Printf("alert2(warn): Y=%.1f H=%.1f (bottom=%.1f)\n", alert2.Frame.Y, alert2.Frame.H, alert2.Frame.Y+alert2.Frame.H)
	fmt.Printf("alert3(err):  Y=%.1f H=%.1f (bottom=%.1f)\n", alert3.Frame.Y, alert3.Frame.H, alert3.Frame.Y+alert3.Frame.H)
	fmt.Printf("alert4(succ): Y=%.1f H=%.1f (bottom=%.1f)\n", alert4.Frame.Y, alert4.Frame.H, alert4.Frame.Y+alert4.Frame.H)

	// Also check col and wrapped
	fmt.Printf("\ncol (VStack): Y=%.1f H=%.1f\n", col.Frame.Y, col.Frame.H)
}

func galleryCardBody(th *theme.Theme) *layout.Element {
	_, lh := yoga.Text().MeasureAt("Card body content goes here.", 14)
	el := layout.New(layout.Box().H(lh + 2*th.Spacing.S).PaddingAll(th.Spacing.S))
	return el
}
