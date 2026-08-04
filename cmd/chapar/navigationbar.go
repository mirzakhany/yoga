package main

import (
	"github.com/mirzakhany/yoga/components"
	"github.com/mirzakhany/yoga/theme"
	"github.com/mirzakhany/yoga/ui"
)

const sidebarWidth float32 = 88

type NavigationBar struct {
	nav *components.Navigation

	pages       []Page
	currentPage Page
}

// NewNavigationBar builds the retained navigation once.
func NewNavigationBar() *NavigationBar {
	s := &NavigationBar{}
	s.nav = components.NewNavigation(components.NavVertical, components.NavIconTop)
	s.nav.El.Style = s.nav.El.Style.W(sidebarWidth).FlexShrink(0)
	// Rail recedes: darker than the lighter left pane (theme.Panel).
	s.nav.Background = &theme.Current().Background
	s.nav.OnSelect = func(index int, id string) { s.setCurrentPage(s.pages[index]) }
	return s
}

func (s *NavigationBar) Layout(c *ui.Ctx) *ui.Element { return s.nav.Layout(c) }

func (s *NavigationBar) setCurrentPage(page Page) {
	for _, p := range s.pages {
		if p.Id() == page.Id() {
			s.currentPage = p
			break
		}
	}
}

func (s *NavigationBar) CurrentPage() Page {
	return s.currentPage
}

func (s *NavigationBar) SetPages(pages []Page) {
	s.pages = pages
	s.nav.Clear()
	for _, page := range pages {
		s.nav.Add(components.NavItem{ID: page.Id(), Label: page.Label(), Icon: page.Icon()})
	}
	if len(pages) > 0 {
		s.currentPage = pages[0]
		s.nav.Selected = s.currentPage.Index()
	}
}
