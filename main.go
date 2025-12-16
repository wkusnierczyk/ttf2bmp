package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	// IMPORT LOCAL PACKAGE
	"ttf2bmp/converter"
)

type Config struct {
	FontPattern string
	Sizes       []int
	Chars       string
	OutputDir   string
}

// Global UI buffer
var logBuffer []string

func main() {
	var fontsFlag, sizesFlag, charsFlag, outDir string

	flag.StringVar(&fontsFlag, "fonts", "", "Glob pattern (e.g. 'assets/*.ttf')")
	flag.StringVar(&fontsFlag, "f", "", "Short for --fonts")
	flag.StringVar(&sizesFlag, "sizes", "", "Comma sizes (e.g. '12,24')")
	flag.StringVar(&sizesFlag, "s", "", "Short for --sizes")
	flag.StringVar(&charsFlag, "chars", "", "Characters to include")
	flag.StringVar(&charsFlag, "c", "", "Short for --chars")
	flag.StringVar(&outDir, "out", ".", "Output dir")
	flag.StringVar(&outDir, "o", ".", "Short for --out")
	flag.Parse()

	cfg, err := validateInputs(fontsFlag, sizesFlag, charsFlag, outDir)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		flag.Usage()
		os.Exit(1)
	}

	files, err := filepath.Glob(cfg.FontPattern)
	if err != nil {
		fmt.Printf("Glob error: %v\n", err)
		os.Exit(1)
	}
	if len(files) == 0 {
		fmt.Printf("No fonts found for pattern: %s\n", cfg.FontPattern)
		os.Exit(0)
	}

	processBatch(files, cfg)
}

func processBatch(files []string, cfg Config) {
	totalJobs := len(files) * len(cfg.Sizes)
	currentJob := 0
	successCount := 0

	// UI Setup
	logBuffer = make([]string, 5)
	fmt.Print("\n\n\n\n\n\n") // Reserve 6 lines
	// Hide cursor
	fmt.Print("\033[?25l")
	defer fmt.Print("\033[?25h")

	os.MkdirAll(cfg.OutputDir, 0755)
	start := time.Now()

	for _, fontPath := range files {
		baseName := filepath.Base(fontPath)
		nameNoExt := strings.TrimSuffix(baseName, filepath.Ext(baseName))

		for _, size := range cfg.Sizes {
			currentJob++
			outPrefix := filepath.Join(cfg.OutputDir, fmt.Sprintf("%s-%d", nameNoExt, size))

			msg := fmt.Sprintf("Processing %s @ %dpx...", baseName, size)
			updateUI(currentJob, totalJobs, msg)

			// CALL LIBRARY
			err := converter.Generate(fontPath, size, cfg.Chars, outPrefix)

			if err != nil {
				updateUI(currentJob, totalJobs, fmt.Sprintf("FAIL %s: %v", baseName, err))
			} else {
				successCount++
			}
		}
	}

	fmt.Print("\033[6A\033[J") // Clear UI area
	fmt.Printf("Done in %v. %d/%d successful.\n", time.Since(start).Round(time.Millisecond), successCount, totalJobs)
}

func updateUI(current, total int, msg string) {
	logBuffer = append(logBuffer[1:], msg)
	percent := 0
	if total > 0 {
		percent = (current * 100) / total
	}

	width := 50
	filled := (percent * width) / 100
	bar := fmt.Sprintf("[%s%s]", strings.Repeat("=", filled), strings.Repeat(" ", width-filled))

	fmt.Print("\033[6A")
	fmt.Printf("%s %3d%% (%d/%d)\033[K\n", bar, percent, current, total)
	for _, line := range logBuffer {
		if len(line) > 75 {
			line = line[:72] + "..."
		}
		fmt.Printf("%s\033[K\n", line)
	}
}

func validateInputs(f, s, c, o string) (Config, error) {
	if f == "" || s == "" || c == "" {
		return Config{}, fmt.Errorf("missing arguments")
	}

	var sizeInts []int
	for _, p := range strings.Split(s, ",") {
		val, err := strconv.Atoi(strings.TrimSpace(p))
		if err != nil {
			return Config{}, err
		}
		sizeInts = append(sizeInts, val)
	}
	sort.Ints(sizeInts)
	return Config{FontPattern: f, Sizes: sizeInts, Chars: c, OutputDir: o}, nil
}
