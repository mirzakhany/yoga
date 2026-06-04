package theme

import (
	"github.com/mirzakhany/yoga/highlight"
	"github.com/mirzakhany/yoga/render"
)

// rgb is a small alias for opaque colors (alpha 255).
func rgb(r, g, b uint8) render.Color { return render.RGBA8(r, g, b, 255) }

// syntax builds a syntax color map from per-class colors.
func syntax(def, keyword, str, comment, number, typ render.Color) map[highlight.ColorClass]render.Color {
	return map[highlight.ColorClass]render.Color{
		highlight.ClassDefault: def,
		highlight.ClassKeyword: keyword,
		highlight.ClassString:  str,
		highlight.ClassComment: comment,
		highlight.ClassNumber:  number,
		highlight.ClassType:    typ,
	}
}

// builtins returns all themes shipped with the framework.
func builtins() []Theme {
	return []Theme{
		dark(),
		light(),
		githubDark(),
		githubLight(),
		catppuccin(),
		dracula(),
		nord(),
		solarizedDark(),
	}
}

func dark() Theme {
	return Theme{
		Name: "dark", Dark: true,
		Background: rgb(24, 24, 29),
		Panel:      rgb(33, 33, 40),
		PanelAlt:   rgb(43, 43, 52),
		Text:       rgb(213, 217, 227),
		TextDim:    rgb(126, 131, 145),
		Accent:     rgb(94, 129, 232),
		AccentText: rgb(255, 255, 255),
		Hover:      rgb(54, 56, 68),
		Active:     rgb(70, 74, 92),
		Border:     rgb(54, 56, 68),
		Selection:  rgb(58, 70, 104),
		Error:      rgb(224, 108, 117),
		Warning:    rgb(229, 192, 123),
		Success:    rgb(152, 195, 121),
		Syntax: syntax(
			rgb(213, 217, 227), rgb(197, 134, 192), rgb(152, 195, 121),
			rgb(106, 115, 125), rgb(209, 154, 102), rgb(86, 182, 194)),
	}
}

func light() Theme {
	return Theme{
		Name: "light", Dark: false,
		Background: rgb(250, 250, 250),
		Panel:      rgb(240, 240, 242),
		PanelAlt:   rgb(230, 230, 233),
		Text:       rgb(30, 32, 38),
		TextDim:    rgb(110, 116, 128),
		Accent:     rgb(40, 110, 230),
		AccentText: rgb(255, 255, 255),
		Hover:      rgb(225, 228, 235),
		Active:     rgb(210, 215, 225),
		Border:     rgb(215, 218, 225),
		Selection:  rgb(180, 205, 250),
		Error:      rgb(200, 50, 60),
		Warning:    rgb(190, 130, 20),
		Success:    rgb(40, 140, 70),
		Syntax: syntax(
			rgb(30, 32, 38), rgb(168, 40, 150), rgb(30, 130, 60),
			rgb(140, 145, 155), rgb(180, 90, 20), rgb(20, 120, 150)),
	}
}

func githubDark() Theme {
	return Theme{
		Name: "github-dark", Dark: true,
		Background: rgb(13, 17, 23),
		Panel:      rgb(22, 27, 34),
		PanelAlt:   rgb(33, 38, 45),
		Text:       rgb(230, 237, 243),
		TextDim:    rgb(125, 133, 144),
		Accent:     rgb(47, 129, 247),
		AccentText: rgb(255, 255, 255),
		Hover:      rgb(33, 38, 45),
		Active:     rgb(48, 54, 61),
		Border:     rgb(48, 54, 61),
		Selection:  rgb(56, 90, 140),
		Error:      rgb(248, 81, 73),
		Warning:    rgb(210, 153, 34),
		Success:    rgb(63, 185, 80),
		Syntax: syntax(
			rgb(230, 237, 243), rgb(255, 123, 114), rgb(165, 214, 255),
			rgb(139, 148, 158), rgb(121, 192, 255), rgb(255, 166, 87)),
	}
}

func githubLight() Theme {
	return Theme{
		Name: "github-light", Dark: false,
		Background: rgb(255, 255, 255),
		Panel:      rgb(246, 248, 250),
		PanelAlt:   rgb(234, 238, 242),
		Text:       rgb(31, 35, 40),
		TextDim:    rgb(101, 109, 118),
		Accent:     rgb(9, 105, 218),
		AccentText: rgb(255, 255, 255),
		Hover:      rgb(234, 238, 242),
		Active:     rgb(215, 222, 228),
		Border:     rgb(208, 215, 222),
		Selection:  rgb(184, 215, 255),
		Error:      rgb(207, 34, 46),
		Warning:    rgb(154, 103, 0),
		Success:    rgb(26, 127, 55),
		Syntax: syntax(
			rgb(31, 35, 40), rgb(207, 34, 46), rgb(10, 48, 105),
			rgb(110, 119, 129), rgb(5, 80, 174), rgb(149, 56, 0)),
	}
}

func catppuccin() Theme {
	return Theme{
		Name: "catppuccin", Dark: true,
		Background: rgb(30, 30, 46),
		Panel:      rgb(24, 24, 37),
		PanelAlt:   rgb(49, 50, 68),
		Text:       rgb(205, 214, 244),
		TextDim:    rgb(166, 173, 200),
		Accent:     rgb(137, 180, 250),
		AccentText: rgb(30, 30, 46),
		Hover:      rgb(49, 50, 68),
		Active:     rgb(69, 71, 90),
		Border:     rgb(69, 71, 90),
		Selection:  rgb(88, 91, 112),
		Error:      rgb(243, 139, 168),
		Warning:    rgb(249, 226, 175),
		Success:    rgb(166, 227, 161),
		Syntax: syntax(
			rgb(205, 214, 244), rgb(203, 166, 247), rgb(166, 227, 161),
			rgb(108, 112, 134), rgb(250, 179, 135), rgb(249, 226, 175)),
	}
}

func dracula() Theme {
	return Theme{
		Name: "dracula", Dark: true,
		Background: rgb(40, 42, 54),
		Panel:      rgb(33, 34, 44),
		PanelAlt:   rgb(68, 71, 90),
		Text:       rgb(248, 248, 242),
		TextDim:    rgb(98, 114, 164),
		Accent:     rgb(189, 147, 249),
		AccentText: rgb(40, 42, 54),
		Hover:      rgb(68, 71, 90),
		Active:     rgb(88, 91, 112),
		Border:     rgb(68, 71, 90),
		Selection:  rgb(68, 71, 90),
		Error:      rgb(255, 85, 85),
		Warning:    rgb(241, 250, 140),
		Success:    rgb(80, 250, 123),
		Syntax: syntax(
			rgb(248, 248, 242), rgb(255, 121, 198), rgb(241, 250, 140),
			rgb(98, 114, 164), rgb(189, 147, 249), rgb(139, 233, 253)),
	}
}

func nord() Theme {
	return Theme{
		Name: "nord", Dark: true,
		Background: rgb(46, 52, 64),
		Panel:      rgb(59, 66, 82),
		PanelAlt:   rgb(67, 76, 94),
		Text:       rgb(216, 222, 233),
		TextDim:    rgb(123, 136, 161),
		Accent:     rgb(136, 192, 208),
		AccentText: rgb(46, 52, 64),
		Hover:      rgb(67, 76, 94),
		Active:     rgb(76, 86, 106),
		Border:     rgb(76, 86, 106),
		Selection:  rgb(76, 86, 106),
		Error:      rgb(191, 97, 106),
		Warning:    rgb(235, 203, 139),
		Success:    rgb(163, 190, 140),
		Syntax: syntax(
			rgb(216, 222, 233), rgb(129, 161, 193), rgb(163, 190, 140),
			rgb(97, 110, 136), rgb(180, 142, 173), rgb(143, 188, 187)),
	}
}

func solarizedDark() Theme {
	return Theme{
		Name: "solarized-dark", Dark: true,
		Background: rgb(0, 43, 54),
		Panel:      rgb(7, 54, 66),
		PanelAlt:   rgb(7, 54, 66),
		Text:       rgb(131, 148, 150),
		TextDim:    rgb(88, 110, 117),
		Accent:     rgb(38, 139, 210),
		AccentText: rgb(0, 43, 54),
		Hover:      rgb(7, 54, 66),
		Active:     rgb(88, 110, 117),
		Border:     rgb(88, 110, 117),
		Selection:  rgb(7, 54, 66),
		Error:      rgb(220, 50, 47),
		Warning:    rgb(181, 137, 0),
		Success:    rgb(133, 153, 0),
		Syntax: syntax(
			rgb(131, 148, 150), rgb(133, 153, 0), rgb(42, 161, 152),
			rgb(88, 110, 117), rgb(211, 54, 130), rgb(181, 137, 0)),
	}
}
