package main

import (
	"bufio"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/golang/freetype/truetype"
	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"

	_ "golang.org/x/image/bmp"
)

func main() {
	ttfPath := flag.String("ttf", "", "Original TTF")
	fntPath := flag.String("fnt", "", "Generated FNT")
	chars := flag.String("chars", "", "Chars to test")
	size := flag.Float64("size", 12, "Size")
	flag.Parse()

	if *ttfPath == "" || *fntPath == "" {
		fmt.Println("Missing arguments")
		os.Exit(1)
	}

	bmpPath := strings.TrimSuffix(*fntPath, ".fnt") + ".bmp"

	// Render Both
	refImg, _ := renderTTF(*ttfPath, *chars, *size)
	canImg, err := renderFNT(*fntPath, bmpPath, *chars)
	if err != nil {
		fmt.Println("Error rendering FNT:", err)
		os.Exit(1)
	}

	// Compare
	diffImg, score := compare(refImg, canImg)

	// Save Debug
	os.MkdirAll("debug_test", 0755)
	savePNG(refImg, "debug_test/ref.png")
	savePNG(canImg, "debug_test/can.png")
	savePNG(diffImg, "debug_test/diff.png")

	fmt.Printf("Diff Score: %.4f%%\n", score*100)
	if score > 0.05 { // 5% tolerance
		fmt.Println("FAIL")
		os.Exit(1)
	}
	fmt.Println("PASS")
}

func renderTTF(path, text string, s float64) (image.Image, error) {
	b, _ := ioutil.ReadFile(path)
	f, _ := truetype.Parse(b)

	img := image.NewRGBA(image.Rect(0, 0, len(text)*int(s), int(s)*2))
	d := &font.Drawer{
		Dst: img, Src: image.White,
		Face: truetype.NewFace(f, &truetype.Options{Size: s, DPI: 72}),
		Dot:  fixed.P(10, int(s)+10),
	}
	d.DrawString(text)
	return img, nil
}

func renderFNT(fnt, bmp, text string) (image.Image, error) {
	charMap := parseFNT(fnt)
	f, err := os.Open(bmp)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	atlas, _, err := image.Decode(f)
	if err != nil {
		return nil, err
	}

	img := image.NewRGBA(image.Rect(0, 0, len(text)*50, 100))
	cursorX, cursorY := 10, 30 // Approximate baseline match to TTF

	for _, r := range text {
		if c, ok := charMap[r]; ok {
			// Draw
			draw.Draw(img,
				image.Rect(cursorX+c.XOffset, cursorY+c.YOffset, cursorX+c.XOffset+c.W, cursorY+c.YOffset+c.H),
				atlas, image.Point{c.X, c.Y}, draw.Over,
			)
			cursorX += c.XAdvance
		}
	}
	return img, nil
}

func compare(img1, img2 image.Image) (image.Image, float64) {
	b := img1.Bounds() // Simplified: Assuming sizes roughly match or taking img1
	diffImg := image.NewRGBA(b)
	var totalDiff uint64

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

			if d > 20 {
				totalDiff += uint64(d)
				diffImg.Set(x, y, color.RGBA{255, 0, 0, 255})
			} else {
				diffImg.Set(x, y, color.RGBA{v1, v1, v1, 100})
			}
		}
	}
	return diffImg, float64(totalDiff) / float64(b.Dx()*b.Dy()*255)
}

func parseFNT(path string) map[rune]CharDef {
	// Simplified parsing for brevity, same logic as verify_fonts.go
	f, _ := os.Open(path)
	defer f.Close()
	res := make(map[rune]CharDef)
	s := bufio.NewScanner(f)
	for s.Scan() {
		l := s.Text()
		if strings.HasPrefix(l, "char ") {
			d := make(map[string]int)
			for _, fd := range strings.Fields(l) {
				p := strings.Split(fd, "=")
				if len(p) == 2 {
					v, _ := strconv.Atoi(p[1])
					d[p[0]] = v
				}
			}
			res[rune(d["id"])] = CharDef{d["id"], d["x"], d["y"], d["width"], d["height"], d["xoffset"], d["yoffset"], d["xadvance"]}
		}
	}
	return res
}

type CharDef struct{ ID, X, Y, W, H, XOffset, YOffset, XAdvance int }

func savePNG(i image.Image, p string) { f, _ := os.Create(p); png.Encode(f, i); f.Close() }
