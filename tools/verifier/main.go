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

	_ "golang.org/x/image/bmp"
)

func main() {
	fntPath := flag.String("fnt", "", "Path to .fnt file")
	// NEW: Output flag
	outPath := flag.String("out", "", "Output path for verification image (e.g. test_output/result.png)")
	flag.Parse()

	if *fntPath == "" {
		log.Fatal("Please provide -fnt path")
	}

	// Default output if not specified: "verification_result.png" in the same folder
	finalOutPath := *outPath
	if finalOutPath == "" {
		finalOutPath = filepath.Join(filepath.Dir(*fntPath), "verification_result.png")
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

	// 3. Create a canvas
	b := srcImg.Bounds()
	dstImg := image.NewRGBA(b)
	draw.Draw(dstImg, b, srcImg, image.Point{}, draw.Src)

	// 4. Draw Red Boxes
	red := color.RGBA{255, 0, 0, 255}
	for _, c := range chars {
		drawRect(dstImg, c.X, c.Y, c.W, c.H, red)
	}

	// 5. Save Result
	if err := saveImg(finalOutPath, dstImg); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Verification complete. Check %s\n", finalOutPath)
}

// Helper to load image with proper close handling
func loadImg(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = f.Close() // Explicitly ignore error on read-only close
	}()

	srcImg, _, err := image.Decode(f)
	return srcImg, err
}

// Helper to save image with proper close handling
func saveImg(path string, img image.Image) (err error) {
	outF, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() {
		cerr := outF.Close()
		if err == nil { // Only report close error if write succeeded
			err = cerr
		}
	}()

	return png.Encode(outF, img)
}

// Simple helper to draw a hollow rectangle
func drawRect(img *image.RGBA, x, y, w, h int, c color.RGBA) {
	// Top & Bottom
	for i := x; i < x+w; i++ {
		img.Set(i, y, c)
		img.Set(i, y+h-1, c)
	}
	// Left & Right
	for j := y; j < y+h; j++ {
		img.Set(x, j, c)
		img.Set(x+w-1, j, c)
	}
}

// --- FNT Parsing Logic ---

type CharDef struct {
	ID, X, Y, W, H int
}

func parseFNT(path string) (map[int]CharDef, string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, "", err
	}
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
