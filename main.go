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

	"ttf2bmp/converter"
)

// Set via linker flags
var Version = "dev"

type Config struct {
	FontPattern string
	Sizes       []int
	Chars       string
	OutputDir   string
	Format      string
	Padding     int // New field
}

// Global UI buffer
var logBuffer []string

func main() {
	var fontsFlag, sizesFlag, charsFlag, outDir, typeFlag string
	var paddingFlag int
	var showVersion bool

	flag.Usage = func() {
		_, _ = fmt.Fprintf(flag.CommandLine.Output(), "Usage of %s (%s):\n", "ttf2bmp", Version)
		flag.PrintDefaults()
	}

	flag.StringVar(&fontsFlag, "fonts", "", "Glob pattern (e.g. 'assets/*.ttf')")
	flag.StringVar(&fontsFlag, "f", "", "Short for --fonts")
	flag.StringVar(&sizesFlag, "sizes", "", "Comma sizes (e.g. '12,24')")
	flag.StringVar(&sizesFlag, "s", "", "Short for --sizes")
	flag.StringVar(&charsFlag, "chars", "", "Characters to include")
	flag.StringVar(&charsFlag, "c", "", "Short for --chars")
	flag.StringVar(&outDir, "out", ".", "Output dir")
	flag.StringVar(&outDir, "o", ".", "Short for --out")
	flag.StringVar(&typeFlag, "type", "png", "Output type: 'png' or 'bmp'")
	flag.StringVar(&typeFlag, "t", "png", "Short for --type")
	// NEW FLAGS
	flag.IntVar(&paddingFlag, "padding", 2, "Padding between characters (pixels)")
	flag.IntVar(&paddingFlag, "p", 2, "Short for --padding")

	flag.BoolVar(&showVersion, "version", false, "Print version")

	flag.Parse()

	if showVersion {
		fmt.Printf("ttf2bmp version %s\n", Version)
		os.Exit(0)
	}

	cfg, err := validateInputs(fontsFlag, sizesFlag, charsFlag, outDir, typeFlag, paddingFlag)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		flag.Usage()
		os.Exit(1)
	}

	// Glob / File detection
	files, err := filepath.Glob(cfg.FontPattern)
	if err != nil {
		fmt.Printf("Glob error: %v\n", err)
		os.Exit(1)
	}
	if len(files) == 0 {
		if _, err := os.Stat(cfg.FontPattern); err == nil {
			files = []string{cfg.FontPattern}
		} else {
			fmt.Printf("No fonts found for pattern: %s\n", cfg.FontPattern)
			os.Exit(0)
		}
	}

	processBatch(files, cfg)
}

func processBatch(files []string, cfg Config) {
	totalJobs := len(files) * len(cfg.Sizes)
	currentJob := 0
	successCount := 0
	var failures []string

	// UI Setup
	logBuffer = make([]string, 5)
	fmt.Print("\n\n\n\n\n\n")
	fmt.Print("\033[?25l")
	defer fmt.Print("\033[?25h")

	if err := os.MkdirAll(cfg.OutputDir, 0755); err != nil {
		fmt.Print("\033[?25h")
		fmt.Printf("Error: Failed to create output directory: %v\n", err)
		os.Exit(1)
	}

	start := time.Now()

	for _, fontPath := range files {
		baseName := filepath.Base(fontPath)
		nameNoExt := strings.TrimSuffix(baseName, filepath.Ext(baseName))

		for _, size := range cfg.Sizes {
			currentJob++
			outPrefix := filepath.Join(cfg.OutputDir, fmt.Sprintf("%s-%d", nameNoExt, size))

			// Include padding in the log message for clarity
			msg := fmt.Sprintf("Processing %s @ %dpx (pad:%d) ...", baseName, size, cfg.Padding)
			updateUI(currentJob, totalJobs, msg)

			// Pass cfg.Padding to Generate
			err := converter.Generate(fontPath, size, cfg.Chars, outPrefix, cfg.Format, cfg.Padding)

			if err != nil {
				errMsg := fmt.Sprintf("FAIL %s @ %dpx: %v", baseName, size, err)
				updateUI(currentJob, totalJobs, errMsg)
				failures = append(failures, errMsg)
			} else {
				successCount++
			}
		}
	}

	fmt.Print("\033[6A\033[J")
	fmt.Printf("Done in %v. %d/%d successful.\n", time.Since(start).Round(time.Millisecond), successCount, totalJobs)

	if len(failures) > 0 {
		fmt.Println("\n=== FAILURE REPORT ===")
		for _, msg := range failures {
			fmt.Printf(" -> %s\n", msg)
		}
		fmt.Println("======================")
		os.Exit(1)
	}
}

func validateInputs(f, s, c, o, t string, p int) (Config, error) {
	if f == "" || s == "" || c == "" {
		return Config{}, fmt.Errorf("missing arguments")
	}

	t = strings.ToLower(t)
	if t != "png" && t != "bmp" {
		return Config{}, fmt.Errorf("invalid type: %s (must be 'png' or 'bmp')", t)
	}

	if p < 0 {
		return Config{}, fmt.Errorf("padding cannot be negative")
	}

	var sizeInts []int
	for _, pStr := range strings.Split(s, ",") {
		val, err := strconv.Atoi(strings.TrimSpace(pStr))
		if err != nil {
			return Config{}, err
		}
		sizeInts = append(sizeInts, val)
	}
	sort.Ints(sizeInts)

	return Config{
		FontPattern: f,
		Sizes:       sizeInts,
		Chars:       c,
		OutputDir:   o,
		Format:      t,
		Padding:     p, // Set padding
	}, nil
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
