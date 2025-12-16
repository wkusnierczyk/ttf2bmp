package main

import (
	"bufio"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"strconv"
	"strings"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"

	_ "golang.org/x/image/bmp"
)

func main() {
	ttfPath := flag.String("ttf", "", "Original TTF")
	fntPath := flag.String("fnt", "", "Generated FNT")
	chars := flag.String("chars", "", "Chars to test")
	size := flag.Float64("size", 12, "Size")
	tolerance := flag.Float64("tol", 0.05, "Tolerance (0.05 = 5%)")
	outDir := flag.String("out", ".", "Output dir")

	flag.Usage = func() {
		_, _ = fmt.Fprintf(flag.CommandLine.Output(), "Usage of %s:\n", "validator")
		flag.PrintDefaults()
	}
	flag.Parse()

	if *ttfPath == "" || *fntPath == "" {
		fmt.Println("Missing arguments: -ttf and -fnt required")
		os.Exit(1)
	}

	bmpPath := strings.TrimSuffix(*fntPath, ".fnt") + ".bmp"

	// 1. Render Reference (TTF)
	// We pass 'nil' for image to just measure metrics first?
	// Actually, renderTTF now handles metrics internally to be safe.
	refImg, err := renderTTF(*ttfPath, *chars, *size)
	if err != nil {
		panic(err)
	}

	// 2. Render Candidate (FNT)
	canImg, err := renderFNT(*fntPath, bmpPath, *chars)
	if err != nil {
		panic(err)
	}

	// 3. Compare
	diffImg, score := compare(refImg, canImg)

	// Save debug images
	_ = savePNG(refImg, *outDir+"/debug_ref.png")
	_ = savePNG(canImg, *outDir+"/debug_can.png")
	_ = savePNG(diffImg, *outDir+"/debug_diff.png")

	fmt.Printf("Diff Score: %.4f%%\n", score*100)
	if score > *tolerance {
		fmt.Println("FAIL")
		os.Exit(1)
	}
	fmt.Println("PASS")
}

func renderTTF(path, text string, s float64) (image.Image, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	f, err := opentype.Parse(b)
	if err != nil {
		return nil, err
	}

	face, err := opentype.NewFace(f, &opentype.FaceOptions{
		Size:    s,
		DPI:     72,
		Hinting: font.HintingFull,
	})
	if err != nil {
		return nil, err
	}
	defer func() { _ = face.Close() }()

	metrics := face.Metrics()
	ascent := metrics.Ascent.Ceil()
	height := metrics.Height.Ceil()

	// Create canvas. We add padding to ensure no clipping during validation.
	// Width = approx char count * size (plenty of room)
	// Height = Line Height
	img := image.NewRGBA(image.Rect(0, 0, len(text)*int(s+5), height))

	// Start drawing at X=10 to allow some padding
	// Y = Ascent (Baseline)
	dot := fixed.P(10, ascent)

	d := &font.Drawer{
		Dst:  img,
		Src:  image.White,
		Face: face,
		Dot:  dot,
	}
	d.DrawString(text)
	return img, nil
}

func renderFNT(fnt, bmp, text string) (image.Image, error) {
	charMap, common, err := parseFNT(fnt)
	if err != nil {
		return nil, err
	}

	f, err := os.Open(bmp)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()

	atlas, _, err := image.Decode(f)
	if err != nil {
		return nil, err
	}

	// Canvas setup must match renderTTF
	// The FNT 'common.Base' tells us where the baseline is relative to the top of the line.
	// We want to draw at the same visual position as renderTTF.

	// In renderTTF:
	//   Canvas Height = LineHeight (from metrics)
	//   Baseline Y    = Ascent (from metrics)

	// In FNT, we have common.LineHeight and common.Base.
	// We should setup the image using these.

	// Note: We use an arbitrarily wide width like renderTTF
	img := image.NewRGBA(image.Rect(0, 0, len(text)*50, common.LineHeight))

	// We want the cursor's baseline to be at 'common.Base'.
	// In FNT, characters are drawn relative to the cursor position.
	// If the FNT generator did its job, the glyphs in the texture have (yoffset)
	// that puts them in the right place relative to the cursor.

	cursorX := 10 // Match renderTTF X padding
	cursorY := 0  // Top of the line

	for _, r := range text {
		if c, ok := charMap[r]; ok {
			// Destination Rect:
			// X = CursorX + XOffset
			// Y = CursorY + YOffset

			// Wait, in our specific generator, we captured the full strip height.
			// So YOffset is 0, and the image block is the full line height.
			// So we just place it at the top of the line (cursorY).

			destRect := image.Rect(
				cursorX+c.XOffset,
				cursorY+c.YOffset,
				cursorX+c.XOffset+c.W,
				cursorY+c.YOffset+c.H,
			)

			draw.Draw(img,
				destRect,
				atlas, image.Point{c.X, c.Y}, draw.Over,
			)
			cursorX += c.XAdvance
		}
	}
	return img, nil
}

func compare(img1, img2 image.Image) (image.Image, float64) {
	b := img1.Bounds()
	diffImg := image.NewRGBA(b)
	var totalDiff uint64
	var pixelCount uint64

	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			r1, _, _, _ := img1.At(x, y).RGBA()
			r2, _, _, _ := img2.At(x, y).RGBA()
			v1, v2 := uint8(r1>>8), uint8(r2>>8)

			d := 0
			if v1 > v2 {
				d = int(v1 - v2)
			} else {
				d = int(v2 - v1)
			}

			// Increased tolerance threshold slightly to ignore anti-aliasing drift
			if d > 30 {
				totalDiff += uint64(d)
				diffImg.Set(x, y, color.RGBA{255, 0, 0, 255})
			} else {
				diffImg.Set(x, y, color.RGBA{v1, v1, v1, 100})
			}
			pixelCount++
		}
	}
	if pixelCount == 0 {
		return diffImg, 0.0
	}
	return diffImg, float64(totalDiff) / float64(pixelCount*255)
}

// Updated data structures to capture 'common' block
type CharDef struct{ ID, X, Y, W, H, XOffset, YOffset, XAdvance int }
type CommonDef struct{ LineHeight, Base int }

func parseFNT(path string) (map[rune]CharDef, CommonDef, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, CommonDef{}, err
	}
	defer func() { _ = f.Close() }()

	chars := make(map[rune]CharDef)
	var common CommonDef

	s := bufio.NewScanner(f)
	for s.Scan() {
		l := s.Text()

		if strings.HasPrefix(l, "common ") {
			d := parseLine(l)
			common.LineHeight = d["lineHeight"]
			common.Base = d["base"]
		} else if strings.HasPrefix(l, "char ") {
			d := parseLine(l)
			chars[rune(d["id"])] = CharDef{d["id"], d["x"], d["y"], d["width"], d["height"], d["xoffset"], d["yoffset"], d["xadvance"]}
		}
	}
	return chars, common, nil
}

func parseLine(l string) map[string]int {
	d := make(map[string]int)
	for _, fd := range strings.Fields(l) {
		p := strings.Split(fd, "=")
		if len(p) == 2 {
			v, _ := strconv.Atoi(p[1])
			d[p[0]] = v
		}
	}
	return d
}

func savePNG(i image.Image, p string) (err error) {
	f, err := os.Create(p)
	if err != nil {
		return err
	}
	defer func() {
		cerr := f.Close()
		if err == nil {
			err = cerr
		}
	}()
	return png.Encode(f, i)
}
