package ttf2bmp

import (
	"bufio"
	"fmt"
	"image"
	"image/draw"
	"image/png"
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
	defer func() {
		err := face.Close()
		if err != nil {
			fmt.Println("Warning: failed to close font face: %w", err)
		}
	}()

	// Font metrics
	metrics := face.Metrics()
	lineHeight := (metrics.Height).Ceil()
	baseLine := (metrics.Ascent).Ceil()

	// 1. Prepare the Destination Image IMMEDIATELY
	atlas := image.NewRGBA(image.Rect(0, 0, config.SheetWidth, config.SheetHeight))
	charMap := make(map[rune]CharData)

	// Cursor state
	currentX, currentY := 1, 1
	rowHeight := 0

	// 2. Single Pass: Render -> Pack -> Draw -> Forget
	for _, r := range config.Runes {
		// Render the individual glyph
		bounds, imageMask, pointMask, advance, ok := face.Glyph(fixed.P(0, 0), r)
		if !ok {
			fmt.Printf("Skipping rune [%c] - not found\n", r)
			continue
		}

		// Calculate dimensions
		maxX := bounds.Max.X
		minX := bounds.Min.X
		maxY := bounds.Max.Y
		minY := bounds.Min.Y
		width := maxX - minX
		height := maxY - minY

		// 3. Smart Wrapping Logic
		// Check if we fit in the current row
		if currentX+width >= config.SheetWidth {
			currentX = 1
			// Move Y down by the tallest item in the previous row
			if rowHeight == 0 {
				currentY += lineHeight + config.Padding
			} else {
				currentY += rowHeight + config.Padding
			}
			rowHeight = 0
		}

		// 4. Height Check
		if currentY+height >= config.SheetHeight {
			return nil, fmt.Errorf("atlas filled up! stopped at rune [%c]. Size (%dx%d) too small",
				r, config.SheetWidth, config.SheetHeight)
		}

		// 5. Draw IMMEDIATELY
		dstRect := image.Rect(currentX, currentY, currentX+width, currentY+height)

		draw.DrawMask(
			atlas,
			dstRect,
			image.White,
			image.Point{},
			imageMask,
			pointMask,
			draw.Over,
		)

		// 6. Store Metadata
		charMap[r] = CharData{
			ID:      r,
			X:       currentX,
			Y:       currentY,
			Width:   width,
			Height:  height,
			XOffset: bounds.Min.X,
			// TODO: validate the formula
			//YOffset:  bounds.Min.Y + baseLine,
			XAdvance: advance.Ceil(),
		}

		// 7. Update Cursor
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
	defer func() {
		_ = f.Close()
	}()
	return png.Encode(f, bf.Image)
}

// SaveFNT saves the AngelCode text format descriptor.
func (bf *BitmapFont) SaveFNT(filename, imgFilename string) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer func() {
		_ = f.Close()
	}()
	w := bufio.NewWriter(f)

	_, _ = fmt.Fprintf(w, "info face=\"%s\" size=%d bold=0 italic=0 charset=\"\" unicode=1 stretchH=100 smooth=1 aa=1 padding=0,0,0,0 spacing=1,1\n", bf.FaceName, bf.FontSize)
	width := bf.Image.Bounds().Max.X
	height := bf.Image.Bounds().Max.Y
	_, _ = fmt.Fprintf(w, "common lineHeight=%d base=%d scaleW=%d scaleH=%d pages=1 packed=0\n", bf.LineHeight, bf.Base, width, height)
	_, _ = fmt.Fprintf(w, "page id=0 file=\"%s\"\n", imgFilename)
	_, _ = fmt.Fprintf(w, "chars count=%d\n", len(bf.Chars))

	for _, c := range bf.Chars {
		_, _ = fmt.Fprintf(w, "char id=%-4d x=%-4d y=%-4d width=%-4d height=%-4d xoffset=%-4d yoffset=%-4d xadvance=%-4d page=0 chnl=15\n",
			c.ID, c.X, c.Y, c.Width, c.Height, c.XOffset, c.YOffset, c.XAdvance)
	}

	return w.Flush()
}

// Helper to load file bytes
func LoadFontBytes(path string) ([]byte, error) {
	return os.ReadFile(path) // Updated from ioutil.ReadFile
}
