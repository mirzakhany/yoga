package layout

import "math"

// Layout enums are defined locally so callers never depend on an external
// solver. The custom engine in engine.go reads Style directly.

type FlexDirection int

const (
	Row FlexDirection = iota
	RowReverse
	Column
	ColumnReverse
)

type Justify int

const (
	JustifyStart Justify = iota
	JustifyCenter
	JustifyEnd
	JustifyBetween
	JustifyAround
	JustifyEvenly
)

type Align int

const (
	AlignAuto Align = iota
	AlignStart
	AlignCenter
	AlignEnd
	AlignStretch
)

type Wrap int

const (
	NoWrap Wrap = iota
	DoWrap
)

type Position int

const (
	PositionRelative Position = iota
	PositionAbsolute
)

type Display int

const (
	DisplayFlex Display = iota
	DisplayGrid
	DisplayStack
)

type TrackKind int

const (
	TrackFixed TrackKind = iota
	TrackFr
	TrackAuto
)

// Track describes a grid row or column track.
type Track struct {
	Kind  TrackKind
	Value float32
}

func Px(v float32) Track  { return Track{Kind: TrackFixed, Value: v} }
func Fr(v float32) Track  { return Track{Kind: TrackFr, Value: v} }
func Auto() Track         { return Track{Kind: TrackAuto} }

// nan marks a style dimension as "unset / auto". We use NaN because 0 is a
// meaningful explicit size.
var nan = float32(math.NaN())

func isUnset(v float32) bool { return math.IsNaN(float64(v)) }

// Edges holds a value per side (used for margin and padding).
type Edges struct {
	Top, Right, Bottom, Left float32
}

// Style is the declarative styling for an Element. Construct it with Box() and
// refine it with the fluent setters below, e.g.
//
//	layout.Box().Direction(layout.Row).FlexGrow(1).PaddingAll(8)
//
// All size fields default to "auto" (NaN); margin/padding default to 0.
type Style struct {
	Disp Display

	Dir     FlexDirection
	Justify Justify
	Align   Align
	SelfAlign Align
	Wrap    Wrap
	Pos     Position

	Grow   float32
	Shrink float32
	Basis  float32

	Width, Height                            float32
	MinWidth, MinHeight, MaxWidth, MaxHeight float32

	Margin  Edges
	Padding Edges

	RowGap, ColGap float32

	// Grid container tracks and auto-row template.
	Cols     []Track
	Rows     []Track
	AutoRows Track

	// Grid item placement (1-based; 0 = auto).
	ColStart, ColSpan int
	RowStart, RowSpan int

	// Stack container alignment (defaults: center/center in Box()).
	StackJustify Justify
	StackAlign   Align

	// Absolute-position offsets, used when Pos == PositionAbsolute. Unset = NaN.
	Left, Top, Right, Bottom float32
}

// Box returns a Style with sensible flexbox defaults: a column container that
// stretches its children, shrink factor 1, and all sizes auto.
func Box() Style {
	return Style{
		Disp:    DisplayFlex,
		Dir:     Column,
		Justify: JustifyStart,
		Align:   AlignStretch,
		SelfAlign: AlignAuto,
		Wrap:    NoWrap,
		Pos:     PositionRelative,
		Grow:    0,
		Shrink:  1,
		Basis:   nan,

		Width: nan, Height: nan,
		MinWidth: nan, MinHeight: nan, MaxWidth: nan, MaxHeight: nan,
		Left: nan, Top: nan, Right: nan, Bottom: nan,

		StackJustify: JustifyCenter,
		StackAlign:   AlignCenter,
		AutoRows:     Fr(1),
	}
}

func (s Style) Display(d Display) Style         { s.Disp = d; return s }
func (s Style) Direction(d FlexDirection) Style { s.Dir = d; return s }
func (s Style) JustifyContent(j Justify) Style  { s.Justify = j; return s }
func (s Style) AlignItems(a Align) Style { s.Align = a; return s }
func (s Style) AlignSelf(a Align) Style  { s.SelfAlign = a; return s }
func (s Style) FlexGrow(v float32) Style        { s.Grow = v; return s }
func (s Style) FlexShrink(v float32) Style      { s.Shrink = v; return s }
func (s Style) W(v float32) Style               { s.Width = v; return s }
func (s Style) H(v float32) Style               { s.Height = v; return s }
func (s Style) Size(w, h float32) Style         { s.Width, s.Height = w, h; return s }
func (s Style) Min(w, h float32) Style          { s.MinWidth, s.MinHeight = w, h; return s }
func (s Style) Max(w, h float32) Style          { s.MaxWidth, s.MaxHeight = w, h; return s }

func (s Style) Gap(v float32) Style             { s.RowGap, s.ColGap = v, v; return s }
func (s Style) GapXY(col, row float32) Style    { s.ColGap, s.RowGap = col, row; return s }

func (s Style) GridCols(tracks ...Track) Style  { s.Cols = append([]Track(nil), tracks...); return s }
func (s Style) GridRows(tracks ...Track) Style  { s.Rows = append([]Track(nil), tracks...); return s }
func (s Style) GridArea(colStart, colSpan, rowStart, rowSpan int) Style {
	s.ColStart, s.ColSpan = colStart, colSpan
	s.RowStart, s.RowSpan = rowStart, rowSpan
	return s
}
func (s Style) Col(start, span int) Style { s.ColStart, s.ColSpan = start, span; return s }
func (s Style) Row(start, span int) Style { s.RowStart, s.RowSpan = start, span; return s }

func (s Style) StackPosition(jx Justify, ay Align) Style {
	s.StackJustify, s.StackAlign = jx, ay
	return s
}

// PaddingAll sets a uniform padding on all edges.
func (s Style) PaddingAll(v float32) Style {
	s.Padding = Edges{v, v, v, v}
	return s
}

// PaddingXY sets horizontal and vertical padding.
func (s Style) PaddingXY(x, y float32) Style {
	s.Padding = Edges{Top: y, Right: x, Bottom: y, Left: x}
	return s
}

// MarginAll sets a uniform margin on all edges.
func (s Style) MarginAll(v float32) Style {
	s.Margin = Edges{v, v, v, v}
	return s
}

// Absolute positions the element out of normal flow at (left, top).
func (s Style) Absolute(left, top float32) Style {
	s.Pos = PositionAbsolute
	s.Left, s.Top = left, top
	return s
}

// AbsLeft/AbsTop/AbsRight/AbsBottom pin individual edges of an absolutely
// positioned element. Pinning opposite edges (e.g. top and bottom) stretches
// the element between them.
func (s Style) AbsLeft(v float32) Style   { s.Pos = PositionAbsolute; s.Left = v; return s }
func (s Style) AbsTop(v float32) Style    { s.Pos = PositionAbsolute; s.Top = v; return s }
func (s Style) AbsRight(v float32) Style  { s.Pos = PositionAbsolute; s.Right = v; return s }
func (s Style) AbsBottom(v float32) Style { s.Pos = PositionAbsolute; s.Bottom = v; return s }
