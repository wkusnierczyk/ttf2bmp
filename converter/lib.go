package converter

import (
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

// Generate creates the Font files (image + fnt).
// Now accepts 'padding' to add space between characters.
func Generate(fontPath string, size int, chars string, outPrefix string, format string, padding int) (err error) {
	// 1. Read & Parse Font
	fontBytes, err := os.ReadFile(fontPath)
	if err != nil {
		return fmt.Errorf("reading font file: %w", err)
	}
	f, err := opentype.Parse(fontBytes)
	if err != nil {
		return fmt.Errorf("parsing font: %w", err)
	}

	// 2. Setup Font Face
	face, err := opentype.NewFace(f, &opentype.FaceOptions{
		Size:    float64(size),
		DPI:     72,
		Hinting: font.HintingFull,
	})
	if err != nil {
		return fmt.Errorf("creating face: %w", err)
	}
	defer func() {
		if cerr := face.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()

	// 3. Metrics & Canvas Setup
	metrics := face.Metrics()
	ascent := metrics.Ascent.Ceil()
	lineHeight := metrics.Height.Ceil()

	var totalWidth int
	var validCharCount int

	// Measure loop: Calculate total width including padding
	for _, char := range chars {
		if _, advance, ok := face.GlyphBounds(char); ok {
			totalWidth += advance.Ceil() + padding // Add padding for every char
			validCharCount++
		}
	}

	img := image.NewRGBA(image.Rect(0, 0, totalWidth, lineHeight))

	// Initialize the Drawer
	drawer := &font.Drawer{
		Dst:  img,
		Src:  image.White,
		Face: face,
		Dot:  fixed.P(0, ascent),
	}

	// 4. Draw Characters Individually (to insert padding)
	// We capture the starting X position of each char to match FNT generation exactly
	charPositions := make(map[rune]int)
	currentX := 0

	for _, char := range chars {
		_, advance, ok := face.GlyphBounds(char)
		if !ok {
			continue
		}

		width := advance.Ceil()

		// Record position for FNT (before drawing)
		charPositions[char] = currentX

		// Draw the character
		drawer.DrawString(string(char))

		// Manually add padding to the drawer's dot
		drawer.Dot.X += fixed.I(padding)

		// Advance local integer tracker
		currentX += width + padding
	}

	// 5. Save Image
	ext := "." + format
	if err := func() error {
		imgFile, err := os.Create(outPrefix + ext)
		if err != nil {
			return err
		}
		defer func() {
			cerr := imgFile.Close()
			if err == nil {
				err = cerr
			}
		}()

		if format == "bmp" {
			if err := EncodeBMP(imgFile, img); err != nil {
				return err
			}
		} else {
			if err := png.Encode(imgFile, img); err != nil {
				return err
			}
		}
		return nil
	}(); err != nil {
		return err
	}

	// 6. Save FNT Data
	if err := func() error {
		fntFile, err := os.Create(outPrefix + ".fnt")
		if err != nil {
			return err
		}
		defer func() {
			cerr := fntFile.Close()
			if err == nil {
				err = cerr
			}
		}()

		fileName := filepath.Base(outPrefix) + ext

		// Header
		// We set 'spacing' to reflect the horizontal padding.
		if _, err := fmt.Fprintf(fntFile, "info face=\"%s\" size=%d bold=0 italic=0 charset=\"\" unicode=0 stretchH=100 smooth=1 aa=1 padding=0,0,0,0 spacing=%d,1\n", filepath.Base(fontPath), size, padding); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(fntFile, "common lineHeight=%d base=%d scaleW=%d scaleH=%d pages=1 packed=0\n", lineHeight, ascent, totalWidth, lineHeight); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(fntFile, "page id=0 file=\"%s\"\n", fileName); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(fntFile, "chars count=%d\n", validCharCount); err != nil {
			return err
		}

		// Characters
		for _, char := range chars {
			_, advance, ok := face.GlyphBounds(char)
			if !ok {
				continue
			}

			width := advance.Ceil()
			xPos := charPositions[char] // Retrieve the exact drawn position

			if _, err := fmt.Fprintf(fntFile, "char id=%d x=%d y=0 width=%d height=%d xoffset=0 yoffset=0 xadvance=%d page=0 chnl=15\n",
				char, xPos, width, lineHeight, width); err != nil {
				return err
			}
		}
		return nil
	}(); err != nil {
		return err
	}

	return nil
}
