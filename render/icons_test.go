package render

import "testing"

func TestIconsRasterize(t *testing.T) {
	if _, ok := iconRegistry["yoga"]; !ok {
		t.Fatal("yoga icon missing from registry")
	}
	for _, name := range IconNames() {
		mask, err := rasterizeIcon(iconRegistry[name], 24)
		if err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}
		nonzero := 0
		for _, p := range mask.Pix {
			if p > 0 {
				nonzero++
			}
		}
		if nonzero == 0 {
			t.Errorf("%s: empty mask", name)
		}
	}
}
