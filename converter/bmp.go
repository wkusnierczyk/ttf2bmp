package converter

import (
	"encoding/binary"
	"image"
	"io"
)

// EncodeBMP writes the image to w in Windows BMP format (32-bit BGRA).
// It uses a negative height in the DIB header to store pixels top-down,
// which matches the coordinate system used by AngelCode/Games.
func EncodeBMP(w io.Writer, img image.Image) error {
	b := img.Bounds()
	width := b.Dx()
	height := b.Dy()

	// BMP Header Size = 14 bytes
	// DIB Header Size = 40 bytes (BITMAPINFOHEADER)
	// Offset to pixel data = 54 bytes
	const (
		fileHeaderSize = 14
		dibHeaderSize  = 40
		offset         = fileHeaderSize + dibHeaderSize
		bpp            = 32 // Bits per pixel (RGBA)
	)

	// Row size must be a multiple of 4 bytes
	rowSize := (width*bpp + 31) / 32 * 4
	imageSize := rowSize * height
	fileSize := offset + imageSize

	// 1. Write File Header
	// Signature "BM"
	if _, err := w.Write([]byte{'B', 'M'}); err != nil {
		return err
	}
	// File Size, Reserved1, Reserved2, Offset
	header := make([]byte, 12)
	binary.LittleEndian.PutUint32(header[0:4], uint32(fileSize))
	binary.LittleEndian.PutUint32(header[8:12], uint32(offset))
	if _, err := w.Write(header); err != nil {
		return err
	}

	// 2. Write DIB Header (BITMAPINFOHEADER)
	dib := make([]byte, dibHeaderSize)
	binary.LittleEndian.PutUint32(dib[0:4], uint32(dibHeaderSize))
	binary.LittleEndian.PutUint32(dib[4:8], uint32(width))
	// Negative height tells parsers the image is top-down
	binary.LittleEndian.PutUint32(dib[8:12], uint32(-height))
	binary.LittleEndian.PutUint16(dib[12:14], 1)           // Planes
	binary.LittleEndian.PutUint16(dib[14:16], uint16(bpp)) // BitCount
	binary.LittleEndian.PutUint32(dib[16:20], 0)           // Compression (BI_RGB)
	binary.LittleEndian.PutUint32(dib[20:24], uint32(imageSize))
	binary.LittleEndian.PutUint32(dib[24:28], 2835) // XPixelsPerMeter (~72 DPI)
	binary.LittleEndian.PutUint32(dib[28:32], 2835) // YPixelsPerMeter
	// ColorsUsed, ColorsImportant left as 0

	if _, err := w.Write(dib); err != nil {
		return err
	}

	// 3. Write Pixel Data
	// BMP 32-bit expects BGRA order
	rowBuffer := make([]byte, rowSize)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		i := 0
		for x := b.Min.X; x < b.Max.X; x++ {
			r, g, bVal, a := img.At(x, y).RGBA()
			// RGBA() returns 0-65535, so shift >> 8 to get 0-255
			rowBuffer[i+0] = uint8(bVal >> 8) // Blue
			rowBuffer[i+1] = uint8(g >> 8)    // Green
			rowBuffer[i+2] = uint8(r >> 8)    // Red
			rowBuffer[i+3] = uint8(a >> 8)    // Alpha
			i += 4
		}
		if _, err := w.Write(rowBuffer); err != nil {
			return err
		}
	}

	return nil
}
