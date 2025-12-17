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
// Now accepts 'hinting' ("none", "vertical", "full")
func Generate(fontPath string, size int, chars string, outPrefix string, format string, padding int, hinting string) (err error) {
	// 1. Read & Parse Font
	fontBytes, err := os.ReadFile(fontPath)
	if err != nil {
		return fmt.Errorf("reading font file: %w", err)
	}
	f, err := opentype.Parse(fontBytes)
	if err != nil {
		return fmt.Errorf("parsing font: %w", err)
	}

	// 2. Resolve Hinting Option
	var h font.Hinting
	switch hinting {
	case "none":
		h = font.HintingNone
	case "vertical":
		h = font.HintingVertical
	default:
		h = font.HintingFull // Default to crisp/sharp
	}

	// 3. Setup Font Face
	face, err := opentype.NewFace(f, &opentype.FaceOptions{
		Size:    float64(size),
		DPI:     72,
		Hinting: h, // Use the selected hinting
	})
	if err != nil {
		return fmt.Errorf("creating face: %w", err)
	}
	defer func() {
		if cerr := face.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()

	// 4. Metrics & Canvas Setup
	metrics := face.Metrics()
	ascent := metrics.Ascent.Ceil()
	lineHeight := metrics.Height.Ceil()

	var totalWidth int
	var validCharCount int

	// Measure loop
	for _, char := range chars {
		if _, advance, ok := face.GlyphBounds(char); ok {
			totalWidth += advance.Ceil() + padding
			validCharCount++
		}
	}

	img := image.NewRGBA(image.Rect(0, 0, totalWidth, lineHeight))

	// Initialize Drawer
	drawer := &font.Drawer{
		Dst:  img,
		Src:  image.White,
		Face: face,
		Dot:  fixed.P(0, ascent),
	}

	// 5. Draw Characters Individually
	charPositions := make(map[rune]int)
	currentX := 0

	for _, char := range chars {
		_, advance, ok := face.GlyphBounds(char)
		if !ok {
			continue
		}

		width := advance.Ceil()

		// Record position
		charPositions[char] = currentX

		// CRITICAL FIX: Explicitly set the Dot to the exact integer position.
		// This prevents sub-pixel accumulation errors (drifting) and ensures
		// the image pixels align 1:1 with the FNT coordinates.
		drawer.Dot = fixed.P(currentX, ascent)

		// Draw
		drawer.DrawString(string(char))

		// Advance local integer tracker
		currentX += width + padding
	}

	// 6. Save Image
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

	// 7. Save FNT Data
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

		for _, char := range chars {
			_, advance, ok := face.GlyphBounds(char)
			if !ok {
				continue
			}
			width := advance.Ceil()
			xPos := charPositions[char]

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
