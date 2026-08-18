package main

import (
	"github.com/mirzakhany/yoga/theme"
	"github.com/mirzakhany/yoga/ui"
)

const sidebarWidth float32 = 88

type NavigationBar struct {
	pages       []Page
	currentPage Page
}

func NewNavigationBar() *NavigationBar {
	return &NavigationBar{}
}

func (s *NavigationBar) Layout(c *ui.Ctx) ui.View {
	items := make([]ui.NavItem, 0, len(s.pages))
	selected := 0
	for i, page := range s.pages {
		items = append(items, ui.NavItem{ID: page.Id(), Label: page.Label(), Icon: page.Icon()})
		if s.currentPage != nil && page.Id() == s.currentPage.Id() {
			selected = i
		}
	}
	bg := theme.Current().Background
	return ui.Nav("chapar-nav", ui.NavVertical, ui.NavIconTop, items...).
		Selected(selected).
		OnSelectItem(func(index int, _ string) {
			if index >= 0 && index < len(s.pages) {
				s.setCurrentPage(s.pages[index])
			}
		}).
		NavBackground(&bg).
		Width(sidebarWidth)
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
	if len(pages) > 0 {
		s.currentPage = pages[0]
	}
}
