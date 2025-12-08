package ttf2bmp

import (
	"bufio"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	// "io"
	"io/ioutil"
	// "math"
	"os"

	"github.com/golang/freetype"
	"github.com/golang/freetype/truetype"
	// "golang.org/x/image/font"
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
	FontBytes []byte
	FontSize  float64
	DPI       float64
	SheetW    int
	SheetH    int
	Runes     []rune
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
func Generate(cfg Config) (*BitmapFont, error) {
	parsedFont, err := freetype.ParseFont(cfg.FontBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse font: %w", err)
	}

	// Create a context for rendering
	c := freetype.NewContext()
	c.SetDPI(cfg.DPI)
	c.SetFont(parsedFont)
	c.SetFontSize(cfg.FontSize)

	// Calculate scale to convert FUnits to pixels
	opts := truetype.Options{
		Size: cfg.FontSize,
		DPI:  cfg.DPI,
	}
	face := truetype.NewFace(parsedFont, &opts)
	defer face.Close()

	// 1. Measure all glyphs
	type renderGlyph struct {
		r       rune
		mask    *image.Alpha
		bounds  image.Rectangle
		advance fixed.Int26_6
		bearing fixed.Point26_6
	}

	var glyphs []renderGlyph
	maxHeight := 0

	for _, r := range cfg.Runes {
		// We use the face to get the glyph path and bounds
		idx := parsedFont.Index(r)

		// Create a temporary mask to measure exact pixel bounds
		// Note: This is a simplified approach. For production, you might want to
		// render directly to the atlas to save memory, but 2-pass is safer for packing.
		mask, bounds, _, ok := face.Glyph(fixed.P(0, 0), idx)
		if !ok {
			continue
		}

		adv, ok := face.GlyphAdvance(idx)
		if !ok {
			continue
		}

		// Calculate vertical metrics
		boundsHeight := bounds.Max.Y - bounds.Min.Y
		if boundsHeight > maxHeight {
			maxHeight = boundsHeight
		}

		glyphs = append(glyphs, renderGlyph{
			r:       r,
			mask:    mask,
			bounds:  bounds,
			advance: adv,
		})
	}

	// 2. Pack Glyphs (Simple Shelf Packing)
	atlas := image.NewRGBA(image.Rect(0, 0, cfg.SheetW, cfg.SheetH))
	charMap := make(map[rune]CharData)

	currentX, currentY := 1, 1 // Padding
	rowHeight := 0

	metrics := face.Metrics()
	lineHeight := (metrics.Height).Ceil()
	baseLine := (metrics.Ascent).Ceil()

	for _, g := range glyphs {
		gw := g.bounds.Max.X - g.bounds.Min.X
		gh := g.bounds.Max.Y - g.bounds.Min.Y

		// Wrap to next line if needed
		if currentX+gw+1 >= cfg.SheetW {
			currentX = 1
			currentY += rowHeight + 1
			rowHeight = 0
		}

		if currentY+gh+1 >= cfg.SheetH {
			return nil, fmt.Errorf("atlas size (%dx%d) too small for font size %v", cfg.SheetW, cfg.SheetH, cfg.FontSize)
		}

		// Draw glyph to atlas
		// The Glyph function returns an image where the Dot is at origin relative to bounds.
		// We need to shift it to our currentX, currentY.
		drawPoint := image.Point{currentX, currentY}

		// Draw the mask onto the RGBA atlas (white text, transparent background)
		rect := image.Rectangle{Min: drawPoint, Max: drawPoint.Add(image.Point{gw, gh})}
		draw.DrawMask(atlas, rect, image.White, image.Point{}, g.mask, g.bounds.Min, draw.Over)

		// Store Metrics
		charMap[g.r] = CharData{
			ID:       g.r,
			X:        currentX,
			Y:        currentY,
			Width:    gw,
			Height:   gh,
			XOffset:  g.bounds.Min.X,
			YOffset:  g.bounds.Min.Y + baseLine, // Offset from top of line
			XAdvance: g.advance.Ceil(),
		}

		// Advance cursor
		currentX += gw + 1
		if gh > rowHeight {
			rowHeight = gh
		}
	}

	return &BitmapFont{
		Image:      atlas,
		Chars:      charMap,
		LineHeight: lineHeight,
		Base:       baseLine,
		FaceName:   parsedFont.Name(truetype.NameIDFontFamily),
		FontSize:   int(cfg.FontSize),
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
