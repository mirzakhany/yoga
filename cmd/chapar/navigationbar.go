package main

import (
	"github.com/mirzakhany/yoga/components"
	"github.com/mirzakhany/yoga/ui"
)

const sidebarWidth float32 = 88

type NavigationBar struct {
	nav *components.Navigation

	currentPage   int
	currentPageId string
}

// NewSidebar builds the retained navigation once.
func NewNavigationBar() *NavigationBar {
	s := &NavigationBar{}
	s.nav = components.NewNavigation(components.NavVertical, components.NavIconTop)
	s.nav.El.Style = s.nav.El.Style.W(sidebarWidth).FlexShrink(0)
	s.nav.Add(components.NavItem{ID: "requests", Label: "Requests", Icon: "edit"})
	s.nav.Add(components.NavItem{ID: "environments", Label: "Envs", Icon: "code"})
	s.nav.Add(components.NavItem{ID: "workspaces", Label: "Workspaces", Icon: "folder_open"})
	s.nav.Add(components.NavItem{ID: "settings", Label: "Settings", Icon: "settings"})
	s.nav.Selected = 0
	s.nav.OnSelect = func(index int, id string) { s.setCurrentPage(index, id) }
	return s
}

func (s *NavigationBar) Layout(c *ui.Ctx) *ui.Element { return s.nav.Layout(c) }

func (s *NavigationBar) setCurrentPage(page int, id string) {
	s.currentPage = page
	s.currentPageId = id
}

func (s *NavigationBar) CurrentPage() int {
	return s.currentPage
}

func (s *NavigationBar) CurrentPageId() string {
	return s.currentPageId
}
