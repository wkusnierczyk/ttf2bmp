package converter

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// Helper to locate a valid TTF for testing (downloaded by CI)
const testFontPath = "../test_data/Roboto-Regular.ttf"

func TestMain(m *testing.M) {
	// Setup: Ensure test_data directory exists
	if err := os.MkdirAll("../test_data", 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Test setup failed: could not create ../test_data: %v\n", err)
		os.Exit(1)
	}

	// Run Tests
	code := m.Run()

	// Teardown: Cleanup output
	if err := os.RemoveAll("../test_output"); err != nil {
		fmt.Fprintf(os.Stderr, "Test teardown warning: could not remove ../test_output: %v\n", err)
	}

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
	if err := os.MkdirAll(outDir, 0755); err != nil {
		t.Fatalf("Failed to create output directory: %v", err)
	}

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
	if err := os.MkdirAll(outDir, 0755); err != nil {
		b.Fatalf("Failed to create benchmark directory: %v", err)
	}

	outPrefix := filepath.Join(outDir, "bench_font")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// We must check the error here to satisfy linter,
		// and also to ensure we aren't benchmarking a fast-fail error path.
		if err := Generate(testFontPath, 24, "ABCDEFGHIJKLMNOPQRSTUVWXYZ", outPrefix); err != nil {
			b.Fatalf("Benchmark failed on iteration %d: %v", i, err)
		}
	}
}
