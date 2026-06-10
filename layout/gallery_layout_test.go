package layout_test

import (
	"fmt"
	"testing"

	"github.com/mirzakhany/yoga"
	"github.com/mirzakhany/yoga/components"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
	"github.com/mirzakhany/yoga/theme"
)

func layoutRoot(root *layout.Element, w, h float32) { root.Calculate(w, h) }

func TestGalleryRowAndCardLayout(t *testing.T) {
	th := theme.Current()
	text, err := shape.NewEngine(1, false)
	if err != nil {
		t.Fatal(err)
	}
	sheet := render.NewSpriteSheet(text.Atlas)
	yoga.SetResources(text, sheet, nil)

	btnA := components.NewButtonVariant("Primary", components.VariantPrimary, nil)
	btnB := components.NewButtonVariant("Secondary", components.VariantSecondary, nil)
	btnC := components.NewButtonVariant("Subtle", components.VariantSubtle, nil)

	row := layout.New(layout.Box().Direction(layout.Row).Gap(th.Spacing.S).AlignItems(layout.AlignCenter), btnA.El, btnB.El, btnC.El)
	col := layout.New(layout.Box().Direction(layout.Column).Gap(th.Spacing.S), row)
	card := components.NewCard("Buttons", "Button variants", col)

	sections := layout.New(layout.Box().Direction(layout.Column).Gap(th.Spacing.L).PaddingAll(th.Spacing.L), card.El)
	svContent := sections
	root := layout.New(layout.Box().Direction(layout.Column).FlexGrow(1), svContent)
	layoutRoot(root, 700, 500)

	assertSep := func(name string, a, b *layout.Element, axis string) {
		t.Helper()
		switch axis {
		case "x":
			end := a.Frame.X + a.Frame.W
			if b.Frame.X < end {
				t.Errorf("%s overlaps %s on x: a=[%.1f..%.1f] b starts %.1f", name, axis, a.Frame.X, end, b.Frame.X)
			}
		case "y":
			end := a.Frame.Y + a.Frame.H
			if b.Frame.Y < end-0.5 {
				t.Errorf("%s overlaps on y: a=[%.1f..%.1f] b starts %.1f", name, a.Frame.Y, end, b.Frame.Y)
			}
		}
	}

	t.Logf("btnA %v btnB %v btnC %v", btnA.El.Frame, btnB.El.Frame, btnC.El.Frame)
	assertSep("btnA/btnB", btnA.El, btnB.El, "x")
	assertSep("btnB/btnC", btnB.El, btnC.El, "x")

	if card.El.Frame.H < 80 {
		t.Fatalf("card too short: %.1f", card.El.Frame.H)
	}
	if row.Frame.Y >= card.El.Frame.Y+card.El.Frame.H {
		t.Fatalf("row outside card: row.Y=%.1f card bottom=%.1f", row.Frame.Y, card.El.Frame.Y+card.El.Frame.H)
	}

	// Labels row
	labels := []string{"Body label", "Caption", "Muted", "Strong"}
	var labelEls []*layout.Element
	for _, s := range labels {
		labelEls = append(labelEls, components.NewLabel(s, components.LabelBody).El)
	}
	labelRow := layout.New(layout.Box().Direction(layout.Row).Gap(th.Spacing.S).AlignItems(layout.AlignCenter), labelEls...)
	layoutRoot(layout.New(layout.Box(), labelRow), 600, 100)
	for i := 1; i < len(labelEls); i++ {
		assertSep(fmt.Sprintf("label%d", i-1), labelEls[i-1], labelEls[i], "x")
	}

	// Checkbox stack
	c1 := components.NewCheckbox("Enable notifications")
	c2 := components.NewCheckbox("Dark mode sync")
	stack := layout.New(layout.Box().Direction(layout.Column).Gap(th.Spacing.S), c1.El, c2.El)
	layoutRoot(layout.New(layout.Box(), stack), 400, 120)
	assertSep("check1/check2", c1.El, c2.El, "y")

	// Radio row
	grp := components.NewRadioGroup()
	ra := grp.Add("Option A")
	rb := grp.Add("Option B")
	rc := grp.Add("Option C")
	radioRow := layout.New(layout.Box().Direction(layout.Row).Gap(th.Spacing.S).AlignItems(layout.AlignCenter), ra.El, rb.El, rc.El)
	layoutRoot(layout.New(layout.Box(), radioRow), 600, 60)
	assertSep("radioA/radioB", ra.El, rb.El, "x")
	assertSep("radioB/radioC", rb.El, rc.El, "x")
}
