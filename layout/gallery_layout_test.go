package layout_test

import (
	"fmt"
	"testing"

	"github.com/mirzakhany/yoga"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
	"github.com/mirzakhany/yoga/theme"
	"github.com/mirzakhany/yoga/ui"
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
	ctx := ui.New(text, ui.NewFocusScope(), nil)
	ctx.SetIcons(sheet)
	ctx.BeginFrame(700, 500, nil, nil)

	btnA := ui.Button("a", ui.Text("Primary")).Primary().Layout(ctx)
	btnB := ui.Button("b", ui.Text("Secondary")).Layout(ctx)
	btnC := ui.Button("c", ui.Text("Subtle")).Subtle().Layout(ctx)

	row := layout.New(layout.Box().Direction(layout.Row).Gap(th.Spacing.S).AlignItems(layout.AlignCenter), btnA, btnB, btnC)
	col := layout.New(layout.Box().Direction(layout.Column).Gap(th.Spacing.S), row)
	cardEl := ui.Card("Buttons", "Button variants", ui.Raw(col)).Layout(ctx)

	sections := layout.New(layout.Box().Direction(layout.Column).Gap(th.Spacing.L).PaddingAll(th.Spacing.L), cardEl)
	root := layout.New(layout.Box().Direction(layout.Column).FlexGrow(1), sections)
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

	t.Logf("btnA %v btnB %v btnC %v", btnA.Frame, btnB.Frame, btnC.Frame)
	assertSep("btnA/btnB", btnA, btnB, "x")
	assertSep("btnB/btnC", btnB, btnC, "x")

	if cardEl.Frame.H < 80 {
		t.Fatalf("card too short: %.1f", cardEl.Frame.H)
	}
	if row.Frame.Y >= cardEl.Frame.Y+cardEl.Frame.H {
		t.Fatalf("row outside card: row.Y=%.1f card bottom=%.1f", row.Frame.Y, cardEl.Frame.Y+cardEl.Frame.H)
	}

	labels := []string{"Body label", "Caption", "Muted", "Strong"}
	var labelEls []*layout.Element
	for _, s := range labels {
		labelEls = append(labelEls, ui.Text(s).Layout(ctx))
	}
	labelRow := layout.New(layout.Box().Direction(layout.Row).Gap(th.Spacing.S).AlignItems(layout.AlignCenter), labelEls...)
	layoutRoot(layout.New(layout.Box(), labelRow), 600, 100)
	for i := 1; i < len(labelEls); i++ {
		assertSep(fmt.Sprintf("label%d", i-1), labelEls[i-1], labelEls[i], "x")
	}

	c1 := ui.Checkbox("n1", "Enable notifications").Layout(ctx)
	c2 := ui.Checkbox("n2", "Dark mode sync").Layout(ctx)
	stack := layout.New(layout.Box().Direction(layout.Column).Gap(th.Spacing.S), c1, c2)
	layoutRoot(layout.New(layout.Box(), stack), 400, 120)
	assertSep("check1/check2", c1, c2, "y")

	ra := ui.Radio("ra", "Option A").Check(true).Layout(ctx)
	rb := ui.Radio("rb", "Option B").Layout(ctx)
	rc := ui.Radio("rc", "Option C").Layout(ctx)
	radioRow := layout.New(layout.Box().Direction(layout.Row).Gap(th.Spacing.S).AlignItems(layout.AlignCenter), ra, rb, rc)
	layoutRoot(layout.New(layout.Box(), radioRow), 600, 60)
	assertSep("radioA/radioB", ra, rb, "x")
	assertSep("radioB/radioC", rb, rc, "x")
}
