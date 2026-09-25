package converter

import (
	"bufio"
	"image/png"
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
	filled, hollow, _, _ := generatePair(t, 100)
	for i := range filled {
		if filled[i] != hollow[i] {
			t.Fatalf("pixel %d: a stroke wider than any stem should change nothing, got %d, want %d", i, hollow[i], filled[i])
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
