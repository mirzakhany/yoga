package main

import "github.com/mirzakhany/yoga/ui"

type Page interface {
	Layout(c *ui.Ctx) ui.View
	Index() int
	Id() string
	Label() string
	Icon() string
}

type page struct {
	index  int
	id     string
	label  string
	icon   string
	layout func(c *ui.Ctx) ui.View
}

func NewPage(index int, id string, label string, icon string, layout func(c *ui.Ctx) ui.View) Page {
	return &page{index: index, id: id, label: label, icon: icon, layout: layout}
}

func (p *page) Layout(c *ui.Ctx) ui.View { return p.layout(c) }
func (p *page) Index() int               { return p.index }
func (p *page) Id() string               { return p.id }
func (p *page) Label() string            { return p.label }
func (p *page) Icon() string             { return p.icon }
