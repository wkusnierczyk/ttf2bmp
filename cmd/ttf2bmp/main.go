package main

import (
	"flag"
	"fmt"
	"log"
	"path/filepath"
	"ttf2bmp"
)

func main() {
	fontPath := flag.String("font", "", "Path to the .ttf font file (required)")
	outputName := flag.String("out", "output", "Base name for output files (will generate .png and .fnt)")
	size := flag.Float64("size", 32, "Font size in pixels")
	dpi := flag.Float64("dpi", 72, "DPI (dots per inch)")
	sheetSize := flag.Int("sheet", 512, "Width and Height of the texture atlas (square)")
	chars := flag.String("chars", "", "Custom characters to include (default: ASCII 32-126)")

	flag.Parse()

	if *fontPath == "" {
		flag.Usage()
		log.Fatal("Please provide a font file using -font")
	}

	// Read Font File
	fontBytes, err := ttf2bmp.LoadFontBytes(*fontPath)
	if err != nil {
		log.Fatalf("Error reading font file: %v", err)
	}

	// Determine runes
	var runeList []rune
	if *chars != "" {
		runeList = []rune(*chars)
	} else {
		// Default ASCII range
		for i := 32; i < 127; i++ {
			runeList = append(runeList, rune(i))
		}
	}

	config := ttf2bmp.Config{
		FontBytes: fontBytes,
		FontSize:  *size,
		DPI:       *dpi,
		SheetW:    *sheetSize,
		SheetH:    *sheetSize,
		Runes:     runeList,
	}

	fmt.Printf("Generating font from %s (Size: %.0f)...\n", *fontPath, *size)

	bmFont, err := ttf2bmp.Generate(config)
	if err != nil {
		log.Fatalf("Error generating bitmap font: %v", err)
	}

	// Output files
	pngName := *outputName + ".png"
	fntName := *outputName + ".fnt"

	// Save PNG
	if err := bmFont.SavePNG(pngName); err != nil {
		log.Fatalf("Error saving PNG: %v", err)
	}

	// Save FNT
	// Note: The FNT file references the PNG filename, so we pass just the base name
	if err := bmFont.SaveFNT(fntName, filepath.Base(pngName)); err != nil {
		log.Fatalf("Error saving FNT: %v", err)
	}

	fmt.Printf("Success!\nSaved Atlas: %s\nSaved Descriptor: %s\n", pngName, fntName)
}
