package theme

import (
	"fmt"
	"testing"

	"github.com/mirzakhany/yoga/highlight"
	"github.com/mirzakhany/yoga/render"
)

// check is one contrast requirement: fg painted on bg must reach want.
type check struct {
	label  string
	fg, bg render.Color
	want   float64
}

func hexOf(c render.Color) string {
	return fmt.Sprintf("#%02X%02X%02X@%.2f",
		int(c.R*255+0.5), int(c.G*255+0.5), int(c.B*255+0.5), c.A)
}

// contrastChecks lists every pairing a reader can actually end up looking at.
// Backgrounds are the surfaces each foreground is painted on in the widget set;
// when a token is painted on several, it is checked against all of them.
func contrastChecks(t Theme) []check {
	var out []check
	add := func(label string, fg, bg render.Color, want float64) {
		out = append(out, check{label, fg, bg, want})
	}

	// Text on every background a row can have.
	textBgs := map[string]render.Color{
		"Surface":           t.Surface,
		"Chrome":            t.Chrome,
		"ChromeMuted":       t.ChromeMuted,
		"ListHover":         t.ListHover,
		"ListActive":        t.ListActive,
		"Selection":         t.Selection,
		"SelectionInactive": t.SelectionInactive,
		"CurrentLine":       Flatten(t.CurrentLine, t.Surface),
		"SearchMatch":       t.SearchMatch,
		"SearchMatchActive": t.SearchMatchActive,
		"BracketMatch":      t.BracketMatch,
		"ErrorSurface":      t.ErrorSurface,
		"WarningSurface":    t.WarningSurface,
		"SuccessSurface":    t.SuccessSurface,
		"InfoSurface":       t.InfoSurface,
	}
	for name, bg := range textBgs {
		add("Foreground on "+name, t.Foreground, bg, ContrastText)
	}

	add("ForegroundMuted on Surface", t.ForegroundMuted, t.Surface, ContrastText)
	add("ForegroundMuted on Chrome", t.ForegroundMuted, t.Chrome, ContrastText)
	add("ForegroundMuted on ChromeMuted", t.ForegroundMuted, t.ChromeMuted, ContrastText)
	add("ForegroundSubtle on Surface", t.ForegroundSubtle, t.Surface, ContrastLargeText)
	add("ForegroundSubtle on Chrome", t.ForegroundSubtle, t.Chrome, ContrastLargeText)
	add("ForegroundDisabled on Surface", t.ForegroundDisabled, t.Surface, ContrastDisabled)

	// Accent fills keep their label readable in every interaction state.
	add("AccentForeground on Accent", t.AccentForeground, t.Accent, ContrastText)
	add("AccentForeground on AccentHover", t.AccentForeground, t.AccentHover, ContrastText)
	add("AccentForeground on AccentPressed", t.AccentForeground, t.AccentPressed, ContrastText)

	// Links are text, so they need text contrast wherever they are placed.
	add("Link on Surface", t.Link, t.Surface, ContrastText)
	add("Link on Chrome", t.Link, t.Chrome, ContrastText)
	add("LinkHover on Surface", t.LinkHover, t.Surface, ContrastText)
	add("LinkHover on Chrome", t.LinkHover, t.Chrome, ContrastText)
	add("LinkVisited on Surface", t.LinkVisited, t.Surface, ContrastText)

	// Status text uses the *Foreground variants, never the saturated base.
	status := []struct {
		name     string
		fg, surf render.Color
	}{
		{"Error", t.ErrorForeground, t.ErrorSurface},
		{"Warning", t.WarningForeground, t.WarningSurface},
		{"Success", t.SuccessForeground, t.SuccessSurface},
		{"Info", t.InfoForeground, t.InfoSurface},
	}
	for _, s := range status {
		add(s.name+"Foreground on Surface", s.fg, t.Surface, ContrastText)
		add(s.name+"Foreground on Chrome", s.fg, t.Chrome, ContrastText)
		add(s.name+"Foreground on "+s.name+"Surface", s.fg, s.surf, ContrastText)
	}

	// Non-text: boundaries, focus, and the scrollbar handle.
	add("BorderControl on Surface", t.BorderControl, t.Surface, ContrastNonText)
	add("BorderControl on Chrome", t.BorderControl, t.Chrome, ContrastNonText)
	add("FocusRing on Surface", t.FocusRing, t.Surface, ContrastNonText)
	add("FocusRing on Chrome", t.FocusRing, t.Chrome, ContrastNonText)
	// The ring is stroked on a control's edge, so whatever FocusRingOn hands
	// back must clear that control's own fill too.
	for name, fill := range map[string]render.Color{
		"Accent": t.Accent, "AccentHover": t.AccentHover, "AccentPressed": t.AccentPressed,
		"Chrome": t.Chrome, "ChromeMuted": t.ChromeMuted, "ListActive": t.ListActive,
	} {
		add("FocusRingOn("+name+") vs "+name, t.FocusRingOn(fill), fill, ContrastNonText)
	}
	add("ScrollThumb on ScrollTrack", t.ScrollThumb, t.ScrollTrack, ContrastNonText)
	add("Caret on Surface", t.Caret, t.Surface, ContrastNonText)

	// Syntax has to clear text contrast on the editor background.
	for _, c := range []highlight.ColorClass{
		highlight.ClassDefault, highlight.ClassKeyword, highlight.ClassString,
		highlight.ClassComment, highlight.ClassNumber, highlight.ClassType,
	} {
		add(fmt.Sprintf("syntax %v on Surface", c), t.SyntaxColor(c), t.Surface, ContrastText)
	}
	return out
}

func TestBuiltinThemesMeetContrastTargets(t *testing.T) {
	for _, name := range Names() {
		th, _ := Get(name)
		t.Run(name, func(t *testing.T) {
			for _, c := range contrastChecks(th) {
				if got := ContrastRatio(c.fg, c.bg); got < c.want {
					t.Errorf("%s: %.2f:1, want %.2f:1 (fg %s, bg %s)",
						c.label, got, c.want, hexOf(c.fg), hexOf(c.bg))
				}
			}
		})
	}
}

// distinction is a minimum ratio between two states that must look different.
// These are perceptual separations, well below text contrast: a hover that is
// only a hair off its rest state reads as no feedback at all.
func TestBuiltinThemesSeparateInteractionStates(t *testing.T) {
	const (
		hoverStep  = 1.15 // rest row -> hovered row
		activeStep = 1.12 // hovered row -> selected row
		accentStep = 1.05 // accent fill -> accent hover
		matchStep  = 1.10 // search hit -> active search hit
	)
	for _, name := range Names() {
		th, _ := Get(name)
		t.Run(name, func(t *testing.T) {
			pairs := []struct {
				label string
				a, b  render.Color
				want  float64
			}{
				{"ListHover vs Chrome", th.ListHover, th.Chrome, hoverStep},
				{"ListActive vs ListHover", th.ListActive, th.ListHover, activeStep},
				{"AccentHover vs Accent", th.AccentHover, th.Accent, accentStep},
				{"AccentPressed vs Accent", th.AccentPressed, th.Accent, accentStep},
			}
			for _, p := range pairs {
				if got := ContrastRatio(p.a, p.b); got < p.want {
					t.Errorf("%s: %.3f:1, want %.3f:1 (%s vs %s)",
						p.label, got, p.want, hexOf(p.a), hexOf(p.b))
				}
			}
		})
	}
}

// Some states are separated by hue rather than by lightness — an amber search
// hit against an orange active hit — because the text painted over both caps
// how dark either may go. Contrast ratio scores those ~1.0:1, so they are
// measured with Distance instead.
func TestBuiltinThemesSeparateSameLightnessStates(t *testing.T) {
	const minDistance = 0.055
	for _, name := range Names() {
		th, _ := Get(name)
		t.Run(name, func(t *testing.T) {
			pairs := []struct {
				label string
				a, b  render.Color
			}{
				{"SearchMatchActive vs SearchMatch", th.SearchMatchActive, th.SearchMatch},
				{"Selection vs SelectionInactive", th.Selection, th.SelectionInactive},
				{"SearchMatch vs Surface", th.SearchMatch, th.Surface},
				{"BracketMatch vs Surface", th.BracketMatch, th.Surface},
			}
			for _, p := range pairs {
				if got := Distance(p.a, p.b); got < minDistance {
					t.Errorf("%s: distance %.3f, want %.3f (%s vs %s)",
						p.label, got, minDistance, hexOf(p.a), hexOf(p.b))
				}
			}
		})
	}
}

// Family siblings must exist, and each must sit on the side it claims.
func TestThemeFamilySiblingsResolve(t *testing.T) {
	for _, name := range Names() {
		th, _ := Get(name)
		light, ok := Get(th.LightSibling)
		if !ok {
			t.Errorf("%s: LightSibling %q not registered", name, th.LightSibling)
		} else if light.Dark && light.Name != th.Name {
			t.Errorf("%s: LightSibling %q is a dark theme", name, light.Name)
		}
		dark, ok := Get(th.DarkSibling)
		if !ok {
			t.Errorf("%s: DarkSibling %q not registered", name, th.DarkSibling)
		} else if !dark.Dark && dark.Name != th.Name {
			t.Errorf("%s: DarkSibling %q is a light theme", name, dark.Name)
		}
	}
}
