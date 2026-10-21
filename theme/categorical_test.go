package theme

import (
	"testing"
)

// TestCategoricalHuesStayDistinct guards against contrast fitting pulling two
// hues onto the same color (yellow and orange both going brown on a light
// surface, say).
func TestCategoricalHuesStayDistinct(t *testing.T) {
	const minDistance = 0.08
	for _, name := range Names() {
		th, _ := Get(name)
		worst, wi, wj := 1.0, 0, 0
		for i := range NumHues {
			for j := i + 1; j < NumHues; j++ {
				if d := Distance(th.Categorical[i], th.Categorical[j]); d < worst {
					worst, wi, wj = d, i, j
				}
			}
		}
		t.Logf("%-20s closest pair %d/%d at %.3f", name, wi, wj, worst)
		if worst < minDistance {
			t.Errorf("%s: Categorical[%d] and [%d] are only %.3f apart", name, wi, wj, worst)
		}
	}
}

func TestCategoricalKeepsExplicitValues(t *testing.T) {
	th := yogaDark()
	want := rgb(1, 2, 3)
	th.Categorical[HueBlue] = want
	normalize(&th)
	if th.Categorical[HueBlue] != want {
		t.Fatal("normalize overwrote an explicit categorical color")
	}
	if colorUnset(th.CategoricalSurface[HueBlue]) || colorUnset(th.Categorical[HueRed]) {
		t.Fatal("unset categorical entries were not derived")
	}
}
