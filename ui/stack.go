package ui

import "github.com/mirzakhany/yoga/layout"

// Element is the layout node type, re-exported so call sites can use ui.Element
// without a second import.
type Element = layout.Element

// VStack arranges children in a column. Like SwiftUI's VStack, spacing defaults
// to none — add it fluently with .Gap(n). The returned *Element supports the
// full in-place modifier chain (.Grow, .Padding, .Background, ...).
//
//	ui.VStack(header, body, footer).Gap(8).Grow(1)
func VStack(children ...*Element) *Element {
	return layout.New(layout.Box().Direction(layout.Column), children...)
}

// HStack arranges children in a row, vertically centered (SwiftUI default).
func HStack(children ...*Element) *Element {
	return layout.New(layout.Box().Direction(layout.Row).AlignItems(layout.AlignCenter), children...)
}

// ZStack layers children on top of one another, centered.
func ZStack(children ...*Element) *Element {
	return layout.New(layout.Box().Display(layout.DisplayStack), children...)
}

// Spacer is a flexible gap that expands to push siblings apart in a stack.
func Spacer() *Element { return layout.Spacer() }
