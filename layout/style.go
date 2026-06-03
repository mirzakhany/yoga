package layout

import (
	"math"

	"github.com/kjk/flex"
)

// The layout package re-exports the flex enums under friendlier names so that
// callers building UIs never import the underlying engine directly. Swapping
// the engine (see Engine in layout.go) therefore does not ripple out to user
// code.
type (
	FlexDirection = flex.FlexDirection
	Justify       = flex.Justify
	Align         = flex.Align
	Position      = flex.PositionType
	Wrap          = flex.Wrap
)

const (
	Row           = flex.FlexDirectionRow
	RowReverse    = flex.FlexDirectionRowReverse
	Column        = flex.FlexDirectionColumn
	ColumnReverse = flex.FlexDirectionColumnReverse

	JustifyStart   = flex.JustifyFlexStart
	JustifyCenter  = flex.JustifyCenter
	JustifyEnd     = flex.JustifyFlexEnd
	JustifyBetween = flex.JustifySpaceBetween
	JustifyAround  = flex.JustifySpaceAround

	AlignAuto    = flex.AlignAuto
	AlignStart   = flex.AlignFlexStart
	AlignCenter  = flex.AlignCenter
	AlignEnd     = flex.AlignFlexEnd
	AlignStretch = flex.AlignStretch

	PositionRelative = flex.PositionTypeRelative
	PositionAbsolute = flex.PositionTypeAbsolute

	NoWrap = flex.WrapNoWrap
	DoWrap = flex.WrapWrap
)

// nan marks a style dimension as "unset / auto". We use NaN because 0 is a
// meaningful explicit size; the flex setters interpret NaN as auto/undefined.
var nan = float32(math.NaN())

func isUnset(v float32) bool { return math.IsNaN(float64(v)) }

// Edges holds a value per side (used for margin and padding).
type Edges struct {
	Top, Right, Bottom, Left float32
}

// Style is the declarative styling for an Element. Construct it with Box() and
// refine it with the fluent setters below, e.g.
//
//	layout.Box().Direction(layout.Row).Grow(1).Padding(8)
//
// All size fields default to "auto" (NaN); margin/padding default to 0.
type Style struct {
	Dir     FlexDirection
	Justify Justify
	Align   Align
	Wrap    Wrap
	Pos     Position

	Grow   float32
	Shrink float32
	Basis  float32

	Width, Height                            float32
	MinWidth, MinHeight, MaxWidth, MaxHeight float32

	Margin  Edges
	Padding Edges

	// Absolute-position offsets, used when Pos == PositionAbsolute. Unset = NaN.
	Left, Top, Right, Bottom float32
}

// Box returns a Style with sensible flexbox defaults: a column container that
// stretches its children, shrink factor 1, and all sizes auto.
func Box() Style {
	return Style{
		Dir:     Column,
		Justify: JustifyStart,
		Align:   AlignStretch,
		Wrap:    NoWrap,
		Pos:     PositionRelative,
		Grow:    0,
		Shrink:  1,
		Basis:   nan,

		Width: nan, Height: nan,
		MinWidth: nan, MinHeight: nan, MaxWidth: nan, MaxHeight: nan,
		Left: nan, Top: nan, Right: nan, Bottom: nan,
	}
}

func (s Style) Direction(d FlexDirection) Style { s.Dir = d; return s }
func (s Style) JustifyContent(j Justify) Style  { s.Justify = j; return s }
func (s Style) AlignItems(a Align) Style        { s.Align = a; return s }
func (s Style) Grow_(v float32) Style           { s.Grow = v; return s }
func (s Style) Shrink_(v float32) Style         { s.Shrink = v; return s }
func (s Style) W(v float32) Style               { s.Width = v; return s }
func (s Style) H(v float32) Style               { s.Height = v; return s }
func (s Style) Size(w, h float32) Style         { s.Width, s.Height = w, h; return s }
func (s Style) Min(w, h float32) Style          { s.MinWidth, s.MinHeight = w, h; return s }
func (s Style) Max(w, h float32) Style          { s.MaxWidth, s.MaxHeight = w, h; return s }

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

// apply writes the style onto a flex node. NaN values become auto/undefined,
// which is exactly the flex setters' contract, so unset fields need no guards.
func (s Style) apply(n *flex.Node) {
	n.StyleSetFlexDirection(s.Dir)
	n.StyleSetJustifyContent(s.Justify)
	n.StyleSetAlignItems(s.Align)
	n.StyleSetFlexWrap(s.Wrap)
	n.StyleSetPositionType(s.Pos)

	n.StyleSetFlexGrow(s.Grow)
	n.StyleSetFlexShrink(s.Shrink)
	n.StyleSetFlexBasis(s.Basis)

	n.StyleSetWidth(s.Width)
	n.StyleSetHeight(s.Height)
	n.StyleSetMinWidth(s.MinWidth)
	n.StyleSetMinHeight(s.MinHeight)
	n.StyleSetMaxWidth(s.MaxWidth)
	n.StyleSetMaxHeight(s.MaxHeight)

	n.StyleSetMargin(flex.EdgeTop, s.Margin.Top)
	n.StyleSetMargin(flex.EdgeRight, s.Margin.Right)
	n.StyleSetMargin(flex.EdgeBottom, s.Margin.Bottom)
	n.StyleSetMargin(flex.EdgeLeft, s.Margin.Left)

	n.StyleSetPadding(flex.EdgeTop, s.Padding.Top)
	n.StyleSetPadding(flex.EdgeRight, s.Padding.Right)
	n.StyleSetPadding(flex.EdgeBottom, s.Padding.Bottom)
	n.StyleSetPadding(flex.EdgeLeft, s.Padding.Left)

	if !isUnset(s.Left) {
		n.StyleSetPosition(flex.EdgeLeft, s.Left)
	}
	if !isUnset(s.Top) {
		n.StyleSetPosition(flex.EdgeTop, s.Top)
	}
	if !isUnset(s.Right) {
		n.StyleSetPosition(flex.EdgeRight, s.Right)
	}
	if !isUnset(s.Bottom) {
		n.StyleSetPosition(flex.EdgeBottom, s.Bottom)
	}
}
