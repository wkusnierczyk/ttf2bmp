package converter

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGenerate(t *testing.T) {
	// 1. Setup paths
	// We look for the font in the project root's test_data folder
	wd, _ := os.Getwd()
	projectRoot := filepath.Dir(wd)
	fontPath := filepath.Join(projectRoot, "test_data", "Go-Regular.ttf")

	// If the test font isn't there (e.g. running 'go test' before 'make fetch-test-data'),
	// we skip rather than fail.
	if _, err := os.Stat(fontPath); os.IsNotExist(err) {
		t.Skipf("Test font not found at %s. Run 'make fetch-test-data' first.", fontPath)
	}

	outDir := t.TempDir()
	prefix := filepath.Join(outDir, "test-output")

	// 2. Call Generate (Updated signature)
	// We test "png" since that is the default functionality we restored.
	err := Generate(fontPath, 32, "A", prefix, "png")
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// 3. Verify output files exist
	if _, err := os.Stat(prefix + ".png"); os.IsNotExist(err) {
		t.Error("Expected .png file to be created")
	}
	if _, err := os.Stat(prefix + ".fnt"); os.IsNotExist(err) {
		t.Error("Expected .fnt file to be created")
	}
}
