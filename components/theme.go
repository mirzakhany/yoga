// Package components is layer 3: stateful widgets built from layout.Element
// trees plus paint and input hooks. Each widget owns an Element (its `El`)
// which the application nests into the overall tree; the widget's Paint hook
// emits geometry and its OnMouse hook updates interaction state.
package components

import (
	"github.com/mirzakhany/yoga/highlight"
	"github.com/mirzakhany/yoga/render"
)

// Theme is the color palette shared by all widgets.
type Theme struct {
	Background render.Color // window / workspace background
	Panel      render.Color // sidebars, menus
	PanelAlt   render.Color // slightly different panel shade (tracks, gutters)
	Text       render.Color
	TextDim    render.Color
	Accent     render.Color
	Hover      render.Color
	Active     render.Color
	Border     render.Color
	Selection  render.Color

	Syntax map[highlight.ColorClass]render.Color
}

// DarkTheme is a calm dark coding palette.
func DarkTheme() Theme {
	return Theme{
		Background: render.RGBA8(24, 24, 29, 255),
		Panel:      render.RGBA8(33, 33, 40, 255),
		PanelAlt:   render.RGBA8(43, 43, 52, 255),
		Text:       render.RGBA8(213, 217, 227, 255),
		TextDim:    render.RGBA8(126, 131, 145, 255),
		Accent:     render.RGBA8(94, 129, 232, 255),
		Hover:      render.RGBA8(54, 56, 68, 255),
		Active:     render.RGBA8(70, 74, 92, 255),
		Border:     render.RGBA8(54, 56, 68, 255),
		Selection:  render.RGBA8(58, 70, 104, 255),
		Syntax: map[highlight.ColorClass]render.Color{
			highlight.ClassDefault: render.RGBA8(213, 217, 227, 255),
			highlight.ClassKeyword: render.RGBA8(197, 134, 192, 255),
			highlight.ClassString:  render.RGBA8(152, 195, 121, 255),
			highlight.ClassComment: render.RGBA8(106, 115, 125, 255),
			highlight.ClassNumber:  render.RGBA8(209, 154, 102, 255),
			highlight.ClassType:    render.RGBA8(86, 182, 194, 255),
		},
	}
}

// SyntaxColor resolves a token class to a color, defaulting to plain text.
func (t Theme) SyntaxColor(c highlight.ColorClass) render.Color {
	if col, ok := t.Syntax[c]; ok {
		return col
	}
	return t.Text
}

func f32max(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}

func f32min(a, b float32) float32 {
	if a < b {
		return a
	}
	return b
}

func clampf(v, lo, hi float32) float32 {
	return f32max(lo, f32min(v, hi))
}
