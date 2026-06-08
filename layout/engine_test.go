package layout

import (
	"math"
	"testing"
)

func layoutRoot(root *Element, w, h float32) {
	root.Calculate(w, h)
}

func approx(a, b float32) bool {
	return math.Abs(float64(a-b)) < 0.5
}

func TestFlexColumnGrow(t *testing.T) {
	fixed := New(Box().H(40))
	grow := New(Box().FlexGrow(1))
	root := New(Box(), fixed, grow)
	layoutRoot(root, 200, 300)

	if !approx(fixed.Frame.H, 40) {
		t.Fatalf("fixed height: got %v want 40", fixed.Frame.H)
	}
	if !approx(grow.Frame.Y, 40) {
		t.Fatalf("grow y: got %v want 40", grow.Frame.Y)
	}
	if !approx(grow.Frame.H, 260) {
		t.Fatalf("grow height: got %v want 260", grow.Frame.H)
	}
}

func TestFlexRowGrowSplit(t *testing.T) {
	a := New(Box().W(100).FlexShrink(0))
	b := New(Box().FlexGrow(1))
	root := New(Box().Direction(Row), a, b)
	layoutRoot(root, 500, 100)

	if !approx(a.Frame.W, 100) {
		t.Fatalf("a width: got %v want 100", a.Frame.W)
	}
	if !approx(b.Frame.X, 100) {
		t.Fatalf("b x: got %v want 100", b.Frame.X)
	}
	if !approx(b.Frame.W, 400) {
		t.Fatalf("b width: got %v want 400", b.Frame.W)
	}
}

func TestFlexJustifyCenter(t *testing.T) {
	a := New(Box().W(50).H(20))
	b := New(Box().W(50).H(20))
	root := New(
		Box().Direction(Row).H(40).JustifyContent(JustifyCenter),
		a, b,
	)
	layoutRoot(root, 200, 40)

	if !approx(a.Frame.X, 50) {
		t.Fatalf("a x: got %v want 50", a.Frame.X)
	}
	if !approx(b.Frame.X, 100) {
		t.Fatalf("b x: got %v want 100", b.Frame.X)
	}
}

func TestFlexAlignCenter(t *testing.T) {
	child := New(Box().W(40).H(10))
	root := New(
		Box().Direction(Row).H(40).AlignItems(AlignCenter),
		child,
	)
	layoutRoot(root, 100, 40)

	if !approx(child.Frame.Y, 15) {
		t.Fatalf("child y: got %v want 15", child.Frame.Y)
	}
}

func TestFlexGap(t *testing.T) {
	a := New(Box().W(10).H(10))
	b := New(Box().W(10).H(10))
	root := New(
		Box().Direction(Row).Gap(20),
		a, b,
	)
	layoutRoot(root, 100, 20)

	if !approx(b.Frame.X, 30) {
		t.Fatalf("b x with gap: got %v want 30", b.Frame.X)
	}
}

func TestFlexPaddingOffset(t *testing.T) {
	child := New(Box().W(10).H(10))
	root := New(Box().PaddingAll(8), child)
	layoutRoot(root, 100, 100)

	if !approx(child.Frame.X, 8) || !approx(child.Frame.Y, 8) {
		t.Fatalf("child frame: got (%v,%v) want (8,8)", child.Frame.X, child.Frame.Y)
	}
}

func TestAbsolutePinStretch(t *testing.T) {
	host := New(Box().FlexGrow(1))
	bar := New(Box().W(12).AbsTop(0).AbsRight(0).AbsBottom(0))
	host.Add(bar)
	root := New(Box().Size(200, 100), host)
	layoutRoot(root, 200, 100)

	if !approx(bar.Frame.X, 188) {
		t.Fatalf("bar x: got %v want 188", bar.Frame.X)
	}
	if !approx(bar.Frame.W, 12) {
		t.Fatalf("bar w: got %v want 12", bar.Frame.W)
	}
	if !approx(bar.Frame.H, 100) {
		t.Fatalf("bar h: got %v want 100", bar.Frame.H)
	}
}

func TestGridFixed2x2(t *testing.T) {
	a := New(Box())
	b := New(Box())
	c := New(Box())
	d := New(Box())
	root := New(
		Box().Display(DisplayGrid).GridCols(Px(50), Px(50)).GridRows(Px(30), Px(30)).Gap(10),
		a, b, c, d,
	)
	layoutRoot(root, 110, 70)

	if !approx(a.Frame.W, 50) || !approx(a.Frame.H, 30) {
		t.Fatalf("a: got %vx%v", a.Frame.W, a.Frame.H)
	}
	if !approx(b.Frame.X, 60) {
		t.Fatalf("b x: got %v want 60", b.Frame.X)
	}
	if !approx(c.Frame.Y, 40) {
		t.Fatalf("c y: got %v want 40", c.Frame.Y)
	}
}

func TestGridFrTracks(t *testing.T) {
	a := New(Box())
	b := New(Box())
	root := New(
		Box().Display(DisplayGrid).GridCols(Px(50), Fr(1)).Gap(10),
		a, b,
	)
	layoutRoot(root, 160, 40)

	if !approx(a.Frame.W, 50) {
		t.Fatalf("a w: got %v want 50", a.Frame.W)
	}
	if !approx(b.Frame.W, 100) {
		t.Fatalf("b w: got %v want 100", b.Frame.W)
	}
}

func TestGridSpan(t *testing.T) {
	wide := New(Box().Col(1, 2))
	root := New(
		Box().Display(DisplayGrid).GridCols(Px(40), Px(40)).GridRows(Px(20)),
		wide,
	)
	layoutRoot(root, 80, 20)

	if !approx(wide.Frame.W, 80) {
		t.Fatalf("span width: got %v want 80", wide.Frame.W)
	}
}

func TestStackCenter(t *testing.T) {
	child := New(Box().Size(20, 10))
	root := New(
		Box().Display(DisplayStack).Size(100, 50).StackPosition(JustifyCenter, AlignCenter),
		child,
	)
	layoutRoot(root, 100, 50)

	if !approx(child.Frame.X, 40) || !approx(child.Frame.Y, 20) {
		t.Fatalf("centered: got (%v,%v) want (40,20)", child.Frame.X, child.Frame.Y)
	}
}

func TestStackTopLeading(t *testing.T) {
	child := New(Box().Size(20, 10))
	root := New(
		Box().Display(DisplayStack).Size(100, 50).StackPosition(JustifyStart, AlignStart),
		child,
	)
	layoutRoot(root, 100, 50)

	if !approx(child.Frame.X, 0) || !approx(child.Frame.Y, 0) {
		t.Fatalf("top-leading: got (%v,%v)", child.Frame.X, child.Frame.Y)
	}
}

func TestStackAbsoluteChild(t *testing.T) {
	host := New(Box().Display(DisplayStack).Size(100, 50))
	overlay := New(Box().Size(10, 10).Absolute(5, 5))
	host.Add(overlay)
	root := New(Box(), host)
	layoutRoot(root, 100, 50)

	if !approx(overlay.Frame.X, 5) || !approx(overlay.Frame.Y, 5) {
		t.Fatalf("absolute in stack: got (%v,%v)", overlay.Frame.X, overlay.Frame.Y)
	}
}
