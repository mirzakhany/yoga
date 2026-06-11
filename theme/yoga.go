package theme

// yogaDark is the default Yoga theme — neutral IDE chrome with a soft blue accent.
func yogaDark() Theme {
	t := Theme{
		Name: "yoga-dark", Dark: true,

		Surface:            rgb(24, 24, 29),
		Chrome:             rgb(33, 33, 40),
		ChromeMuted:        rgb(43, 43, 52),
		Foreground:         rgb(213, 217, 227),
		ForegroundMuted:    rgb(126, 131, 145),
		ForegroundSubtle:   rgb(98, 104, 118),
		ForegroundDisabled: rgba(126, 131, 145, 0.45),
		Accent:             rgb(94, 129, 232),
		AccentHover:        rgb(110, 145, 245),
		AccentPressed:      rgb(70, 100, 200),
		AccentForeground:   rgb(255, 255, 255),
		Border:             rgb(54, 56, 68),
		BorderStrong:       rgb(70, 74, 88),
		ListHover:          rgb(54, 56, 68),
		ListActive:         rgb(70, 74, 92),
		FocusRing:          rgb(94, 129, 232),

		Selection:        rgb(58, 70, 104),
		ScrollTrack:      rgb(36, 36, 44),
		ScrollThumb:      rgb(100, 104, 118),
		ScrollThumbHover: rgb(94, 129, 232),

		Error:   rgb(224, 108, 117),
		Warning: rgb(229, 192, 123),
		Success: rgb(152, 195, 121),

		Spacing:    DefaultSpacing(),
		Radius:     DefaultRadius(),
		Stroke:     DefaultStroke(),
		Typography: DefaultTypography(),
		Metrics:    DefaultComponentMetrics(),
		Elevation:  DefaultElevationDark(),

		Syntax: syntax(
			rgb(213, 217, 227), rgb(197, 134, 192), rgb(152, 195, 121),
			rgb(130, 140, 150), rgb(209, 154, 102), rgb(86, 182, 194)), // comment: was 3.67:1, now ~4.8:1
	}
	syncLegacyFromYoga(&t)
	return t
}

// yogaLight is a clean light Yoga theme for daytime work.
func yogaLight() Theme {
	t := Theme{
		Name: "yoga-light", Dark: false,

		Surface:            rgb(250, 250, 252),
		Chrome:             rgb(240, 240, 244),
		ChromeMuted:        rgb(230, 230, 236),
		Foreground:         rgb(30, 32, 38),
		ForegroundMuted:    rgb(110, 116, 128),
		ForegroundSubtle:   rgb(140, 146, 158),
		ForegroundDisabled: rgba(110, 116, 128, 0.45),
		Accent:             rgb(40, 110, 230),
		AccentHover:        rgb(55, 125, 245),
		AccentPressed:      rgb(30, 90, 200),
		AccentForeground:   rgb(255, 255, 255),
		Border:             rgb(215, 218, 225),
		BorderStrong:       rgb(190, 195, 205),
		ListHover:          rgb(225, 228, 235),
		ListActive:         rgb(210, 215, 225),
		FocusRing:          rgb(40, 110, 230),

		Selection:        rgb(180, 205, 250),
		ScrollTrack:      rgb(220, 222, 228),
		ScrollThumb:      rgb(160, 165, 175),
		ScrollThumbHover: rgb(40, 110, 230),

		Error:   rgb(200, 50, 60),
		Warning: rgb(190, 130, 20),
		Success: rgb(40, 140, 70),

		Spacing:    DefaultSpacing(),
		Radius:     DefaultRadius(),
		Stroke:     DefaultStroke(),
		Typography: DefaultTypography(),
		Metrics:    DefaultComponentMetrics(),
		Elevation:  DefaultElevationLight(),

		Syntax: syntax(
			rgb(30, 32, 38), rgb(168, 40, 150), rgb(30, 130, 60),
			rgb(100, 106, 118), rgb(180, 90, 20), rgb(20, 120, 150)), // comment: was 3.03:1, now ~5.1:1
	}
	syncLegacyFromYoga(&t)
	return t
}

// yogaHighContrast is an accessibility-oriented Yoga theme with strong contrast.
func yogaHighContrast() Theme {
	t := Theme{
		Name: "yoga-high-contrast", Dark: true,

		Surface:            rgb(0, 0, 0),
		Chrome:             rgb(16, 16, 16),
		ChromeMuted:        rgb(32, 32, 32),
		Foreground:         rgb(255, 255, 255),
		ForegroundMuted:    rgb(200, 200, 200),
		ForegroundSubtle:   rgb(160, 160, 160),
		ForegroundDisabled: rgba(160, 160, 160, 0.5),
		Accent:             rgb(255, 213, 0),
		AccentHover:        rgb(255, 230, 80),
		AccentPressed:      rgb(220, 180, 0),
		AccentForeground:   rgb(0, 0, 0),
		Border:             rgb(255, 255, 255),
		BorderStrong:       rgb(255, 255, 255),
		ListHover:          rgb(48, 48, 48),
		ListActive:         rgb(64, 64, 64),
		FocusRing:          rgb(255, 213, 0),

		Selection:        rgb(80, 80, 0),
		ScrollTrack:      rgb(24, 24, 24),
		ScrollThumb:      rgb(200, 200, 200),
		ScrollThumbHover: rgb(255, 213, 0),

		Error:   rgb(255, 100, 100),
		Warning: rgb(255, 213, 0),
		Success: rgb(100, 255, 100),

		Spacing:    DefaultSpacing(),
		Radius:     DefaultRadius(),
		Stroke:     DefaultStroke(),
		Typography: DefaultTypography(),
		Metrics:    DefaultComponentMetrics(),
		Elevation:  DefaultElevationDark(),

		Syntax: syntax(
			rgb(255, 255, 255), rgb(255, 180, 180), rgb(180, 255, 180),
			rgb(180, 180, 180), rgb(255, 220, 140), rgb(140, 220, 255)),
	}
	syncLegacyFromYoga(&t)
	return t
}
