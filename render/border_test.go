package render

import "testing"

func TestAddRoundedRectStrokeEmitsGeometry(t *testing.T) {
	var dl DrawList
	r := Rect{X: 10, Y: 10, W: 100, H: 40}
	corners := UniformCorners(4)
	widths := BorderEdges{Top: 1}
	dl.AddRoundedRectStroke(r, corners, widths, RGBA8(255, 0, 0, 255), BorderDotted)
	if len(dl.Vertices) == 0 {
		t.Fatal("expected dotted top border vertices")
	}

	dl.Reset()
	widths = BorderEdges{Left: 2}
	dl.AddRoundedRectStroke(r, corners, widths, RGBA8(0, 255, 0, 255), BorderDashed)
	if len(dl.Vertices) == 0 {
		t.Fatal("expected dashed left border vertices")
	}
}

func TestAddRoundedRectCornersIndependent(t *testing.T) {
	var dl DrawList
	r := Rect{X: 0, Y: 0, W: 50, H: 50}
	corners := Corners{TopLeft: 8, TopRight: 0, BottomRight: 0, BottomLeft: 0}
	dl.AddRoundedRectCorners(r, corners, RGBA8(0, 0, 255, 255))
	if len(dl.Vertices) == 0 {
		t.Fatal("expected fill vertices")
	}
}

func TestPaintBoxNoPanicTinyRect(t *testing.T) {
	var dl DrawList
	dl.PaintBox(Rect{X: 0, Y: 0, W: 1, H: 1}, UniformCorners(4), UniformBorder(1),
		RGBA8(10, 10, 10, 255), RGBA8(20, 20, 20, 255), BorderSolid)
	dl.PaintBox(Rect{}, UniformCorners(0), BorderEdges{}, Color{}, Color{}, BorderDotted)
}

func TestAddRoundedRectCornersSingleCorner(t *testing.T) {
	var dl DrawList
	r := Rect{X: 0, Y: 0, W: 100, H: 60}
	tr := float32(8)
	corners := Corners{TopRight: tr}
	dl.AddRoundedRectCorners(r, corners, RGBA8(255, 255, 255, 255))

	maxY := float32(0)
	for _, v := range dl.Vertices {
		if v.Pos[1] > maxY {
			maxY = v.Pos[1]
		}
	}
	if maxY > r.Y+r.H+0.01 {
		t.Fatalf("geometry extends below rect: maxY=%v rectBottom=%v", maxY, r.Y+r.H)
	}
}

func TestPaintBoxSingleTopRightCorner(t *testing.T) {
	var dl DrawList
	r := Rect{X: 10, Y: 10, W: 180, H: 80}
	corners := Corners{TopRight: 8}
	widths := UniformBorder(1)
	dl.PaintBox(r, corners, widths, RGBA8(30, 30, 30, 255), RGBA8(200, 200, 200, 255), BorderSolid)

	maxY := float32(0)
	for _, v := range dl.Vertices {
		if v.Pos[1] > maxY {
			maxY = v.Pos[1]
		}
	}
	if maxY > r.Y+r.H+0.01 {
		t.Fatalf("border extends below rect: maxY=%v rectBottom=%v", maxY, r.Y+r.H)
	}
}

func TestUniformBorderDetection(t *testing.T) {
	if _, ok := uniformBorder(UniformBorder(2)); !ok {
		t.Fatal("expected uniform border")
	}
	if _, ok := uniformBorder(BorderEdges{Top: 2, Bottom: 2}); ok {
		t.Fatal("partial border should not be uniform full box")
	}
}
