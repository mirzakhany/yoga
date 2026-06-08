package components

import (
	"testing"

	"github.com/mirzakhany/yoga/theme"
)

func TestRadioGroupInitial(t *testing.T) {
	g := NewRadioGroup(theme.Current())
	if g.Value != -1 {
		t.Fatalf("initial value = %d, want -1", g.Value)
	}
}
