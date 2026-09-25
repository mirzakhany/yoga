package pack

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/mirzakhany/yoga/internal/pack/windows"
)

// BuildOpts controls a compile-only build.
type BuildOpts struct {
	Config  Config
	OS      string // web|darwin|linux|windows
	Arch    string // GOARCH, or "universal" for a darwin amd64+arm64 binary
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
		cmd := exec.Command("go", goBuildArgs(cfg, target, wasmPath)...)
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
	if target == "linux" {
		binName = cfg.Linux.Binary
	}
	if target == "windows" {
		binName += ".exe"
	}
	outPath := filepath.Join(outDir, binName)

	if target == "darwin" && arch == "universal" {
		return outPath, buildUniversal(opts, outPath)
	}

	if target == "windows" {
		syso := filepath.Join(opts.WorkDir, cfg.Main, "zz_yoga_rsrc_windows_"+arch+".syso")
		icon := cfg.Icon
		if icon != "" && !filepath.IsAbs(icon) {
			icon = filepath.Join(opts.WorkDir, icon)
		}
		err := windows.WriteSyso(syso, windows.Resources{
			Name:        cfg.Name,
			Version:     cfg.Version,
			Icon:        icon,
			Company:     cfg.Windows.Company,
			Description: cfg.Windows.Description,
			Copyright:   cfg.Windows.Copyright,
			Arch:        arch,
		})
		if err != nil {
			return "", err
		}
		defer os.Remove(syso)
	}

	if err := goBuild(cfg, opts.WorkDir, target, arch, outPath); err != nil {
		return "", err
	}
	return outPath, nil
}

// buildUniversal compiles darwin amd64 and arm64 binaries and joins them
// with lipo.
func buildUniversal(opts BuildOpts, outPath string) error {
	var parts []string
	for _, arch := range []string{"amd64", "arm64"} {
		part := outPath + "-" + arch
		if err := goBuild(opts.Config, opts.WorkDir, "darwin", arch, part); err != nil {
			return err
		}
		defer os.Remove(part)
		parts = append(parts, part)
	}
	cmd := exec.Command("lipo", append([]string{"-create", "-output", outPath}, parts...)...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("build darwin universal: lipo: %w", err)
	}
	return nil
}

func goBuild(cfg Config, workDir, target, arch, outPath string) error {
	cmd := exec.Command("go", goBuildArgs(cfg, target, outPath)...)
	cmd.Dir = workDir
	// Desktop Yoga requires CGO for GLFW/wgpu-native.
	env := append(os.Environ(),
		"GOOS="+target,
		"GOARCH="+arch,
		"CGO_ENABLED=1",
	)
	if target == "darwin" {
		// Keep the C parts (GLFW, tree-sitter) loadable on the oldest macOS
		// Info.plist allows. The flag goes in CGO_CFLAGS, not only
		// MACOSX_DEPLOYMENT_TARGET: the build cache keys on the former, so
		// objects cached from a plain `go build` are not reused.
		min := "-mmacosx-version-min=" + cfg.Darwin.MinSystem
		env = append(env,
			"MACOSX_DEPLOYMENT_TARGET="+cfg.Darwin.MinSystem,
			"CGO_CFLAGS="+withDefault(os.Getenv("CGO_CFLAGS"), "-O2 -g")+" "+min,
			"CGO_LDFLAGS="+withDefault(os.Getenv("CGO_LDFLAGS"), "-O2 -g")+" "+min,
		)
	}
	cmd.Env = env
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("build %s/%s: %w", target, arch, err)
	}
	return nil
}

func goBuildArgs(cfg Config, target, outPath string) []string {
	args := []string{"build"}
	args = append(args, cfg.Build.Flags...)
	if len(cfg.Build.Tags) > 0 {
		args = append(args, "-tags", strings.Join(cfg.Build.Tags, ","))
	}
	ldflags := strings.ReplaceAll(cfg.Build.Ldflags, "{version}", cfg.Version)
	if target == "windows" && !cfg.Windows.Console {
		ldflags = strings.TrimSpace(ldflags + " -H=windowsgui")
	}
	if ldflags != "" {
		args = append(args, "-ldflags", ldflags)
	}
	return append(args, "-o", outPath, cfg.Main)
}

func withDefault(v, def string) string {
	if v == "" {
		return def
	}
	return v
}
