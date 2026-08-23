// Command generate-lucide rasterizes a pinned Lucide release into icons/*.go.
package main

import (
	"archive/zip"
	"bytes"
	"compress/gzip"
	"errors"
	"flag"
	"fmt"
	"go/format"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"github.com/mirzakhany/yoga/render"
)

const lucideVersion = "1.33.0"

type generatedIcon struct {
	exportName string
	fileName   string
	name       string
	gz         []byte
}

func main() {
	var (
		cacheDir  = flag.String("cache", ".cache/lucide-"+lucideVersion, "directory for Lucide SVG sources")
		iconsDir  = flag.String("icons", "icons", "output package directory")
		catalogDir = flag.String("catalog", "icons/catalog", "output catalog package directory")
		extraDir  = flag.String("extra", "cmd/generate-lucide/extra", "extra SVG icons (yoga, …)")
	)
	flag.Parse()

	if err := run(*cacheDir, *iconsDir, *catalogDir, *extraDir); err != nil {
		fmt.Fprintln(os.Stderr, "generate-lucide:", err)
		os.Exit(1)
	}
}

func run(cacheDir, iconsDir, catalogDir, extraDir string) error {
	if err := ensureLucideSVGs(cacheDir); err != nil {
		return err
	}
	files, err := sourceFiles(cacheDir)
	if err != nil {
		return err
	}
	extras, err := extraFiles(extraDir)
	if err != nil {
		return err
	}
	files = append(files, extras...)

	icons := make([]generatedIcon, 0, len(files))
	used := make(map[string]string, len(files))
	for _, file := range files {
		gi, err := convertFile(file)
		if err != nil {
			fmt.Fprintf(os.Stderr, "skip %s: %v\n", filepath.Base(file), err)
			continue
		}
		if prev, ok := used[gi.exportName]; ok {
			return fmt.Errorf("Go name collision: %s and %s both become %s", prev, file, gi.exportName)
		}
		used[gi.exportName] = file
		icons = append(icons, gi)
	}
	if len(icons) == 0 {
		return errors.New("no icons generated")
	}
	sort.Slice(icons, func(i, j int) bool {
		return icons[i].exportName < icons[j].exportName
	})

	if err := writeIconShards(iconsDir, icons); err != nil {
		return err
	}
	if err := writeCatalog(catalogDir, icons); err != nil {
		return err
	}
	fmt.Printf("generated %d icons (Lucide %s)\n", len(icons), lucideVersion)
	return nil
}

func ensureLucideSVGs(cacheDir string) error {
	iconsDir := filepath.Join(cacheDir, "icons")
	if entries, err := os.ReadDir(iconsDir); err == nil && len(entries) > 100 {
		return nil
	}
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return err
	}
	url := fmt.Sprintf("https://github.com/lucide-icons/lucide/releases/download/%s/lucide-icons-%s.zip", lucideVersion, lucideVersion)
	fmt.Println("downloading", url)
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download: %s", resp.Status)
	}
	zipPath := filepath.Join(cacheDir, "lucide-icons.zip")
	f, err := os.Create(zipPath)
	if err != nil {
		return err
	}
	if _, err := io.Copy(f, resp.Body); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return unzipLucide(zipPath, cacheDir)
}

func sourceFiles(dir string) ([]string, error) {
	iconsDir := filepath.Join(dir, "icons")
	entries, err := os.ReadDir(iconsDir)
	if err != nil {
		return nil, err
	}
	var files []string
	for _, e := range entries {
		if e.IsDir() || !strings.EqualFold(filepath.Ext(e.Name()), ".svg") {
			continue
		}
		files = append(files, filepath.Join(iconsDir, e.Name()))
	}
	sort.Strings(files)
	if len(files) == 0 {
		return nil, fmt.Errorf("no SVG files in %s", iconsDir)
	}
	return files, nil
}

func extraFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var files []string
	for _, e := range entries {
		if e.IsDir() || !strings.EqualFold(filepath.Ext(e.Name()), ".svg") {
			continue
		}
		files = append(files, filepath.Join(dir, e.Name()))
	}
	sort.Strings(files)
	return files, nil
}

func convertFile(file string) (generatedIcon, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return generatedIcon{}, err
	}
	data = rewriteCurrentColor(data)
	mask, err := render.RasterizeSVG(data, 40)
	if err != nil {
		return generatedIcon{}, err
	}
	nonzero := 0
	for _, p := range mask.Pix {
		if p > 0 {
			nonzero++
		}
	}
	if nonzero == 0 {
		return generatedIcon{}, errors.New("empty raster mask")
	}
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	if _, err := gw.Write(mask.Pix); err != nil {
		return generatedIcon{}, err
	}
	if err := gw.Close(); err != nil {
		return generatedIcon{}, err
	}
	base := strings.TrimSuffix(filepath.Base(file), filepath.Ext(file))
	return generatedIcon{
		exportName: exportName(base),
		fileName:   filepath.Base(file),
		name:       base,
		gz:         buf.Bytes(),
	}, nil
}

func rewriteCurrentColor(svg []byte) []byte {
	s := string(svg)
	s = strings.ReplaceAll(s, "currentColor", "#000000")
	s = strings.ReplaceAll(s, "currentcolor", "#000000")
	return []byte(s)
}

func exportName(base string) string {
	var name strings.Builder
	upperNext := true
	for _, value := range base {
		if !unicode.IsLetter(value) && !unicode.IsDigit(value) {
			upperNext = true
			continue
		}
		if name.Len() == 0 && unicode.IsDigit(value) {
			name.WriteByte('X')
		}
		if upperNext {
			value = unicode.ToUpper(value)
			upperNext = false
		}
		name.WriteRune(value)
	}
	if name.Len() == 0 {
		return "Icon"
	}
	return name.String()
}

func writeIconShards(outDir string, icons []generatedIcon) error {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}
	if err := removeGenerated(filepath.Join(outDir, "icons_*_gen.go")); err != nil {
		return err
	}
	shards := make(map[byte][]generatedIcon)
	for _, ic := range icons {
		key := byte(unicode.ToLower(rune(ic.exportName[0])))
		if key < 'a' || key > 'z' {
			key = 'x'
		}
		shards[key] = append(shards[key], ic)
	}
	for key, values := range shards {
		var out bytes.Buffer
		fmt.Fprintln(&out, "// Code generated by cmd/generate-lucide; DO NOT EDIT.")
		fmt.Fprintln(&out)
		fmt.Fprintln(&out, "package icons")
		fmt.Fprintln(&out)
		for _, ic := range values {
			fmt.Fprintf(&out, "\n// %s is the Lucide %q icon.\n", ic.exportName, ic.name)
			fmt.Fprintf(&out, "var %s = newIcon(%q, []byte(%s))\n", ic.exportName, ic.name, strconv.QuoteToASCII(string(ic.gz)))
		}
		src, err := format.Source(out.Bytes())
		if err != nil {
			return fmt.Errorf("format shard %c: %w\n%s", key, err, out.String())
		}
		if err := os.WriteFile(filepath.Join(outDir, fmt.Sprintf("icons_%c_gen.go", key)), src, 0o644); err != nil {
			return err
		}
	}
	return nil
}

func writeCatalog(outDir string, icons []generatedIcon) error {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}
	var out bytes.Buffer
	fmt.Fprintln(&out, "// Code generated by cmd/generate-lucide; DO NOT EDIT.")
	fmt.Fprintln(&out)
	fmt.Fprintln(&out, "package catalog")
	fmt.Fprintln(&out)
	fmt.Fprintln(&out, "import \"github.com/mirzakhany/yoga/icons\"")
	fmt.Fprintln(&out)
	fmt.Fprintf(&out, "// Count is the number of generated icons (Lucide %s + extras).\n", lucideVersion)
	fmt.Fprintf(&out, "const Count = %d\n\n", len(icons))
	fmt.Fprintln(&out, "// All lists every icon for the component catalog browser.")
	fmt.Fprintln(&out, "var All = []icons.Icon{")
	for _, ic := range icons {
		fmt.Fprintf(&out, "\ticons.%s,\n", ic.exportName)
	}
	fmt.Fprintln(&out, "}")
	src, err := format.Source(out.Bytes())
	if err != nil {
		return fmt.Errorf("format catalog: %w\n%s", err, out.String())
	}
	if err := os.WriteFile(filepath.Join(outDir, "catalog_gen.go"), src, 0o644); err != nil {
		return err
	}
	return writeCatalogTest(outDir, icons)
}

func writeCatalogTest(outDir string, icons []generatedIcon) error {
	var out bytes.Buffer
	fmt.Fprintln(&out, "// Code generated by cmd/generate-lucide; DO NOT EDIT.")
	fmt.Fprintln(&out)
	fmt.Fprintln(&out, "package catalog")
	fmt.Fprintln(&out)
	fmt.Fprintln(&out, "import (")
	fmt.Fprintln(&out, "\t\"testing\"")
	fmt.Fprintln(&out, "\t\"github.com/mirzakhany/yoga/icons\"")
	fmt.Fprintln(&out, ")")
	fmt.Fprintln(&out)
	fmt.Fprintln(&out, "func TestGeneratedCatalog(t *testing.T) {")
	fmt.Fprintf(&out, "\tif len(All) != %d {\n\t\tt.Fatalf(\"len(All) = %%d, want %d\", len(All))\n\t}\n", len(icons), len(icons))
	fmt.Fprintln(&out, "\tfor _, ic := range All {")
	fmt.Fprintln(&out, "\t\tif ic.Empty() {")
	fmt.Fprintln(&out, "\t\t\tt.Fatalf(\"empty icon in catalog\")")
	fmt.Fprintln(&out, "\t\t}")
	fmt.Fprintln(&out, "\t\tif _, err := ic.Alpha(icons.BakePx); err != nil {")
	fmt.Fprintf(&out, "\t\t\tt.Fatalf(\"%%s: %%v\", ic.Name, err)\n")
	fmt.Fprintln(&out, "\t\t}")
	fmt.Fprintln(&out, "\t}")
	fmt.Fprintln(&out, "}")
	src, err := format.Source(out.Bytes())
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(outDir, "catalog_gen_test.go"), src, 0o644)
}

func unzipLucide(zipPath, destDir string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer r.Close()
	for _, f := range r.File {
		if f.FileInfo().IsDir() {
			continue
		}
		if !strings.HasSuffix(strings.ToLower(f.Name), ".svg") {
			continue
		}
		base := filepath.Base(f.Name)
		outPath := filepath.Join(destDir, "icons", base)
		if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
			return err
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		data, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			return err
		}
		if err := os.WriteFile(outPath, data, 0o644); err != nil {
			return err
		}
	}
	return nil
}

func removeGenerated(pattern string) error {
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return err
	}
	for _, m := range matches {
		if err := os.Remove(m); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}
