package components

import (
	"testing"
)

func TestCheckboxChecked(t *testing.T) {
	c := &Checkbox{Checked: false}
	c.Checked = true
	if !c.Checked {
		t.Fatal("expected checked")
	}
}
