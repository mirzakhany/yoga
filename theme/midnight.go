package theme

// yogaMidnight is a cool blue-slate dark theme: deep slate surfaces, a vivid
// blue accent, and an emerald success green. Matches the "v1" mockup palette.
//
// Note: the app's root background reads th.Background and editors read
// th.Surface; normalize() keeps Background == Surface, so they render as one
// color. The visual tiering comes from Chrome (toolbar/tab strips) sitting one
// step above Surface, plus the 1px Border around editor panels.
func yogaMidnight() Theme {
	t := Theme{
		Name: "yoga-midnight", Dark: true,

		Surface:            rgb(21, 26, 33),    // #151A21 editor / workspace
		Chrome:             rgb(28, 35, 44),    // #1C232C toolbars, tab bars, controls
		ChromeMuted:        rgb(30, 37, 46),    // #1E252E gutters, tracks
		Foreground:         rgb(232, 236, 241), // #E8ECF1
		ForegroundMuted:    rgb(154, 164, 178), // #9AA4B2
		ForegroundSubtle:   rgb(90, 101, 115),  // #5A6573
		ForegroundDisabled: rgba(154, 164, 178, 0.40),
		Accent:             rgb(47, 111, 237), // #2F6FED
		AccentHover:        rgb(59, 125, 240), // #3B7DF0
		AccentPressed:      rgb(37, 96, 216),  // #2560D8
		AccentForeground:   rgb(255, 255, 255),
		Border:             rgb(42, 50, 61), // #2A323D
		BorderStrong:       rgb(58, 68, 82), // #3A4452
		ListHover:          rgb(34, 41, 51), // #222933
		ListActive:         rgb(42, 50, 61), // #2A323D
		FocusRing:          rgb(47, 111, 237), // #2F6FED

		Selection:        rgb(40, 64, 110), // #28406E
		ScrollTrack:      rgb(26, 32, 39),  // #1A2027
		ScrollThumb:      rgb(58, 68, 82),  // #3A4452
		ScrollThumbHover: rgb(47, 111, 237), // #2F6FED

		Error:   rgb(237, 106, 94), // #ED6A5E
		Warning: rgb(224, 165, 59), // #E0A53B
		Success: rgb(79, 184, 112), // #4FB870

		Spacing:    DefaultSpacing(),
		Radius:     DefaultRadius(),
		Stroke:     DefaultStroke(),
		Typography: DefaultTypography(),
		Metrics:    DefaultComponentMetrics(),
		Elevation:  DefaultElevationDark(),

		// syntax(def, keyword, string, comment, number, type)
		Syntax: syntax(
			rgb(169, 177, 214), // default   #A9B1D6
			rgb(187, 154, 247), // keyword   #BB9AF7  (true/false/null)
			rgb(158, 206, 106), // string    #9ECE6A
			rgb(96, 104, 128),  // comment   #60687F
			rgb(255, 158, 100), // number    #FF9E64
			rgb(122, 162, 247), // type      #7AA2F7  (JSON keys, if classed as type)
		),
	}
	syncLegacyFromYoga(&t)
	return t
}
