package converter

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png" // Using PNG for internal temp usage, usually you'd write BMP/FNT
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/golang/freetype/truetype"
	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
	// Import BMP encoder (standard library doesn't support BMP encoding easily,
	// usually you'd use a 3rd party or write a simple header.
	// For this example, I'll save as PNG to ensure it compiles,
	// YOU MUST ADAPT THIS TO YOUR BMP WRITER)
)

// Generate is the public function called by the main CLI.
func Generate(fontPath string, size int, chars string, outPrefix string) error {
	// 1. Read the Font File
	fontBytes, err := ioutil.ReadFile(fontPath)
	if err != nil {
		return fmt.Errorf("reading font file: %w", err)
	}
	f, err := truetype.Parse(fontBytes)
	if err != nil {
		return fmt.Errorf("parsing truetype: %w", err)
	}

	// 2. Setup Font Face
	face := truetype.NewFace(f, &truetype.Options{
		Size:    float64(size),
		DPI:     72,
		Hinting: font.HintingFull,
	})

	// 3. Measure Characters to determine Atlas Size
	// (Simplified logic: just putting them in a row for demonstration)
	var totalWidth, maxHeight int
	for _, char := range chars {
		bounds, advance, ok := face.GlyphBounds(char)
		if !ok {
			continue
		}
		totalWidth += advance.Ceil()
		h := (bounds.Max.Y - bounds.Min.Y).Ceil()
		if h > maxHeight {
			maxHeight = h
		}
	}
	// Add some padding
	maxHeight += size
	if maxHeight == 0 {
		maxHeight = size * 2
	}

	// 4. Draw Characters to Image
	img := image.NewRGBA(image.Rect(0, 0, totalWidth, maxHeight))

	// Draw loop
	dot := fixed.P(0, size) // Start at baseline roughly
	drawer := &font.Drawer{
		Dst:  img,
		Src:  image.White,
		Face: face,
		Dot:  dot,
	}

	drawer.DrawString(chars)

	// 5. Save Image (Here strictly acting as your "BMP" saver)
	// NOTE: Go's standard library encodes PNG/JPEG/GIF.
	// For BMP, you need "golang.org/x/image/bmp" but it only supports Decoding usually.
	// Assuming your original tool had BMP writing logic, insert it here.
	// We will save as .bmp extension but write PNG format just so this code runs.

	bmpFile, err := os.Create(outPrefix + ".bmp")
	if err != nil {
		return err
	}
	defer bmpFile.Close()
	// REPALCE THIS WITH ACTUAL BMP ENCODING
	if err := png.Encode(bmpFile, img); err != nil {
		return err
	}

	// 6. Save FNT Data
	// (Simplified Mock FNT writing)
	fntFile, err := os.Create(outPrefix + ".fnt")
	if err != nil {
		return err
	}
	defer fntFile.Close()

	// Write a minimal valid FNT header for testing
	fmt.Fprintf(fntFile, "info face=\"%s\" size=%d bold=0 italic=0 charset=\"\" unicode=0 stretchH=100 smooth=1 aa=1 padding=0,0,0,0 spacing=1,1\n", filepath.Base(fontPath), size)
	fmt.Fprintf(fntFile, "common lineHeight=%d base=%d scaleW=%d scaleH=%d pages=1 packed=0\n", size, size, totalWidth, maxHeight)
	fmt.Fprintf(fntFile, "page id=0 file=\"%s.bmp\"\n", filepath.Base(outPrefix))

	// Write mock character data (needed for the verification tool to work)
	cursorX := 0
	for _, char := range chars {
		// Get glyph width
		_, advance, _ := face.GlyphBounds(char)
		width := advance.Ceil()

		fmt.Fprintf(fntFile, "char id=%d x=%d y=0 width=%d height=%d xoffset=0 yoffset=0 xadvance=%d page=0 chnl=15\n",
			char, cursorX, width, maxHeight, width)

		cursorX += width
	}

	return nil
}
