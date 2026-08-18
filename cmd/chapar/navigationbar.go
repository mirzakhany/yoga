package main

import (
	"github.com/mirzakhany/yoga/theme"
	"github.com/mirzakhany/yoga/ui"
)

const sidebarWidth float32 = 88

type NavigationBar struct {
	nav         *ui.Navigation
	pages       []Page
	currentPage Page
}

func NewNavigationBar() *NavigationBar {
	s := &NavigationBar{}
	s.nav = ui.NewNavigation(ui.NavVertical, ui.NavIconTop)
	s.nav.Background = &theme.Current().Background
	s.nav.OnSelect = func(index int, id string) { s.setCurrentPage(s.pages[index]) }
	return s
}

func (s *NavigationBar) Layout(c *ui.Ctx) ui.View {
	return ui.ViewOf(s.nav).Width(sidebarWidth)
}

func (s *NavigationBar) setCurrentPage(page Page) {
	for _, p := range s.pages {
		if p.Id() == page.Id() {
			s.currentPage = p
			break
		}
	}
}

func (s *NavigationBar) CurrentPage() Page { return s.currentPage }

func (s *NavigationBar) SetPages(pages []Page) {
	s.pages = pages
	s.nav.Clear()
	for _, page := range pages {
		s.nav.Add(ui.NavItem{ID: page.Id(), Label: page.Label(), Icon: page.Icon()})
	}
	if len(pages) > 0 {
		s.currentPage = pages[0]
		s.nav.Selected = s.currentPage.Index()
	}
}
