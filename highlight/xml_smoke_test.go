//go:build !js

package highlight

import (
	"testing"
	"time"
)

func TestXMLHighlightSmoke(t *testing.T) {
	h := NewXML()
	defer h.Close()
	src := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<!-- a comment -->
<root id="top" enabled="true">
  <item n="1">text &amp; more</item>
  <empty/>
</root>`)
	h.Update(src)

	var toks []Token
	for i := 0; i < 200; i++ {
		if got, ok := h.Poll(); ok {
			toks = got
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if len(toks) == 0 {
		t.Fatalf("expected XML tokens, got none")
	}

	var tags, attrNames, strings, comments int
	for _, tk := range toks {
		switch tk.Class {
		case ClassKeyword:
			tags++
		case ClassType:
			attrNames++
		case ClassString:
			strings++
		case ClassComment:
			comments++
		}
	}
	t.Logf("tokens=%d tags=%d attrNames=%d strings=%d comments=%d", len(toks), tags, attrNames, strings, comments)
	if tags == 0 || attrNames == 0 || strings == 0 || comments == 0 {
		t.Fatalf("missing expected classes: tags=%d attrNames=%d strings=%d comments=%d", tags, attrNames, strings, comments)
	}

	// Spot-check that the comment token covers "<!-- a comment -->".
	for _, tk := range toks {
		if tk.Class == ClassComment {
			if got := string(src[tk.Start:tk.End]); got != "<!-- a comment -->" {
				t.Fatalf("comment token = %q", got)
			}
			break
		}
	}
}

// TestForPathXML verifies extension routing picks the XML highlighter for
// XML dialects and Noop for plain text.
func TestForPathXML(t *testing.T) {
	xmlPaths := []string{"a.xml", "b.XmL", "c.svg", "d.xsd", "e.xsl"}
	for _, path := range xmlPaths {
		h := ForPath(path)
		if _, isNoop := h.(Noop); isNoop {
			t.Errorf("ForPath(%q): got Noop, want the XML highlighter", path)
		}
		h.Close()
	}
	for _, path := range []string{"f.txt", "g.md"} {
		h := ForPath(path)
		if _, isNoop := h.(Noop); !isNoop {
			t.Errorf("ForPath(%q): want Noop", path)
		}
		h.Close()
	}
}
