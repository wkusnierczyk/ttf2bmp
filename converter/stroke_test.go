package converter

import (
	"bufio"
	"fmt"
	"image/png"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"testing"
)

// bruteDistance is the definition insideDistance must match: the squared distance
// from each inside pixel to the nearest outside one, the area beyond the image
// counting as outside.
func bruteDistance(inside []bool, w, h int) []float64 {
	out := make([]float64, w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if !inside[y*w+x] {
				continue
			}
			best := -1
			for oy := -1; oy <= h; oy++ {
				for ox := -1; ox <= w; ox++ {
					if ox >= 0 && oy >= 0 && ox < w && oy < h && inside[oy*w+ox] {
						continue
					}
					d := (ox-x)*(ox-x) + (oy-y)*(oy-y)
					if best < 0 || d < best {
						best = d
					}
				}
			}
			out[y*w+x] = float64(best)
		}
	}
	return out
}

func TestInsideDistanceMatchesBruteForce(t *testing.T) {
	r := rand.New(rand.NewSource(1))
	for trial := 0; trial < 50; trial++ {
		w, h := 1+r.Intn(20), 1+r.Intn(20)
		inside := make([]bool, w*h)
		for i := range inside {
			inside[i] = r.Intn(4) != 0
		}
		got, want := insideDistance(inside, w, h), bruteDistance(inside, w, h)
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("%dx%d mask, pixel %d: got %v, want %v", w, h, i, got[i], want[i])
			}
		}
	}
}

func TestInsideDistanceFullSquare(t *testing.T) {
	inside := make([]bool, 25)
	for i := range inside {
		inside[i] = true
	}
	// The centre of a 5x5 square is 3 pixels from the outside beyond each edge.
	if got := insideDistance(inside, 5, 5)[12]; got != 9 {
		t.Errorf("centre: got %v, want 9", got)
	}
}

// generatePair runs Generate filled and hollow on the test font and returns the two
// decoded alpha channels and the two fnt files.
func generatePair(t *testing.T, stroke float64) (filled, hollow []uint8, filledFnt, hollowFnt []string) {
	t.Helper()
	wd, _ := os.Getwd()
	fontPath := filepath.Join(filepath.Dir(wd), "test_data", "Go-Regular.ttf")
	if _, err := os.Stat(fontPath); os.IsNotExist(err) {
		t.Skipf("Test font not found at %s. Run 'make fetch-test-data' first.", fontPath)
	}
	dir := t.TempDir()
	a, b := filepath.Join(dir, "filled"), filepath.Join(dir, "hollow")
	if err := Generate(fontPath, 48, "ABCgo8", a, "png", 2, "none", 0); err != nil {
		t.Fatal(err)
	}
	if err := Generate(fontPath, 48, "ABCgo8", b, "png", 2, "none", stroke); err != nil {
		t.Fatal(err)
	}
	return alpha(t, a+".png"), alpha(t, b+".png"), lines(t, a+".fnt"), lines(t, b+".fnt")
}

func alpha(t *testing.T, path string) []uint8 {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	img, err := png.Decode(f)
	if err != nil {
		t.Fatal(err)
	}
	b := img.Bounds()
	out := make([]uint8, 0, b.Dx()*b.Dy())
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			_, _, _, a := img.At(x, y).RGBA()
			out = append(out, uint8(a>>8))
		}
	}
	return out
}

func lines(t *testing.T, path string) []string {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	var out []string
	s := bufio.NewScanner(f)
	for s.Scan() {
		out = append(out, s.Text())
	}
	return out
}

func TestStrokeKeepsMetrics(t *testing.T) {
	_, _, filled, hollow := generatePair(t, 1.5)
	if len(filled) != len(hollow) {
		t.Fatalf("fnt line counts differ: %d vs %d", len(filled), len(hollow))
	}
	for i := range filled {
		// Line 3 is the page line, which names the image file.
		if i != 2 && filled[i] != hollow[i] {
			t.Errorf("fnt line %d differs:\n  filled: %s\n  hollow: %s", i+1, filled[i], hollow[i])
		}
	}
}

func TestStrokeStaysInsideTheGlyph(t *testing.T) {
	filled, hollow, _, _ := generatePair(t, 1.5)
	var sumFilled, sumHollow int
	for i := range filled {
		if hollow[i] > filled[i] {
			t.Fatalf("pixel %d: hollow alpha %d exceeds filled %d", i, hollow[i], filled[i])
		}
		sumFilled += int(filled[i])
		sumHollow += int(hollow[i])
	}
	if sumHollow == 0 || sumHollow >= sumFilled {
		t.Errorf("a 1.5px stroke should keep some of the ink, but not all of it: %d of %d", sumHollow, sumFilled)
	}
}

func TestWideStrokeIsFilled(t *testing.T) {
	// No point of these 48px glyphs is 5px inside its edge, so a 5px band is the whole
	// glyph. Small enough that the distance transform decides it, unlike a stroke wider
	// than the atlas.
	filled, hollow, _, _ := generatePair(t, 5)
	for i := range filled {
		if filled[i] != hollow[i] {
			t.Fatalf("pixel %d: a stroke wider than any stem should change nothing, got %d, want %d", i, hollow[i], filled[i])
		}
	}
}

// stemRow renders one glyph at 100px and returns the alpha of the row halfway down
// its ink, filled and hollow.
func stemRow(t *testing.T, char string, hinting string, stroke float64) (filled, hollow []uint8) {
	t.Helper()
	wd, _ := os.Getwd()
	fontPath := filepath.Join(filepath.Dir(wd), "test_data", "Go-Regular.ttf")
	if _, err := os.Stat(fontPath); os.IsNotExist(err) {
		t.Skipf("Test font not found at %s. Run 'make fetch-test-data' first.", fontPath)
	}
	dir := t.TempDir()
	a, b := filepath.Join(dir, "filled"), filepath.Join(dir, "hollow")
	if err := Generate(fontPath, 100, char, a, "png", 2, hinting, 0); err != nil {
		t.Fatal(err)
	}
	if err := Generate(fontPath, 100, char, b, "png", 2, hinting, stroke); err != nil {
		t.Fatal(err)
	}
	fa, fb := alpha(t, a+".png"), alpha(t, b+".png")
	var lineHeight int
	if _, err := fmt.Sscanf(lines(t, a+".fnt")[1], "common lineHeight=%d", &lineHeight); err != nil {
		t.Fatal(err)
	}
	width := len(fa) / lineHeight
	top, bottom := -1, -1
	for y := 0; y < len(fa)/width; y++ {
		for x := 0; x < width; x++ {
			if fa[y*width+x] > 0 {
				if top < 0 {
					top = y
				}
				bottom = y
				break
			}
		}
	}
	y := (top + bottom) / 2
	return fa[y*width : (y+1)*width], fb[y*width : (y+1)*width]
}

// The band is measured across the stem of an I: each side should hold W pixels of
// ink, the two sides should agree, and the middle of the stem should be empty. Totals
// over the whole atlas cannot see a band that is shifted or the wrong width; this can.
func TestStrokeWidthOnAStem(t *testing.T) {
	for _, hinting := range []string{"none", "full"} {
		for _, w := range []float64{1, 1.5, 2.25} {
			filled, hollow := stemRow(t, "I", hinting, w)
			left, right := -1, -1
			for x, a := range filled {
				if a > 0 {
					if left < 0 {
						left = x
					}
					right = x
				}
			}
			mid := (left + right) / 2
			var inkLeft, inkRight float64
			for x := left; x <= right; x++ {
				ink := float64(hollow[x]) / 255
				if x <= mid {
					inkLeft += ink
				} else {
					inkRight += ink
				}
			}
			// Measured: 0.03 to 0.05px over W on each side, from where the rasterizer's
			// antialiased outer edge falls. Tight enough to catch the inner edge moving by
			// half a supersample (1/16px), which lands 0.08px under W.
			const tolerance = 0.07
			if math.Abs(inkLeft-w) > tolerance || math.Abs(inkRight-w) > tolerance {
				t.Errorf("hinting %s, stroke %v: ink per side %.3f and %.3f, want %v each", hinting, w, inkLeft, inkRight, w)
			}
			if hollow[mid] != 0 {
				t.Errorf("hinting %s, stroke %v: the middle of a %dpx stem should be empty, alpha %d", hinting, w, right-left+1, hollow[mid])
			}
		}
	}
}

func TestStrokeGrowsWithWidth(t *testing.T) {
	sum := func(w float64) int {
		_, h, _, _ := generatePair(t, w)
		s := 0
		for _, a := range h {
			s += int(a)
		}
		return s
	}
	prev := 0
	for _, w := range []float64{0.5, 0.77, 1, 1.25, 2} {
		s := sum(w)
		if s <= prev {
			t.Errorf("stroke %v keeps %d ink, not more than the narrower stroke's %d", w, s, prev)
		}
		prev = s
	}
}

// fontOrSkip returns the test font's path, or skips the test when it is missing.
func fontOrSkip(t *testing.T) string {
	t.Helper()
	wd, _ := os.Getwd()
	fontPath := filepath.Join(filepath.Dir(wd), "test_data", "Go-Regular.ttf")
	if _, err := os.Stat(fontPath); os.IsNotExist(err) {
		t.Skipf("Test font not found at %s. Run 'make fetch-test-data' first.", fontPath)
	}
	return fontPath
}

// cells renders chars and returns the atlas alpha split into one slice of rows per
// fnt char line, in the order the fnt lists them.
func cells(t *testing.T, chars string, padding int, stroke float64) [][]uint8 {
	t.Helper()
	out := filepath.Join(t.TempDir(), "atlas")
	if err := Generate(fontOrSkip(t), 100, chars, out, "png", padding, "none", stroke); err != nil {
		t.Fatal(err)
	}
	a := alpha(t, out+".png")
	fnt := lines(t, out+".fnt")
	var lineHeight, scaleW int
	if _, err := fmt.Sscanf(fnt[1], "common lineHeight=%d base=%d scaleW=%d", &lineHeight, new(int), &scaleW); err != nil {
		t.Fatal(err)
	}
	var result [][]uint8
	for _, l := range fnt[4:] {
		var id, x, width int
		if _, err := fmt.Sscanf(l, "char id=%d x=%d y=0 width=%d", &id, &x, &width); err != nil {
			t.Fatal(err)
		}
		var cell []uint8
		for y := 0; y < lineHeight; y++ {
			cell = append(cell, a[y*scaleW+x:y*scaleW+x+width]...)
		}
		result = append(result, cell)
	}
	return result
}

func TestMinStrokeKeepsTheGlyph(t *testing.T) {
	filled, hollow := stemRow(t, "I", "none", MinStroke)
	var ink, full float64
	for x := range filled {
		ink += float64(hollow[x]) / 255
		full += float64(filled[x]) / 255
	}
	if ink == 0 || ink >= full {
		t.Errorf("the narrowest stroke should keep a thin band: %.3f of %.3f", ink, full)
	}
	err := Generate(fontOrSkip(t), 32, "A", filepath.Join(t.TempDir(), "x"), "png", 2, "none", MinStroke/2)
	if err == nil {
		t.Error("a stroke below MinStroke would erase the glyph; Generate should refuse it")
	}
}

// A character given twice is drawn twice; the fnt points at the second copy, but
// both are in the image and both must be hollow.
func TestDuplicateCharsAreAllHollow(t *testing.T) {
	filled := filepath.Join(t.TempDir(), "filled")
	if err := Generate(fontOrSkip(t), 100, "II", filled, "png", 2, "none", 0); err != nil {
		t.Fatal(err)
	}
	hollow := filepath.Join(t.TempDir(), "hollow")
	if err := Generate(fontOrSkip(t), 100, "II", hollow, "png", 2, "none", 1.5); err != nil {
		t.Fatal(err)
	}
	f, h := alpha(t, filled+".png"), alpha(t, hollow+".png")
	var inkF, inkH int
	for i := range f {
		inkF += int(f[i])
		inkH += int(h[i])
	}
	// Two identical I's: if only one were hollowed, more than half the ink would remain.
	one := cells(t, "I", 2, 1.5)[0]
	inkOne := 0
	for _, a := range one {
		inkOne += int(a)
	}
	if inkH != 2*inkOne {
		t.Errorf("two hollow I's hold %d ink, want twice one hollow I, %d (filled: %d)", inkH, 2*inkOne, inkF)
	}
}

// With no padding, neighbouring glyphs can touch. Each must still be measured on its
// own, so the outline runs round each glyph, not round the pair as one shape.
func TestTouchingGlyphsKeepTheirOwnOutline(t *testing.T) {
	pair := cells(t, "__", 0, 1.5)
	single := cells(t, "_", 0, 1.5)
	for i, cell := range pair {
		for j := range cell {
			if cell[j] != single[0][j] {
				t.Fatalf("underscore %d of 2, pixel %d: alpha %d, want %d as for an underscore on its own", i+1, j, cell[j], single[0][j])
			}
		}
	}
}
