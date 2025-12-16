package converter

import (
	"os"
	"path/filepath"
	"testing"
)

// Helper to locate a valid TTF for testing.
// In CI, we will download a font to this location.
const testFontPath = "../test_data/Roboto-Regular.ttf"

func TestMain(m *testing.M) {
	// Setup: Ensure test_data directory exists
	os.MkdirAll("../test_data", 0755)

	// Run Tests
	code := m.Run()

	// Teardown: Cleanup output
	os.RemoveAll("../test_output")
	os.Exit(code)
}

func TestGenerate_Validation(t *testing.T) {
	// Test missing file
	err := Generate("non_existent.ttf", 12, "A", "out")
	if err == nil {
		t.Error("Expected error for missing font file, got nil")
	}
}

func TestGenerate_Success(t *testing.T) {
	// Check if we have the test font (skip if running locally without setup)
	if _, err := os.Stat(testFontPath); os.IsNotExist(err) {
		t.Skipf("Skipping test: %s not found. (Download it or map a system font)", testFontPath)
	}

	outDir := "../test_output"
	os.MkdirAll(outDir, 0755)
	outPrefix := filepath.Join(outDir, "test_font")

	err := Generate(testFontPath, 32, "ABCabc123", outPrefix)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Verify Files Created
	if _, err := os.Stat(outPrefix + ".bmp"); os.IsNotExist(err) {
		t.Error("BMP file was not created")
	}
	if _, err := os.Stat(outPrefix + ".fnt"); os.IsNotExist(err) {
		t.Error("FNT file was not created")
	}
}

// Benchmark the generation process
func BenchmarkGenerate(b *testing.B) {
	if _, err := os.Stat(testFontPath); os.IsNotExist(err) {
		b.Skipf("Skipping benchmark: %s not found", testFontPath)
	}

	// Quiet output for benchmark
	outDir := "../test_output/bench"
	os.MkdirAll(outDir, 0755)
	outPrefix := filepath.Join(outDir, "bench_font")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// We use a small char set for the benchmark to measure overhead + render
		Generate(testFontPath, 24, "ABCDEFGHIJKLMNOPQRSTUVWXYZ", outPrefix)
	}
}
