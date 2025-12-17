package main

import (
	"bufio"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	_ "golang.org/x/image/bmp" // Support BMP decoding
)

func main() {
	fntPath := flag.String("fnt", "", "Path to .fnt file")
	outDir := flag.String("out", "", "Output directory for verification result (optional)")
	flag.Parse()

	if *fntPath == "" {
		log.Fatal("Please provide -fnt path")
	}

	// 1. Parse FNT
	chars, imgFileName, err := parseFNT(*fntPath)
	if err != nil {
		log.Fatalf("Failed to parse FNT: %v", err)
	}

	// 2. Load the Atlas Image
	dir := filepath.Dir(*fntPath)
	imgPath := filepath.Join(dir, imgFileName)

	srcImg, err := loadImg(imgPath)
	if err != nil {
		log.Fatalf("Failed to load image %s: %v", imgPath, err)
	}

	// 3. Create a canvas to draw on
	b := srcImg.Bounds()
	dstImg := image.NewRGBA(b)
	draw.Draw(dstImg, b, srcImg, image.Point{}, draw.Src)

	// 4. Draw Red Boxes
	red := color.RGBA{255, 0, 0, 255}
	for _, c := range chars {
		drawRect(dstImg, c.X, c.Y, c.W, c.H, red)
	}

	// 5. Determine Output Path
	targetDir := dir
	if *outDir != "" {
		targetDir = *outDir
		if err := os.MkdirAll(targetDir, 0755); err != nil {
			log.Fatalf("Failed to create output dir: %v", err)
		}
	}
	outPath := filepath.Join(targetDir, "verification_result.png")

	// 6. Save Result
	if err := saveImg(outPath, dstImg); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Verification complete. Check %s\n", outPath)
}

// --- Helpers ---

func loadImg(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	srcImg, _, err := image.Decode(f)
	return srcImg, err
}

func saveImg(path string, img image.Image) (err error) {
	outF, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := outF.Close(); err == nil {
			err = cerr
		}
	}()
	return png.Encode(outF, img)
}

func drawRect(img *image.RGBA, x, y, w, h int, c color.RGBA) {
	for i := x; i < x+w; i++ {
		img.Set(i, y, c)
		img.Set(i, y+h-1, c)
	}
	for j := y; j < y+h; j++ {
		img.Set(x, j, c)
		img.Set(x+w-1, j, c)
	}
}

// --- FNT Parsing ---

type CharDef struct {
	ID, X, Y, W, H int
}

func parseFNT(path string) (map[int]CharDef, string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, "", err
	}
	// FIX: Explicitly ignore error on defer Close
	defer func() { _ = f.Close() }()

	chars := make(map[int]CharDef)
	var imgFile string

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "page") {
			fields := parseLine(line)
			if val, ok := fields["file"]; ok {
				imgFile = val
			}
		} else if strings.HasPrefix(line, "char ") {
			d := parseLineInts(line)
			id := d["id"]
			chars[id] = CharDef{ID: id, X: d["x"], Y: d["y"], W: d["width"], H: d["height"]}
		}
	}
	imgFile = strings.Trim(imgFile, "\"")
	if imgFile == "" {
		return nil, "", fmt.Errorf("no 'file' attribute found in page tag")
	}
	return chars, imgFile, scanner.Err()
}

func parseLine(line string) map[string]string {
	m := make(map[string]string)
	parts := strings.Split(line, " ")
	for _, p := range parts {
		kv := strings.SplitN(p, "=", 2)
		if len(kv) == 2 {
			m[kv[0]] = kv[1]
		}
	}
	return m
}

func parseLineInts(line string) map[string]int {
	m := make(map[string]int)
	parts := strings.Split(line, " ")
	for _, p := range parts {
		kv := strings.SplitN(p, "=", 2)
		if len(kv) == 2 {
			val, _ := strconv.Atoi(kv[1])
			m[kv[0]] = val
		}
	}
	return m
}
