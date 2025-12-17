package converter

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGenerate(t *testing.T) {
	// 1. Setup Paths
	// We assume the test runs from within the 'converter' directory,
	// so the test_data is one level up.
	wd, _ := os.Getwd()
	rootDir := filepath.Dir(wd)
	fontPath := filepath.Join(rootDir, "test_data", "Go-Regular.ttf")

	// Verify font exists before running (skip if missing)
	if _, err := os.Stat(fontPath); os.IsNotExist(err) {
		t.Skipf("Test font not found at %s. Run 'make fetch-test-data' first.", fontPath)
	}

	// Create a temporary output directory
	tempDir := t.TempDir()
	outPrefix := filepath.Join(tempDir, "test_output")

	// 2. Execution
	err := Generate(fontPath, 32, "ABC", outPrefix, "png", 2, "full")

	// 3. Assertions
	if err != nil {
		t.Fatalf("Generate() failed: %v", err)
	}

	// Check if .png was created
	if _, err := os.Stat(outPrefix + ".png"); os.IsNotExist(err) {
		t.Errorf("Expected %s.png to exist", outPrefix)
	}

	// Check if .fnt was created
	if _, err := os.Stat(outPrefix + ".fnt"); os.IsNotExist(err) {
		t.Errorf("Expected %s.fnt to exist", outPrefix)
	}
}
