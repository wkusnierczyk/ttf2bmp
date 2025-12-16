package converter

import (
	"fmt"
	"image"
	"os"
	"path/filepath"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

// Generate creates the BMP and FNT files for a specific font and size.
func Generate(fontPath string, size int, chars string, outPrefix string) (err error) {
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

	// 3. Get Correct Metrics
	// Metrics.Ascent = Distance from top of line to baseline
	// Metrics.Height = Total recommended line height
	metrics := face.Metrics()
	ascent := metrics.Ascent.Ceil()
	lineHeight := metrics.Height.Ceil()

	// 4. Measure Total Width
	var totalWidth int
	for _, char := range chars {
		_, advance, ok := face.GlyphBounds(char)
		if !ok {
			continue
		}
		totalWidth += advance.Ceil()
	}

	// 5. Draw Characters
	// We use lineHeight for the canvas height to ensure all ascenders/descenders fit.
	img := image.NewRGBA(image.Rect(0, 0, totalWidth, lineHeight))

	// Set the Dot (Baseline) exactly at the Ascent line
	dot := fixed.P(0, ascent)

	drawer := &font.Drawer{
		Dst:  img,
		Src:  image.White,
		Face: face,
		Dot:  dot,
	}
	drawer.DrawString(chars)

	// 6. Save Image (BMP)
	if err := func() error {
		bmpFile, err := os.Create(outPrefix + ".bmp")
		if err != nil {
			return err
		}
		defer func() {
			cerr := bmpFile.Close()
			if err == nil {
				err = cerr
			}
		}()

		if err := EncodeBMP(bmpFile, img); err != nil {
			return err
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

		// HEADER:
		// base=<ascent> : Tells renderers where the baseline is relative to the top of the char
		// lineHeight=<height> : Total vertical space for one line
		if _, err := fmt.Fprintf(fntFile, "info face=\"%s\" size=%d bold=0 italic=0 charset=\"\" unicode=0 stretchH=100 smooth=1 aa=1 padding=0,0,0,0 spacing=1,1\n", filepath.Base(fontPath), size); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(fntFile, "common lineHeight=%d base=%d scaleW=%d scaleH=%d pages=1 packed=0\n", lineHeight, ascent, totalWidth, lineHeight); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(fntFile, "page id=0 file=\"%s.bmp\"\n", filepath.Base(outPrefix)); err != nil {
			return err
		}

		// CHARACTERS:
		// y=0, height=lineHeight : Since we draw the full strip, every char block captures the full line height.
		// xoffset=0, yoffset=0 : The ink is already baked into the correct position relative to the block.
		cursorX := 0
		for _, char := range chars {
			_, advance, _ := face.GlyphBounds(char)
			width := advance.Ceil()

			if _, err := fmt.Fprintf(fntFile, "char id=%d x=%d y=0 width=%d height=%d xoffset=0 yoffset=0 xadvance=%d page=0 chnl=15\n",
				char, cursorX, width, lineHeight, width); err != nil {
				return err
			}

			cursorX += width
		}
		return nil
	}(); err != nil {
		return err
	}

	return nil
}
