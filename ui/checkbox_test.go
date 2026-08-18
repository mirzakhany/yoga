package ui

import "testing"

func TestCheckboxNode(t *testing.T) {
	n := Checkbox("c1", "task").Check(true)
	if !n.checked {
		t.Fatal("expected checked")
	}
	n.LabelMuted(true).LabelStrike(true)
	if !n.labelMuted || !n.labelStrike {
		t.Fatal("expected muted strikethrough label style")
	}
}
