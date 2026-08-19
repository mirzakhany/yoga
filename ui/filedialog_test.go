package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/shape"
)

func TestFileDialogLayoutRegistersOverlays(t *testing.T) {
	text, err := shape.NewEngine(1, false)
	if err != nil {
		t.Fatal(err)
	}
	SetFrameResources(text, nil, nil)

	dir := t.TempDir()
	d := NewFileDialog()
	d.Show(FileDialogOpts{Dir: dir, Title: "Open File"})

	c := New(text, NewFocusScope(), nil)
	c.BeginFrame(800, 600, nil, nil)
	d.Layout(c)
	if got := len(c.Overlays()); got < 2 {
		t.Fatalf("open file dialog should register scrim+panel, got %d", got)
	}
	if !d.Open {
		t.Fatal("dialog should stay open")
	}
}

func TestFileDialogPanelStacksAboveScrimWithPriorOverlay(t *testing.T) {
	text, err := shape.NewEngine(1, false)
	if err != nil {
		t.Fatal(err)
	}
	SetFrameResources(text, nil, nil)

	d := NewFileDialog()
	d.Show(FileDialogOpts{Dir: t.TempDir()})

	prior := layout.New(layout.Box())
	prior.Overlay = true

	c := New(text, NewFocusScope(), nil)

	root := BuildFrame(c, func(c *Ctx) View {
		c.Overlay(prior)
		return d
	}, 800, 600, nil, nil)

	ov := c.Overlays()
	scrimI, panelI := -1, -1
	for i, e := range ov {
		if e == d.scrim.host {
			scrimI = i
		}
		if e == d.panel {
			panelI = i
		}
	}
	if scrimI < 0 || panelI < 0 {
		t.Fatalf("missing scrim/panel in overlays (n=%d scrim=%d panel=%d)", len(ov), scrimI, panelI)
	}
	if !(scrimI < panelI) {
		t.Fatalf("panel must paint above scrim: scrim=%d panel=%d", scrimI, panelI)
	}
	if ov[0] != prior {
		t.Fatal("prior overlay should remain first")
	}

	// Click inside the panel: the scrim must not win hit-testing.
	f := d.panel.Frame
	mouse := &input.Mouse{X: f.X + f.W/2, Y: f.Y + f.H/2, Released: true}
	layout.Dispatch(root, mouse)
	if !mouse.Consumed {
		t.Fatal("click on the panel should be consumed by the dialog, not fall through")
	}
}

func TestFileDialogConfirmFilesAndFolders(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "note.txt")
	sub := filepath.Join(dir, "docs")
	if err := os.WriteFile(filePath, []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatal(err)
	}

	d := NewFileDialog()
	var got []string
	d.Show(FileDialogOpts{
		Dir: dir,
		Filters: []FileFilter{
			{Label: "Text", Exts: []string{".txt"}},
			{Label: "All files", Exts: nil},
		},
		OnConfirm: func(paths []string) { got = append([]string(nil), paths...) },
	})
	idx := fileRow(d, filePath)
	if idx < 0 {
		t.Fatalf("missing file row, have %v", rowNames(d))
	}
	d.table.Rows[idx].Selected = true
	if !d.canConfirm() {
		t.Fatal("selecting a file should enable confirm")
	}
	d.confirm()
	if len(got) != 1 || got[0] != filePath {
		t.Fatalf("confirm files: %v", got)
	}
	if d.Open {
		t.Fatal("confirm should close")
	}

	got = nil
	d.Show(FileDialogOpts{
		Mode:      FileDialogOpenFolder,
		Dir:       dir,
		OnConfirm: func(paths []string) { got = append([]string(nil), paths...) },
	})
	d.confirm()
	if len(got) != 1 || got[0] != dir {
		t.Fatalf("folder mode with no selection confirms cwd: %v want %v", got, dir)
	}

	got = nil
	d.Show(FileDialogOpts{
		Mode:      FileDialogOpenFolder,
		Dir:       dir,
		OnConfirm: func(paths []string) { got = append([]string(nil), paths...) },
	})
	if i := fileRow(d, sub); i >= 0 {
		d.table.Rows[i].Selected = true
	}
	d.confirm()
	if len(got) != 1 || got[0] != sub {
		t.Fatalf("folder row confirm: %v", got)
	}
}

func TestFileDialogCancel(t *testing.T) {
	d := NewFileDialog()
	canceled := false
	d.Show(FileDialogOpts{
		Dir:      t.TempDir(),
		OnCancel: func() { canceled = true },
	})
	d.HandleKeys([]input.KeyEvent{{Key: input.KeyEscape}})
	if d.Open {
		t.Fatal("escape should close")
	}
	if !canceled {
		t.Fatal("OnCancel should run")
	}
}

func TestFileDialogFilterHidesNonMatchingFiles(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.go"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "b.md"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	d := NewFileDialog()
	d.Show(FileDialogOpts{
		Dir: dir,
		Filters: []FileFilter{
			{Label: "Go", Exts: []string{".go"}},
			{Label: "All files", Exts: nil},
		},
	})
	if !hasRow(d, "a.go") || hasRow(d, "b.md") {
		t.Fatalf("go filter rows: %v", rowNames(d))
	}
	d.filterIdx = 1
	d.applyRows()
	if !hasRow(d, "a.go") || !hasRow(d, "b.md") {
		t.Fatalf("all-files rows: %v", rowNames(d))
	}
}

func TestFileDialogActivateFolderNavigates(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "inner")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sub, "x.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	d := NewFileDialog()
	d.Show(FileDialogOpts{Dir: dir})
	d.activateRow(sub)
	if d.dir != sub {
		t.Fatalf("dir: %q want %q", d.dir, sub)
	}
	if !hasRow(d, "x.txt") {
		t.Fatalf("expected child file, rows=%v", rowNames(d))
	}
}

func TestFileDialogSaveFileConfirmAddsFilterExt(t *testing.T) {
	dir := t.TempDir()
	d := NewFileDialog()
	var got []string
	d.Show(FileDialogOpts{
		Mode:           FileDialogSaveFile,
		Dir:            dir,
		ShowSaveFilter: true,
		Filters: []FileFilter{
			{Label: "Go", Exts: []string{".go"}},
		},
		OnConfirm: func(paths []string) { got = append([]string(nil), paths...) },
	})
	d.saveName = "newfile"
	if !d.canConfirm() {
		t.Fatal("save should be confirmable with file name")
	}
	d.confirm()
	want := filepath.Join(dir, "newfile.go")
	if len(got) != 1 || got[0] != want {
		t.Fatalf("save target: got %v want %q", got, want)
	}
}

func TestFileDialogSaveFileFilterOption(t *testing.T) {
	d := NewFileDialog()
	d.Show(FileDialogOpts{
		Mode:           FileDialogSaveFile,
		Dir:            t.TempDir(),
		ShowSaveFilter: true,
	})
	if !d.showFilter() {
		t.Fatal("save filter should be enabled when ShowSaveFilter is true")
	}
	d.Show(FileDialogOpts{
		Mode: FileDialogSaveFile,
		Dir:  t.TempDir(),
	})
	if d.showFilter() {
		t.Fatal("save filter should be disabled when ShowSaveFilter is false")
	}
}

func TestFileDialogSaveFileRowClickSetsName(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "note.txt")
	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	d := NewFileDialog()
	d.Show(FileDialogOpts{
		Mode: FileDialogSaveFile,
		Dir:  dir,
	})
	d.rowClick(path)
	if d.saveName != "note.txt" {
		t.Fatalf("save name: got %q", d.saveName)
	}
}

func TestFileDialogCreateFolderOption(t *testing.T) {
	dir := t.TempDir()
	d := NewFileDialog()
	d.Show(FileDialogOpts{
		Mode:              FileDialogSaveFile,
		Dir:               dir,
		AllowCreateFolder: true,
	})
	if !d.allowCreateFolder() {
		t.Fatal("AllowCreateFolder should enable create folder")
	}
	d.newFolderName = "new-dir"
	d.creatingFolder = true
	d.createFolder()
	want := filepath.Join(dir, "new-dir")
	if d.dir != want {
		t.Fatalf("should navigate into new folder: got %q want %q", d.dir, want)
	}
	if st, err := os.Stat(want); err != nil || !st.IsDir() {
		t.Fatalf("new folder not created: err=%v", err)
	}
}

func TestFileDialogWheelDoesNotScrollPage(t *testing.T) {
	text, err := shape.NewEngine(1, false)
	if err != nil {
		t.Fatal(err)
	}
	SetFrameResources(text, nil, nil)

	dir := t.TempDir()
	for i := 0; i < 40; i++ {
		name := filepath.Join(dir, fmt.Sprintf("file-%02d.txt", i))
		if err := os.WriteFile(name, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	d := NewFileDialog()
	d.Show(FileDialogOpts{Dir: dir})

	blocks := make([]View, 0, 20)
	for i := 0; i < 20; i++ {
		blocks = append(blocks, Raw(layout.New(layout.Box().H(40).FlexShrink(0))))
	}

	c := New(text, NewFocusScope(), nil)
	body := func(_ *Ctx) View {
		return Column(
			Scroll("page", Column(blocks...).Gap(8)).Grow(1),
			d,
		).Grow(1)
	}
	mouse := &input.Mouse{X: 400, Y: 300}
	root := BuildFrame(c, body, 800, 600, mouse, nil)

	page := c.Widget("page", func() any { return NewScrollView(nil) }).(*ScrollView)
	if page.scrollY != 0 {
		t.Fatalf("setup: page already scrolled: %v", page.scrollY)
	}

	f := d.panel.Frame
	mouse.X = f.X + f.W*0.6
	mouse.Y = f.Y + f.H*0.6
	mouse.ScrollY = -4
	layout.Dispatch(root, mouse)

	if page.scrollY != 0 {
		t.Fatalf("page behind dialog should not scroll: offset=%v", page.scrollY)
	}
	if d.table.scrollY <= 0 {
		t.Fatalf("dialog table should scroll, offset=%v contentH=%v", d.table.scrollY, d.table.contentH)
	}
}

func fileRow(d *FileDialog, path string) int {
	for i, row := range d.table.Rows {
		if row.ID == path {
			return i
		}
	}
	return -1
}

func hasRow(d *FileDialog, name string) bool {
	for _, row := range d.table.Rows {
		if row.Cells["name"] == name {
			return true
		}
	}
	return false
}

func rowNames(d *FileDialog) []string {
	out := make([]string, 0, len(d.table.Rows))
	for _, row := range d.table.Rows {
		out = append(out, row.Cells["name"])
	}
	return out
}
