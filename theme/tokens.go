package theme

import "github.com/mirzakhany/yoga/render"

// Spacing is the Yoga spacing ramp in logical pixels.
type Spacing struct {
	XXS    float32 // 2
	XS     float32 // 4
	SNudge float32 // 6
	S      float32 // 8
	MNudge float32 // 10
	M      float32 // 12
	L      float32 // 16
	XL     float32 // 20
	XXL    float32 // 24
	XXXL   float32 // 32
}

// DefaultSpacing returns the Yoga spacing ramp.
func DefaultSpacing() Spacing {
	return Spacing{
		XXS: 2, XS: 4, SNudge: 6, S: 8, MNudge: 10,
		M: 12, L: 16, XL: 20, XXL: 24, XXXL: 32,
	}
}

// Radius is the Yoga corner-radius ramp in logical pixels.
type Radius struct {
	None     float32 // 0
	Small    float32 // 2
	Medium   float32 // 4
	Large    float32 // 6
	XLarge   float32 // 8
	Circular float32 // 9999 (fully rounded caps)
}

// DefaultRadius returns the Yoga radius ramp.
func DefaultRadius() Radius {
	return Radius{
		None: 0, Small: 2, Medium: 4, Large: 6, XLarge: 8, Circular: 9999,
	}
}

// Stroke is the Yoga stroke-width ramp in logical pixels.
type Stroke struct {
	Thin    float32 // 1
	Thick   float32 // 2
	Thicker float32 // 3
}

// DefaultStroke returns the Yoga stroke ramp.
func DefaultStroke() Stroke {
	return Stroke{Thin: 1, Thick: 2, Thicker: 3}
}

// TypographyStyle describes one text style in the Yoga type ramp.
type TypographyStyle struct {
	Size       float32 // font size in logical px
	LineHeight float32 // line box height in logical px
	Weight     int     // 400 regular, 600 semibold
}

// Typography is the Yoga type ramp used by chrome widgets.
type Typography struct {
	Caption    TypographyStyle // 12/16
	Body       TypographyStyle // 14/20
	BodyStrong TypographyStyle // 14/20 semibold
	Subtitle   TypographyStyle // 16/22
	Title      TypographyStyle // 20/28
}

// DefaultTypography returns Yoga defaults (logical px).
func DefaultTypography() Typography {
	return Typography{
		Caption:    TypographyStyle{Size: 12, LineHeight: 16, Weight: 400},
		Body:       TypographyStyle{Size: 14, LineHeight: 20, Weight: 400},
		BodyStrong: TypographyStyle{Size: 14, LineHeight: 20, Weight: 600},
		Subtitle:   TypographyStyle{Size: 16, LineHeight: 22, Weight: 600},
		Title:      TypographyStyle{Size: 20, LineHeight: 28, Weight: 600},
	}
}

// Shadow describes one elevation shadow layer.
type Shadow struct {
	OffsetX float32
	OffsetY float32
	Blur    float32
	Color   render.Color
}

// Elevation holds Yoga shadow presets for overlays and floating surfaces.
type Elevation struct {
	ShadowSm Shadow
	ShadowMd Shadow
	ShadowLg Shadow
	ShadowXl Shadow
}

// DefaultElevationDark returns shadow presets tuned for dark surfaces.
func DefaultElevationDark() Elevation {
	return Elevation{
		ShadowSm: Shadow{Blur: 3, Color: rgba(0, 0, 0, 0.40)},
		ShadowMd: Shadow{OffsetY: 1, Blur: 5, Color: rgba(0, 0, 0, 0.45)},
		ShadowLg: Shadow{OffsetY: 2, Blur: 8, Color: rgba(0, 0, 0, 0.50)},
		ShadowXl: Shadow{OffsetY: 3, Blur: 12, Color: rgba(0, 0, 0, 0.55)},
	}
}

// DefaultElevationLight returns shadow presets tuned for light surfaces.
func DefaultElevationLight() Elevation {
	return Elevation{
		ShadowSm: Shadow{Blur: 3, Color: rgba(0, 0, 0, 0.10)},
		ShadowMd: Shadow{OffsetY: 1, Blur: 5, Color: rgba(0, 0, 0, 0.12)},
		ShadowLg: Shadow{OffsetY: 2, Blur: 8, Color: rgba(0, 0, 0, 0.14)},
		ShadowXl: Shadow{OffsetY: 3, Blur: 12, Color: rgba(0, 0, 0, 0.16)},
	}
}

// ComponentMetrics holds shared control sizing for Yoga widgets.
type ComponentMetrics struct {
	ControlHeight       float32 // tab bar, menu rows
	MenuItemHeight      float32
	ScrollbarSize       float32
	ScrollbarMinThumb   float32
	ScrollbarThumbInset float32
	IconSizeSM          float32
	IconSizeMD          float32
	TreeIndent          float32
	TreeIconSize        float32
	TreeChevronSize     float32
}

// DefaultComponentMetrics returns Yoga control metrics.
func DefaultComponentMetrics() ComponentMetrics {
	return ComponentMetrics{
		ControlHeight:       32,
		MenuItemHeight:      32,
		ScrollbarSize:       12,
		ScrollbarMinThumb:   24,
		ScrollbarThumbInset: 2,
		IconSizeSM:          16,
		IconSizeMD:          18,
		TreeIndent:          14,
		TreeIconSize:        14,
		TreeChevronSize:     14,
	}
}

func rgba(r, g, b uint8, a float32) render.Color {
	c := render.RGBA8(r, g, b, 255)
	c.A = a
	return c
}
