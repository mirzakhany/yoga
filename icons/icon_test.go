package icons

import "testing"

func TestSearchIcon(t *testing.T) {
	if Search.Empty() {
		t.Fatal("Search icon missing")
	}
	if _, err := Search.Alpha(BakePx); err != nil {
		t.Fatal(err)
	}
}

func TestYogaIcon(t *testing.T) {
	if Yoga.Empty() {
		t.Fatal("Yoga icon missing")
	}
}
