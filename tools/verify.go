package main

import (
	"bufio"
	"flag"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"os"
	"strconv"
	"strings"

	// Support BMP decoding
	_ "golang.org/x/image/bmp"
)

// Simple Char struct
type CharDef struct {
	ID, X, Y, W, H, XOffset, YOffset, XAdvance int
}

func main() {
	fntPath := flag.String("fnt", "", "Path to .fnt")
	flag.Parse()

	if *fntPath == "" {
		fmt.Println("Usage: go run verify_fonts.go -fnt <file.fnt>")
		os.Exit(1)
	}

	// Assuming BMP is same name but .bmp extension
	bmpPath := strings.TrimSuffix(*fntPath, ".fnt") + ".bmp"
	outPath := strings.TrimSuffix(*fntPath, ".fnt") + "_verify.png"

	if err := runVerify(*fntPath, bmpPath, outPath); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Created:", outPath)
}

func runVerify(fnt, bmp, out string) error {
	chars, err := parseFNT(fnt)
	if err != nil {
		return err
	}

	fImg, err := os.Open(bmp)
	if err != nil {
		return err
	}
	defer fImg.Close()

	srcImg, _, err := image.Decode(fImg)
	if err != nil {
		return err
	}

	// Create Output Canvas (1024 width)
	dstImg := image.NewRGBA(image.Rect(0, 0, 1024, 2048)) // Large fixed height for simplicity

	cursorX, cursorY := 10, 50
	rowH := 0

	for _, c := range chars {
		if cursorX+c.XAdvance > 1000 {
			cursorX = 10
			cursorY += rowH + 10
			rowH = 0
		}
		if c.H > rowH {
			rowH = c.H
		}

		draw.Draw(dstImg,
			image.Rect(cursorX+c.XOffset, cursorY+c.YOffset, cursorX+c.XOffset+c.W, cursorY+c.YOffset+c.H),
			srcImg,
			image.Point{c.X, c.Y},
			draw.Over,
		)
		cursorX += c.XAdvance
	}

	fOut, _ := os.Create(out)
	defer fOut.Close()
	return png.Encode(fOut, dstImg)
}

func parseFNT(path string) ([]CharDef, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var chars []CharDef
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "char ") {
			data := make(map[string]int)
			for _, field := range strings.Fields(line) {
				parts := strings.Split(field, "=")
				if len(parts) == 2 {
					val, _ := strconv.Atoi(parts[1])
					data[parts[0]] = val
				}
			}
			chars = append(chars, CharDef{
				ID: data["id"], X: data["x"], Y: data["y"], W: data["width"], H: data["height"],
				XOffset: data["xoffset"], YOffset: data["yoffset"], XAdvance: data["xadvance"],
			})
		}
	}
	return chars, nil
}
