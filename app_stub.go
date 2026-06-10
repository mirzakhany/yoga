//go:build nogpu

package yoga

import (
	"errors"

	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
)

// App is the no-GPU stub. The headless build (-tags nogpu) does not link the
// windowing/WebGPU stack, so New always fails; the type exists only so the
// package compiles and the public API is consistent across build tags.
type App struct{}

// New always returns an error under the nogpu build tag.
func New(Config) (*App, error) {
	return nil, errors.New("yoga: GPU build required (built with -tags nogpu)")
}

// Atlas is a no-op in the headless build.
func (a *App) Atlas() *render.FontAtlas { return nil }

// Text is a no-op in the headless build; use yoga.Text() after SetResources.
func (a *App) Text() *shape.Engine { return nil }

// Clipboard is a no-op in the headless build.
func (a *App) Clipboard() input.Clipboard { return nil }

// SetScene is a no-op in the headless build.
func (a *App) SetScene(Scene) {}

// Run is a no-op in the headless build.
func (a *App) Run() {}

// Close is a no-op in the headless build.
func (a *App) Close() {}
