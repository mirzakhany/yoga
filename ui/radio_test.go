package ui

import (
	"testing"
)

func TestRadioGroupInitial(t *testing.T) {
	g := NewRadioGroup()
	if g.Value != -1 {
		t.Fatalf("initial value = %d, want -1", g.Value)
	}
}
