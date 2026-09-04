package pack

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// BuildOpts controls a compile-only build.
type BuildOpts struct {
	Config  Config
	OS      string // web|darwin|linux|windows
	Arch    string
	WorkDir string
	OutDir  string // override; default dist/<os>
}

// Build compiles the app for the target OS into dist/<os>/.
func Build(opts BuildOpts) (string, error) {
	cfg := opts.Config
	target := TargetOS(opts.OS)
	arch := opts.Arch
	if arch == "" {
		arch = runtime.GOARCH
	}
	outDir := opts.OutDir
	if outDir == "" {
		outDir = filepath.Join(opts.WorkDir, "dist", target)
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return "", err
	}

	if target == "web" {
		// Web build is handled by package web (wasm + assets). Compile-only still emits wasm.
		wasmPath := filepath.Join(outDir, "app.wasm")
		cmd := exec.Command("go", "build", "-o", wasmPath, cfg.Main)
		cmd.Dir = opts.WorkDir
		cmd.Env = append(os.Environ(), "GOOS=js", "GOARCH=wasm", "CGO_ENABLED=0")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return "", fmt.Errorf("build web: %w", err)
		}
		return wasmPath, nil
	}

	if target != runtime.GOOS {
		fmt.Fprintf(os.Stderr, "warning: packaging/building %s on host %s may fail (CGO/GLFW); prefer building on the target OS\n", target, runtime.GOOS)
	}

	binName := cfg.Name
	if target == "windows" {
		binName += ".exe"
	}
	outPath := filepath.Join(outDir, binName)
	cmd := exec.Command("go", "build", "-o", outPath, cfg.Main)
	cmd.Dir = opts.WorkDir
	env := append(os.Environ(),
		"GOOS="+target,
		"GOARCH="+arch,
	)
	// Desktop Yoga requires CGO for GLFW/wgpu-native.
	if target != "web" {
		env = append(env, "CGO_ENABLED=1")
	}
	cmd.Env = env
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("build %s: %w", target, err)
	}
	return outPath, nil
}

// BinaryName returns the expected executable name for a target.
func BinaryName(cfg Config, target string) string {
	name := cfg.Name
	if target == "windows" {
		if !strings.HasSuffix(name, ".exe") {
			name += ".exe"
		}
	}
	return name
}
