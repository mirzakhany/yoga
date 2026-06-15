package components

import (
	"testing"

	"github.com/mirzakhany/yoga/layout"
)

func TestCheckboxChecked(t *testing.T) {
	c := &Checkbox{Checked: false}
	c.Checked = true
	if !c.Checked {
		t.Fatal("expected checked")
	}
}

func TestCheckboxSetLabelStyle(t *testing.T) {
	c := &Checkbox{Label: "task", El: layout.New(layout.Box())}
	c.SetLabelStyle(CheckboxLabelStyle{Muted: true, Strikethrough: true})
	if !c.labelStyle.Muted || !c.labelStyle.Strikethrough {
		t.Fatal("expected muted strikethrough label style")
	}
}
