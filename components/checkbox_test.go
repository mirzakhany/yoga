package components

import (
	"testing"

	"github.com/mirzakhany/yoga/theme"
)

func TestCheckboxChecked(t *testing.T) {
	c := &Checkbox{theme: theme.Current(), Checked: false}
	c.Checked = true
	if !c.Checked {
		t.Fatal("expected checked")
	}
}
