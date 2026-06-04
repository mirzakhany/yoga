// Package components is layer 3: stateful widgets built from layout.Element
// trees plus paint and input hooks. Each widget owns an Element (its `El`)
// which the application nests into the overall tree; the widget's Paint hook
// emits geometry and its OnMouse hook updates interaction state.
//
// Theming: widgets hold a *theme.Theme pointing at the single live active theme
// (theme.Current()). Because the active theme is mutated in place on a switch,
// changing the theme at runtime is reflected by every widget on the next paint
// with no rebuild.
package components

// Small float helpers shared by widgets (kept here, not in the theme package,
// since they are layout math rather than palette concerns).

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
