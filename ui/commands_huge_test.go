package ui

import (
	"strconv"
	"testing"
	"time"

	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/layout"
)

// A scope with tens of thousands of entries, such as every file of a big
// project, must open without building a row for each one.
func TestCommandsHugeScopeOpensFast(t *testing.T) {
	c := newTestCtx(t)
	const n = 50000
	ran := -1
	cmds := make([]*Command, n)
	for i := range cmds {
		s := strconv.Itoa(i)
		cmds[i] = Item("open." + s).Title("file" + s + ".go").Detail("pkg/dir" + s).Scope("files").
			Run(func() { ran = i })
	}
	var host *CommandsHost
	body := func(c *Ctx) View {
		host = c.Commands()
		host.Register(cmds...)
		return Column(Text("page")).Grow(1)
	}
	mouse := &input.Mouse{X: -1, Y: -1}
	BuildFrame(c, body, 800, 600, mouse, nil)
	host.ShowScope("files", "Search files")
	start := time.Now()
	for range 3 {
		BuildFrame(c, body, 800, 600, mouse, nil)
	}
	if d := time.Since(start) / 3; d > 100*time.Millisecond {
		t.Fatalf("frame with %d entries took %v", n, d)
	}
	if got, want := host.listSV.contentH, float32(n)*host.rowHeight(c.Theme()); got < want-1 {
		t.Fatalf("list content height %v, want %v", got, want)
	}

	// End scrolls to the last entry; clicking the list's last row runs it.
	host.FilterKeys([]input.KeyEvent{{Key: input.KeyEnd}})
	BuildFrame(c, body, 800, 600, mouse, nil)
	view := host.listSV.host.Frame
	mouse.X, mouse.Y = view.X+view.W/2, view.Y+view.H-host.rowHeight(c.Theme())/2
	root := BuildFrame(c, body, 800, 600, mouse, nil)
	mouse.Released = true
	layout.Dispatch(root, mouse)
	if ran != n-1 {
		t.Fatalf("clicked the last row, ran %d, want %d", ran, n-1)
	}

	host.ShowScope("files", "Search files")
	host.query = "file49999"
	start = time.Now()
	BuildFrame(c, body, 800, 600, mouse, nil)
	if d := time.Since(start); d > 100*time.Millisecond {
		t.Fatalf("filtered frame took %v", d)
	}
}
