package main

import (
	"github.com/mirzakhany/yoga/icons"
	"github.com/mirzakhany/yoga/ui"
)

type Page interface {
	Layout(c *ui.Ctx) ui.View
	Index() int
	Id() string
	Label() string
	Icon() icons.Icon
}

type page struct {
	index  int
	id     string
	label  string
	icon   icons.Icon
	layout func(c *ui.Ctx) ui.View
}

func NewPage(index int, id string, label string, icon icons.Icon, layout func(c *ui.Ctx) ui.View) Page {
	return &page{index: index, id: id, label: label, icon: icon, layout: layout}
}

func (p *page) Layout(c *ui.Ctx) ui.View { return p.layout(c) }
func (p *page) Index() int               { return p.index }
func (p *page) Id() string               { return p.id }
func (p *page) Label() string            { return p.label }
func (p *page) Icon() icons.Icon         { return p.icon }
