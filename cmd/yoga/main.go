// Command yoga builds and packages Yoga applications for web, macOS, Linux, and Windows.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/mirzakhany/yoga/internal/pack"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	cmd := os.Args[1]
	args := os.Args[2:]

	workDir, err := os.Getwd()
	if err != nil {
		fatal(err)
	}

	switch cmd {
	case "build":
		fatal(cmdBuild(workDir, args))
	case "package", "pkg":
		fatal(cmdPackage(workDir, args))
	case "run":
		fatal(cmdRun(workDir, args))
	case "serve":
		fatal(cmdServe(workDir, args))
	case "help", "-h", "--help":
		usage()
	case "version":
		fmt.Println("yoga-cli 0.1.0")
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n", cmd)
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, `yoga — build and package Yoga apps

Usage:
  yoga build   [-os web|darwin|linux|windows] [-arch ARCH] [-o DIR] [path]
  yoga package [-os web|darwin|linux|windows] [-arch ARCH] [-id BUNDLE_ID] [-format dmg,pkg] [path]
  yoga run     [path] [-- app-args...]
  yoga serve   [-addr host:port] [dir]
  yoga version

Config (optional yoga.toml):
  name, id, version, main, icon, [window], [darwin] (category, copyright, formats,
  dmg background/icon positions, sign identities)

Examples:
  yoga package -os web ./example/todo
  yoga package -os darwin -format dmg,pkg ./example/todo
  yoga package -os darwin -id com.example.todo ./example/todo
  yoga serve -addr 127.0.0.1:8080
  yoga serve ./dist/web
`)
}

func cmdBuild(cwd string, args []string) error {
	fs := flag.NewFlagSet("build", flag.ExitOnError)
	osFlag := fs.String("os", "host", "target OS: web|darwin|linux|windows|host")
	arch := fs.String("arch", "", "target arch (default: host)")
	out := fs.String("o", "", "output directory (default: <cwd>/dist/<os>)")
	_ = fs.Parse(args)
	appDir, err := resolveAppDir(cwd, fs.Args())
	if err != nil {
		return err
	}
	cfg, err := pack.LoadConfig(appDir)
	if err != nil {
		return err
	}
	outDir := *out
	if outDir == "" {
		outDir = filepath.Join(cwd, "dist", pack.TargetOS(*osFlag))
	}
	_, err = pack.Build(pack.BuildOpts{
		Config:  cfg,
		OS:      *osFlag,
		Arch:    *arch,
		WorkDir: appDir,
		OutDir:  outDir,
	})
	return err
}

func cmdPackage(cwd string, args []string) error {
	fs := flag.NewFlagSet("package", flag.ExitOnError)
	osFlag := fs.String("os", "host", "target OS: web|darwin|linux|windows|host")
	arch := fs.String("arch", "", "target arch (default: host)")
	id := fs.String("id", "", "bundle id override (CFBundleIdentifier)")
	formats := fs.String("format", "", "darwin formats: dmg,pkg,app (comma-separated)")
	_ = fs.Parse(args)
	appDir, err := resolveAppDir(cwd, fs.Args())
	if err != nil {
		return err
	}
	cfg, err := pack.LoadConfig(appDir)
	if err != nil {
		return err
	}
	cfg.ApplyCLIOverrides(*id, *formats)
	return pack.Package(pack.PackageOpts{
		Config:  cfg,
		OS:      *osFlag,
		Arch:    *arch,
		AppDir:  appDir,
		OutRoot: cwd,
	})
}

func cmdRun(workDir string, args []string) error {
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	_ = fs.Parse(args)
	rest := fs.Args()
	var appArgs []string
	appDir := workDir
	if len(rest) > 0 {
		if rest[0] == "--" {
			appArgs = rest[1:]
		} else {
			var err error
			appDir, err = resolveAppDir(workDir, rest[:1])
			if err != nil {
				return err
			}
			if len(rest) > 1 && rest[1] == "--" {
				appArgs = rest[2:]
			} else if len(rest) > 1 {
				appArgs = rest[1:]
			}
		}
	}
	cfg, err := pack.LoadConfig(appDir)
	if err != nil {
		return err
	}
	return pack.Run(cfg, appDir, appArgs)
}

func cmdServe(cwd string, args []string) error {
	fs := flag.NewFlagSet("serve", flag.ExitOnError)
	addr := fs.String("addr", "127.0.0.1:8080", "listen address")
	_ = fs.Parse(args)
	dir := filepath.Join(cwd, "dist", "web")
	if rest := fs.Args(); len(rest) > 0 {
		dir = rest[0]
		if !filepath.IsAbs(dir) {
			dir = filepath.Join(cwd, dir)
		}
	}
	return pack.Serve(dir, *addr)
}

func resolveAppDir(cwd string, args []string) (string, error) {
	if len(args) == 0 {
		return cwd, nil
	}
	p := args[0]
	if !filepath.IsAbs(p) {
		p = filepath.Join(cwd, p)
	}
	st, err := os.Stat(p)
	if err != nil {
		return "", err
	}
	if !st.IsDir() {
		return filepath.Dir(p), nil
	}
	return p, nil
}

func fatal(err error) {
	if err == nil {
		return
	}
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
