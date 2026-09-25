package windows

import (
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func TestWriteSyso(t *testing.T) {
	dir := t.TempDir()
	icon := filepath.Join(dir, "icon.png")
	f, err := os.Create(icon)
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(f, image.NewNRGBA(image.Rect(0, 0, 512, 512))); err != nil {
		t.Fatal(err)
	}
	f.Close()

	for _, arch := range []string{"amd64", "arm64"} {
		syso := filepath.Join(dir, "rsrc_windows_"+arch+".syso")
		err := WriteSyso(syso, Resources{Name: "Demo", Version: "1.2.3", Icon: icon, Arch: arch})
		if err != nil {
			t.Fatal(err)
		}
		if st, err := os.Stat(syso); err != nil || st.Size() == 0 {
			t.Fatalf("%s: syso missing or empty (%v)", arch, err)
		}
	}
}

func TestFileVersion(t *testing.T) {
	for in, want := range map[string]string{
		"1.2.3":      "1.2.3.0",
		"v0.7.0":     "0.7.0.0",
		"1.2.3-rc.1": "1.2.3.0",
		"2":          "2.0.0.0",
		"":           "0.0.0.0",
	} {
		if got := fileVersion(in); got != want {
			t.Errorf("fileVersion(%q) = %q, want %q", in, got, want)
		}
	}
}
