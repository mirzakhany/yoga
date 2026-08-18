package ui

import (
	"testing"
)

func TestRadioLayout(t *testing.T) {
	c := New(nil, NewFocusScope(), nil)
	c.BeginFrame(200, 100, nil, nil)
	el := Radio("r", "Option A").Check(true).Layout(c)
	if el == nil {
		t.Fatal("nil radio")
	}
}
