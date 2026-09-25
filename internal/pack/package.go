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
		abs := func(p string) string { return absIn(appDir, p) }
		var appPos, appsPos, winPos [2]int
		if len(cfg.Darwin.DMG.AppPos) >= 2 {
			appPos = [2]int{cfg.Darwin.DMG.AppPos[0], cfg.Darwin.DMG.AppPos[1]}
		}
		if len(cfg.Darwin.DMG.ApplicationsPos) >= 2 {
			appsPos = [2]int{cfg.Darwin.DMG.ApplicationsPos[0], cfg.Darwin.DMG.ApplicationsPos[1]}
		}
		if len(cfg.Darwin.DMG.WindowPos) >= 2 {
			winPos = [2]int{cfg.Darwin.DMG.WindowPos[0], cfg.Darwin.DMG.WindowPos[1]}
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
			Icon:          abs(cfg.Darwin.Icon),
			OutDir:        outDir,
			Artifact:      cfg.ArtifactName("darwin", arch),
			Formats:       cfg.Darwin.Formats,
			Notarize:      cfg.Darwin.Notarize,
			DMG: darwin.DMGOptions{
				Background:      abs(cfg.Darwin.DMG.Background),
				VolumeName:      cfg.Darwin.DMG.VolumeName,
				VolumeIcon:      abs(cfg.Darwin.DMG.VolumeIcon),
				WindowPos:       winPos,
				WindowWidth:     cfg.Darwin.DMG.WindowWidth,
				WindowHeight:    cfg.Darwin.DMG.WindowHeight,
				IconSize:        cfg.Darwin.DMG.IconSize,
				AppPos:          appPos,
				ApplicationsPos: appsPos,
			},
			Sign: darwin.SignOptions{
				Identity:          cfg.Darwin.Sign.Identity,
				InstallerIdentity: cfg.Darwin.Sign.InstallerIdentity,
				Entitlements:      abs(cfg.Darwin.Sign.Entitlements),
			},
		})
	case "linux":
		bin, err := Build(BuildOpts{Config: cfg, OS: "linux", Arch: arch, WorkDir: appDir, OutDir: outDir})
		if err != nil {
			return err
		}
		return linux.Package(linux.Options{
			Name:     cfg.Name,
			ID:       cfg.ID,
			Version:  cfg.Version,
			Binary:   bin,
			BinName:  cfg.Linux.Binary,
			Icon:     absIn(appDir, cfg.Icon),
			Arch:     arch,
			OutDir:   outDir,
			Artifact: cfg.ArtifactName("linux", arch),
			Format:   cfg.Linux.Format,
			Files:    absFiles(appDir, cfg.Linux.Files),
		})
	case "windows":
		bin, err := Build(BuildOpts{Config: cfg, OS: "windows", Arch: arch, WorkDir: appDir, OutDir: outDir})
		if err != nil {
			return err
		}
		return windows.Package(windows.Options{
			Name:     cfg.Name,
			Binary:   bin,
			OutDir:   outDir,
			Artifact: cfg.ArtifactName("windows", arch),
			Files:    absFiles(appDir, cfg.Windows.Files),
		})
	default:
		return fmt.Errorf("unsupported OS %q (want web|darwin|linux|windows)", target)
	}
}

// absIn resolves a config path against the app directory.
func absIn(appDir, p string) string {
	if p == "" || filepath.IsAbs(p) {
		return p
	}
	return filepath.Join(appDir, p)
}

func absFiles(appDir string, files map[string]string) map[string]string {
	if len(files) == 0 {
		return nil
	}
	out := make(map[string]string, len(files))
	for dst, src := range files {
		out[dst] = absIn(appDir, src)
	}
	return out
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
