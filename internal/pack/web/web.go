package web

import (
	"bytes"
	"embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"text/template"
)

//go:embed templates/*
var templateFS embed.FS

// Options configures a web package output.
type Options struct {
	Name    string
	Title   string
	Version string
	Main    string // Go package path to build
	OutDir  string // e.g. dist/web
	WorkDir string // module root
}

// Build compiles the app to WASM and writes the static site into OutDir.
func Build(opts Options) error {
	if opts.Name == "" {
		opts.Name = "App"
	}
	if opts.Title == "" {
		opts.Title = opts.Name
	}
	if opts.Main == "" {
		opts.Main = "."
	}
	if opts.OutDir == "" {
		opts.OutDir = filepath.Join("dist", "web")
	}
	if err := os.MkdirAll(opts.OutDir, 0o755); err != nil {
		return err
	}

	wasmPath := filepath.Join(opts.OutDir, "app.wasm")
	cmd := exec.Command("go", "build", "-o", wasmPath, opts.Main)
	cmd.Dir = opts.WorkDir
	cmd.Env = append(os.Environ(),
		"GOOS=js",
		"GOARCH=wasm",
		"CGO_ENABLED=0",
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("web: go build wasm: %w", err)
	}

	if err := writeTemplates(opts); err != nil {
		return err
	}
	if err := copyWasmExec(opts.OutDir); err != nil {
		return err
	}
	fmt.Printf("web: wrote %s\n", opts.OutDir)
	return nil
}

func writeTemplates(opts Options) error {
	indexSrc, err := templateFS.ReadFile("templates/index.html")
	if err != nil {
		return err
	}
	tpl, err := template.New("index.html").Parse(string(indexSrc))
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	if err := tpl.Execute(&buf, opts); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(opts.OutDir, "index.html"), buf.Bytes(), 0o644); err != nil {
		return err
	}

	loader, err := templateFS.ReadFile("templates/yoga_loader.js")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(opts.OutDir, "yoga_loader.js"), loader, 0o644)
}

func copyWasmExec(outDir string) error {
	candidates := []string{
		filepath.Join(runtime.GOROOT(), "lib", "wasm", "wasm_exec.js"),
		filepath.Join(runtime.GOROOT(), "misc", "wasm", "wasm_exec.js"),
	}
	var src string
	for _, c := range candidates {
		if st, err := os.Stat(c); err == nil && !st.IsDir() {
			src = c
			break
		}
	}
	if src == "" {
		return fmt.Errorf("web: wasm_exec.js not found under GOROOT=%s", runtime.GOROOT())
	}
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(outDir, "wasm_exec.js"), data, 0o644)
}
