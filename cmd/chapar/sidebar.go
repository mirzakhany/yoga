package main

import (
	"github.com/mirzakhany/yoga/components"
	"github.com/mirzakhany/yoga/ui"
)

const sidebarWidth float32 = 88

type Sidebar struct {
	nav *components.Navigation
}

// NewSidebar builds the retained navigation once.
func NewSidebar() *Sidebar {
	s := &Sidebar{}
	s.nav = components.NewNavigation(components.NavVertical, components.NavIconTop)
	s.nav.El.Style = s.nav.El.Style.W(sidebarWidth).FlexShrink(0)
	s.nav.Add(components.NavItem{ID: "requests", Label: "Requests", Icon: "edit"})
	s.nav.Add(components.NavItem{ID: "environments", Label: "Envs", Icon: "code"})
	s.nav.Selected = 0
	return s
}

func (s *Sidebar) Layout(c *ui.Ctx) *ui.Element { return s.nav.Layout(c) }
