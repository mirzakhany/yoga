package ui

import (
	"testing"
	"time"

	"github.com/mirzakhany/yoga/highlight"
)

func TestHoverDocMarksCode(t *testing.T) {
	lines, code, lang := hoverDoc("```go\nfunc Run(n int) error\n```\n\nRun runs n jobs.\n")
	if lang != "go" || code != "func Run(n int) error\n" {
		t.Fatalf("code %q lang %q", code, lang)
	}
	if len(lines) != 3 || !lines[0].code || lines[0].off != 0 || lines[2].code || lines[2].text != "Run runs n jobs." {
		t.Fatalf("lines %+v", lines)
	}
}

func TestHoverCodeGetsColors(t *testing.T) {
	e := newTestEditor(t, []byte("package p\n"))
	e.Path = "x.go"
	e.setHoverText("```go\nfunc Run(n int) error\n```")
	defer e.clearHover()
	deadline := time.Now().Add(3 * time.Second)
	for !e.lspUI.hoverCode.done && time.Now().Before(deadline) {
		e.pollHoverCode()
		time.Sleep(5 * time.Millisecond)
	}
	hc := e.lspUI.hoverCode
	if !hc.done || len(hc.toks) == 0 {
		t.Fatal("no colors for the hover's code")
	}
	var sawKeyword, sawType bool
	for _, tk := range hc.toks {
		sawKeyword = sawKeyword || tk.Class == highlight.ClassKeyword
		sawType = sawType || tk.Class == highlight.ClassType
	}
	if !sawKeyword || !sawType {
		t.Fatalf("tokens %v: want a keyword and a type", hc.toks)
	}
}
