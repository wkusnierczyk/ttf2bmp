package ttf2bmp

import (
	"os"
	"testing"

	"golang.org/x/image/font/gofont/goregular"
)

func TestGenerate_Success(t *testing.T) {
	// Use Go's built-in font for testing so we don't depend on external files
	fontData := goregular.TTF

	cfg := Config{
		FontBytes:   fontData,
		FontSize:    24,
		DPI:         72,
		SheetWidth:  512,
		SheetHeight: 512,
		Runes:       []rune{'A', 'B', 'C', '1', '2', '3'},
	}

	bf, err := Generate(cfg)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if bf == nil {
		t.Fatal("BitmapFont is nil")
	}

	if len(bf.Chars) != 6 {
		t.Errorf("Expected 6 chars, got %d", len(bf.Chars))
	}

	if bf.LineHeight <= 0 {
		t.Error("LineHeight should be > 0")
	}

	// Check if 'A' exists
	if _, ok := bf.Chars['A']; !ok {
		t.Error("Character A is missing")
	}
}

func TestGenerate_AtlasTooSmall(t *testing.T) {
	fontData := goregular.TTF

	// Try to fit 24px font into a 10x10 image (should fail)
	cfg := Config{
		FontBytes:   fontData,
		FontSize:    24,
		DPI:         72,
		SheetWidth:  10,
		SheetHeight: 10,
		Runes:       []rune{'A', 'B', 'C'},
	}

	_, err := Generate(cfg)
	if err == nil {
		t.Error("Expected error for small atlas, got nil")
	}
}

func TestFileExport(t *testing.T) {
	fontData := goregular.TTF
	cfg := Config{
		FontBytes:   fontData,
		FontSize:    16,
		DPI:         72,
		SheetWidth:  256,
		SheetHeight: 256,
		Runes:       []rune{'X'},
	}

	bf, _ := Generate(cfg)

	// Test PNG Export
	err := bf.SavePNG("test_output.png")
	if err != nil {
		t.Errorf("Failed to save PNG: %v", err)
	}
	defer func() {
		_ = os.Remove("test_output.png")
	}()

	// Test FNT Export
	err = bf.SaveFNT("test_output.fnt", "test_output.png")
	if err != nil {
		t.Errorf("Failed to save FNT: %v", err)
	}
	defer func() {
		_ = os.Remove("test_output.fnt")
	}()
}

// Benchmark the generation process
func BenchmarkGenerate(b *testing.B) {
	fontData := goregular.TTF

	// Create a list of 100 characters
	var runes []rune
	for i := 32; i < 132; i++ {
		runes = append(runes, rune(i))
	}

	cfg := Config{
		FontBytes:   fontData,
		FontSize:    32,
		DPI:         72,
		SheetWidth:  1024,
		SheetHeight: 1024,
		Runes:       runes,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = Generate(cfg)
	}
}
