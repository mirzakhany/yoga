package windows

import (
	"fmt"
	"image"
	_ "image/png" // app icons are PNG
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/tc-hib/winres"
	"github.com/tc-hib/winres/version"
)

// Resources describes what WriteSyso embeds into the exe.
type Resources struct {
	Name        string
	Version     string
	Icon        string // PNG or ICO; optional
	Company     string
	Description string
	Copyright   string
	Arch        string // GOARCH
}

// iconSizes are the images an app icon carries: Explorer, taskbar and
// title bar pick the closest one for the current DPI.
var iconSizes = []int{16, 20, 24, 32, 40, 48, 64, 128, 256}

// WriteSyso writes a COFF resource object that `go build` links into the
// exe when it sits in the main package directory. It carries the app icon,
// a manifest that asks for per-monitor DPI awareness, and version info.
func WriteSyso(path string, r Resources) error {
	var rs winres.ResourceSet

	if r.Icon != "" {
		icon, err := loadIcon(r.Icon)
		if err != nil {
			return fmt.Errorf("windows: icon: %w", err)
		}
		// GLFW loads "GLFW_ICON" for the window class, so one resource
		// serves both Explorer and the running window's title bar.
		if err := rs.SetIcon(winres.Name("GLFW_ICON"), icon); err != nil {
			return fmt.Errorf("windows: icon: %w", err)
		}
	}

	rs.SetManifest(winres.AppManifest{
		DPIAwareness:        winres.DPIPerMonitorV2,
		UseCommonControlsV6: true,
		LongPathAware:       true,
	})

	var vi version.Info
	v := fileVersion(r.Version)
	vi.SetFileVersion(v)
	vi.SetProductVersion(v)
	for key, val := range map[string]string{
		version.ProductName:      r.Name,
		version.ProductVersion:   r.Version,
		version.FileVersion:      r.Version,
		version.FileDescription:  firstNonEmpty(r.Description, r.Name),
		version.CompanyName:      r.Company,
		version.LegalCopyright:   r.Copyright,
		version.OriginalFilename: r.Name + ".exe",
	} {
		if val == "" {
			continue
		}
		if err := vi.Set(0, key, val); err != nil {
			return fmt.Errorf("windows: version info: %w", err)
		}
	}
	rs.SetVersionInfo(vi)

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	if err := rs.WriteObject(f, winres.Arch(r.Arch)); err != nil {
		f.Close()
		os.Remove(path)
		return fmt.Errorf("windows: resources: %w", err)
	}
	return f.Close()
}

func loadIcon(path string) (*winres.Icon, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	if strings.EqualFold(filepath.Ext(path), ".ico") {
		return winres.LoadICO(f)
	}
	img, _, err := image.Decode(f)
	if err != nil {
		return nil, err
	}
	return winres.NewIconFromResizedImage(img, iconSizes)
}

// fileVersion turns "1.2.3-beta" into the four-part "1.2.3.0" that the
// binary version fields need.
func fileVersion(v string) string {
	v = strings.TrimPrefix(v, "v")
	if i := strings.IndexAny(v, "-+"); i >= 0 {
		v = v[:i]
	}
	parts := strings.Split(v, ".")
	out := make([]string, 4)
	for i := range out {
		out[i] = "0"
		if i < len(parts) {
			if n, err := strconv.ParseUint(parts[i], 10, 16); err == nil {
				out[i] = strconv.FormatUint(n, 10)
			}
		}
	}
	return strings.Join(out, ".")
}

func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}
