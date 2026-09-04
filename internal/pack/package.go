package pack

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/mirzakhany/yoga/internal/pack/darwin"
	"github.com/mirzakhany/yoga/internal/pack/linux"
	"github.com/mirzakhany/yoga/internal/pack/web"
	"github.com/mirzakhany/yoga/internal/pack/windows"
)

// PackageOpts controls build+package.
type PackageOpts struct {
	Config  Config
	OS      string
	Arch    string
	AppDir  string // app source / yoga.toml directory
	OutRoot string // directory that receives dist/ (usually the CLI cwd)
}

// Package builds and wraps the app for the target OS.
// Artifacts are written to OutRoot/dist/<os>/ so `yoga serve` from the same
// directory finds dist/web after `yoga package -os web [app]`.
func Package(opts PackageOpts) error {
	cfg := opts.Config
	target := TargetOS(opts.OS)
	arch := opts.Arch
	if arch == "" {
		arch = runtime.GOARCH
	}
	appDir := opts.AppDir
	if appDir == "" {
		appDir = opts.OutRoot
	}
	outRoot := opts.OutRoot
	if outRoot == "" {
		outRoot = appDir
	}
	outDir := filepath.Join(outRoot, "dist", target)

	switch target {
	case "web":
		return web.Build(web.Options{
			Name:    cfg.Name,
			Title:   cfg.Window.Title,
			Version: cfg.Version,
			Main:    cfg.Main,
			OutDir:  outDir,
			WorkDir: appDir,
		})
	case "darwin":
		bin, err := Build(BuildOpts{Config: cfg, OS: "darwin", Arch: arch, WorkDir: appDir, OutDir: outDir})
		if err != nil {
			return err
		}
		icon := cfg.Icon
		if icon != "" && !filepath.IsAbs(icon) {
			icon = filepath.Join(appDir, icon)
		}
		bg := cfg.Darwin.DMG.Background
		if bg != "" && !filepath.IsAbs(bg) {
			bg = filepath.Join(appDir, bg)
		}
		ents := cfg.Darwin.Sign.Entitlements
		if ents != "" && !filepath.IsAbs(ents) {
			ents = filepath.Join(appDir, ents)
		}
		var appPos, appsPos [2]int
		if len(cfg.Darwin.DMG.AppPos) >= 2 {
			appPos = [2]int{cfg.Darwin.DMG.AppPos[0], cfg.Darwin.DMG.AppPos[1]}
		}
		if len(cfg.Darwin.DMG.ApplicationsPos) >= 2 {
			appsPos = [2]int{cfg.Darwin.DMG.ApplicationsPos[0], cfg.Darwin.DMG.ApplicationsPos[1]}
		}
		return darwin.Package(darwin.Options{
			Name:          cfg.Name,
			DisplayName:   cfg.Darwin.DisplayName,
			ID:            cfg.BundleID(),
			Version:       cfg.Version,
			BundleVersion: cfg.Darwin.BundleVersion,
			Copyright:     cfg.Darwin.Copyright,
			Category:      cfg.Darwin.Category,
			MinSystem:     cfg.Darwin.MinSystem,
			Binary:        bin,
			Icon:          icon,
			OutDir:        outDir,
			Formats:       cfg.Darwin.Formats,
			DMG: darwin.DMGOptions{
				Background:      bg,
				VolumeName:      cfg.Darwin.DMG.VolumeName,
				WindowWidth:     cfg.Darwin.DMG.WindowWidth,
				WindowHeight:    cfg.Darwin.DMG.WindowHeight,
				IconSize:        cfg.Darwin.DMG.IconSize,
				AppPos:          appPos,
				ApplicationsPos: appsPos,
			},
			Sign: darwin.SignOptions{
				Identity:          cfg.Darwin.Sign.Identity,
				InstallerIdentity: cfg.Darwin.Sign.InstallerIdentity,
				Entitlements:      ents,
			},
		})
	case "linux":
		bin, err := Build(BuildOpts{Config: cfg, OS: "linux", Arch: arch, WorkDir: appDir, OutDir: outDir})
		if err != nil {
			return err
		}
		icon := cfg.Icon
		if icon != "" && !filepath.IsAbs(icon) {
			icon = filepath.Join(appDir, icon)
		}
		return linux.Package(linux.Options{
			Name:    cfg.Name,
			ID:      cfg.ID,
			Version: cfg.Version,
			Binary:  bin,
			Icon:    icon,
			Arch:    arch,
			OutDir:  outDir,
		})
	case "windows":
		bin, err := Build(BuildOpts{Config: cfg, OS: "windows", Arch: arch, WorkDir: appDir, OutDir: outDir})
		if err != nil {
			return err
		}
		icon := cfg.Icon
		if icon != "" && !filepath.IsAbs(icon) {
			icon = filepath.Join(appDir, icon)
		}
		return windows.Package(windows.Options{
			Name:    cfg.Name,
			Version: cfg.Version,
			Binary:  bin,
			Icon:    icon,
			Arch:    arch,
			OutDir:  outDir,
		})
	default:
		return fmt.Errorf("unsupported OS %q (want web|darwin|linux|windows)", target)
	}
}

// Run launches the app on the host with go run.
func Run(cfg Config, workDir string, extraArgs []string) error {
	args := append([]string{"run", cfg.Main}, extraArgs...)
	cmd := exec.Command("go", args...)
	cmd.Dir = workDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	cmd.Env = append(os.Environ(), "CGO_ENABLED=1")
	return cmd.Run()
}

// Serve starts a static file server for dir (the packaged web output folder).
func Serve(dir, addr string) error {
	if st, err := os.Stat(dir); err != nil || !st.IsDir() {
		return fmt.Errorf("serve: %s missing — pass the web output dir (default dist/web), or run: yoga package -os web [app]", dir)
	}
	fmt.Printf("serving %s at http://%s/\n", dir, addr)
	return http.ListenAndServe(addr, http.FileServer(http.Dir(dir)))
}
