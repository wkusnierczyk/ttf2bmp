package ttf2bmp

import (
	"bufio"
	"fmt"
	"image"
	"image/draw"
	"image/png"

	"io/ioutil"
	"os"

	"github.com/golang/freetype"
	"github.com/golang/freetype/truetype"
	"golang.org/x/image/math/fixed"
)

// CharData holds metrics for a single character in the atlas.
type CharData struct {
	ID       rune
	X, Y     int
	Width    int
	Height   int
	XOffset  int
	YOffset  int
	XAdvance int
}

// Config configures the generation process.
type Config struct {
	FontBytes   []byte
	FontSize    float64
	DPI         float64
	SheetWidth  int
	SheetHeight int
	Runes       []rune
	Padding     int
}

// BitmapFont represents the generated font data.
type BitmapFont struct {
	Image      *image.RGBA
	Chars      map[rune]CharData
	LineHeight int
	Base       int
	FaceName   string
	FontSize   int
}

// Generate creates a BitmapFont from the provided configuration.
func Generate(config Config) (*BitmapFont, error) {
	parsedFont, err := freetype.ParseFont(config.FontBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse font: %w", err)
	}

	context := freetype.NewContext()
	context.SetDPI(config.DPI)
	context.SetFont(parsedFont)
	context.SetFontSize(config.FontSize)

	options := truetype.Options{
		Size: config.FontSize,
		DPI:  config.DPI,
	}
	face := truetype.NewFace(parsedFont, &options)
	defer face.Close()

	// 1. Measure all glyphs
	type renderGlyph struct {
		r         rune
		imageMask image.Image // Changed to image.Image interface
		pointMask image.Point // Added imageMask point (offset)
		bounds    image.Rectangle
		advance   fixed.Int26_6
	}

	var glyphs []renderGlyph
	maxHeight := 0

	for _, r := range config.Runes {
		bounds, imageMask, pointMask, advance, ok := face.Glyph(fixed.P(0, 0), r)

		if !ok {
			continue
		}

		// Calculate vertical metrics
		boundsHeight := bounds.Max.Y - bounds.Min.Y
		if boundsHeight > maxHeight {
			maxHeight = boundsHeight
		}

		glyphs = append(glyphs, renderGlyph{
			r:         r,
			imageMask: imageMask,
			pointMask: pointMask,
			bounds:    bounds,
			advance:   advance,
		})

		// debug
		fmt.Printf("[%#U] read with X: %d-%d, Y: %d-%d\n", r, bounds.Min.X, bounds.Max.X, bounds.Min.Y, bounds.Max.Y)

	}

	// 2. Pack Glyphs (Simple Shelf Packing)
	atlas := image.NewRGBA(image.Rect(0, 0, config.SheetWidth, config.SheetHeight))
	charMap := make(map[rune]CharData)

	// Start drawing at (1,1); update as glyphs are placed
	currentX, currentY := 1, 1

	// Initial row height; update as glyphs are placed, reset when wrapping to new row
	rowHeight := 0

	// Font face metrics
	metrics := face.Metrics()
	lineHeight := (metrics.Height).Ceil()
	baseLine := (metrics.Ascent).Ceil()

	for _, glyph := range glyphs {

		// Glyph coordinates and dimensions
		maxX := glyph.bounds.Max.X
		minX := glyph.bounds.Min.X
		maxY := glyph.bounds.Max.Y
		minY := glyph.bounds.Min.Y
		width := maxX - minX
		height := maxY - minY

		// debug
		fmt.Printf("[%#U] X(%d to %d = %d), Y(%d to %d = %d)\n", glyph.r, minX, maxX, width, minY, maxY, height)

		// Check for space in the current row, if insufficient, wrap to a new row
		if currentX+width >= config.SheetWidth {
			currentX = 1
			currentY += rowHeight + config.Padding
			rowHeight = 0
		}

		// Check for space in the current column, if insufficient, return error
		if currentY+height >= config.SheetHeight {
			return nil, fmt.Errorf("atlas size (%dx%d) too small for font size %v", config.SheetWidth, config.SheetHeight, config.FontSize)
		}

		// Create coordinates for the glyph in the atlas
		minPoint := image.Point{currentX, currentY}
		maxPoint := minPoint.Add(image.Point{width, height})

		// Create the destination rectangle on the atlas
		destinationRectangle := image.Rectangle{Min: minPoint, Max: maxPoint}

		// Draw using the imageMask and the imageMask point returned by face.Glyph
		draw.DrawMask(
			atlas,
			destinationRectangle,
			image.White,
			image.Point{},
			glyph.imageMask,
			glyph.pointMask,
			draw.Over,
		)

		// Store Metrics
		charMap[glyph.r] = CharData{
			ID: glyph.r,
			//X:        currentX + config.Padding,
			//Y:        currentY + config.Padding,
			X:        currentX,
			Y:        currentY,
			Width:    width,
			Height:   height,
			XOffset:  glyph.bounds.Min.X,
			YOffset:  glyph.bounds.Min.Y + baseLine,
			XAdvance: glyph.advance.Ceil(),
		}

		// Advance cursor
		currentX += width + config.Padding
		if height > rowHeight {
			rowHeight = height
		}
	}

	return &BitmapFont{
		Image:      atlas,
		Chars:      charMap,
		LineHeight: lineHeight,
		Base:       baseLine,
		FaceName:   parsedFont.Name(truetype.NameIDFontFamily),
		FontSize:   int(config.FontSize),
	}, nil
}

// SavePNG saves the texture atlas.
func (bf *BitmapFont) SavePNG(filename string) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, bf.Image)
}

// SaveFNT saves the AngelCode text format descriptor.
func (bf *BitmapFont) SaveFNT(filename, imgFilename string) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	w := bufio.NewWriter(f)

	// Write Info line
	fmt.Fprintf(w, "info face=\"%s\" size=%d bold=0 italic=0 charset=\"\" unicode=1 stretchH=100 smooth=1 aa=1 padding=0,0,0,0 spacing=1,1\n", bf.FaceName, bf.FontSize)

	// Write Common line
	width := bf.Image.Bounds().Max.X
	height := bf.Image.Bounds().Max.Y
	fmt.Fprintf(w, "common lineHeight=%d base=%d scaleW=%d scaleH=%d pages=1 packed=0\n", bf.LineHeight, bf.Base, width, height)

	// Write Page line
	fmt.Fprintf(w, "page id=0 file=\"%s\"\n", imgFilename)

	// Write Chars count
	fmt.Fprintf(w, "chars count=%d\n", len(bf.Chars))

	// Write individual characters
	for _, c := range bf.Chars {
		fmt.Fprintf(w, "char id=%-4d x=%-4d y=%-4d width=%-4d height=%-4d xoffset=%-4d yoffset=%-4d xadvance=%-4d page=0 chnl=15\n",
			c.ID, c.X, c.Y, c.Width, c.Height, c.XOffset, c.YOffset, c.XAdvance)
	}

	return w.Flush()
}

// Helper to load file bytes
func LoadFontBytes(path string) ([]byte, error) {
	return ioutil.ReadFile(path)
}
