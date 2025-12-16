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

	_ "golang.org/x/image/bmp"
)

type CharDef struct {
	ID, X, Y, W, H, XOffset, YOffset, XAdvance int
}

func main() {
	fntPath := flag.String("fnt", "", "Path to .fnt")

	// CUSTOM USAGE TEXT
	flag.Usage = func() {
		_, err := fmt.Fprintf(flag.CommandLine.Output(), "Usage of %s:\n", "verifier")
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
		flag.PrintDefaults()
	}

	flag.Parse()

	if *fntPath == "" {
		fmt.Println("Usage: go run tools/verifier/main.go -fnt <file.fnt>")
		os.Exit(1)
	}

	bmpPath := strings.TrimSuffix(*fntPath, ".fnt") + ".bmp"
	outPath := strings.TrimSuffix(*fntPath, ".fnt") + "_verify.png"

	if err := runVerify(*fntPath, bmpPath, outPath); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Created verification image:", outPath)
}

func runVerify(fnt, bmp, out string) (err error) {
	chars, err := parseFNT(fnt)
	if err != nil {
		return err
	}

	fImg, err := os.Open(bmp)
	if err != nil {
		return err
	}
	defer func() {
		// For readers, strictly checking close error is less critical,
		// but explicit ignore `_ =` satisfies linter if we don't want to propagate.
		// Propagating is safer.
		if cerr := fImg.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()

	srcImg, _, err := image.Decode(fImg)
	if err != nil {
		return err
	}

	// Create Output Canvas
	dstImg := image.NewRGBA(image.Rect(0, 0, 1024, 2048))
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

	fOut, err := os.Create(out)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := fOut.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()

	return png.Encode(fOut, dstImg)
}

func parseFNT(path string) (chars []CharDef, err error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() {
		if cerr := f.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()

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
	return chars, scanner.Err()
}
