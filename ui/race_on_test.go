//go:build race

package ui

// raceEnabled reports whether the test binary was built with -race, which
// slows code down several times, so timing tests can widen their budgets.
const raceEnabled = true
