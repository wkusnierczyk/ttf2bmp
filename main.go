package main

import (
	"flag"
	"fmt"
	"math"
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
	Padding     int
	Hinting     string // New field
	Stroke      float64
}

var logBuffer []string

func main() {
	var fontsFlag, sizesFlag, charsFlag, outDir, typeFlag, hintingFlag string
	var paddingFlag int
	var strokeFlag float64
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
	flag.IntVar(&paddingFlag, "padding", 2, "Padding between characters (pixels)")
	flag.IntVar(&paddingFlag, "p", 2, "Short for --padding")

	// NEW: Hinting flag
	flag.StringVar(&hintingFlag, "hinting", "full", "Hinting: 'none' (smooth) or 'full' (crisp)")
	flag.StringVar(&hintingFlag, "h", "full", "Short for --hinting")

	flag.Float64Var(&strokeFlag, "stroke", 0, "Outline width in pixels, may be fractional; 0 draws filled glyphs")
	flag.Float64Var(&strokeFlag, "w", 0, "Short for --stroke")

	flag.BoolVar(&showVersion, "version", false, "Print version")

	flag.Parse()

	if showVersion {
		fmt.Printf("ttf2bmp version %s\n", Version)
		os.Exit(0)
	}

	cfg, err := validateInputs(fontsFlag, sizesFlag, charsFlag, outDir, typeFlag, paddingFlag, hintingFlag, strokeFlag)
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
			outPrefix := filepath.Join(cfg.OutputDir, fmt.Sprintf("%s-%d%s", nameNoExt, size, strokeSuffix(cfg.Stroke)))

			msg := fmt.Sprintf("Processing %s @ %dpx (pad:%d, hint:%s, stroke:%g)...", baseName, size, cfg.Padding, cfg.Hinting, cfg.Stroke)
			updateUI(currentJob, totalJobs, msg)

			// Pass Hinting
			err := converter.Generate(fontPath, size, cfg.Chars, outPrefix, cfg.Format, cfg.Padding, cfg.Hinting, cfg.Stroke)

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

func validateInputs(f, s, c, o, t string, p int, h string, w float64) (Config, error) {
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

	h = strings.ToLower(h)
	if h != "none" && h != "vertical" && h != "full" {
		return Config{}, fmt.Errorf("invalid hinting: %s (use 'none', 'vertical', 'full')", h)
	}

	if w != 0 && !(w >= converter.MinStroke && !math.IsInf(w, 0)) {
		return Config{}, fmt.Errorf("invalid stroke: %v (must be 0 for filled glyphs, or at least %v pixels)", w, converter.MinStroke)
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
		Padding:     p,
		Hinting:     h, // Set hinting
		Stroke:      w,
	}, nil
}

// strokeSuffix names the output of a hollow font apart from the filled one of the same
// face and size, so the two can share a directory: "-stroke1", "-stroke0p77". The
// decimal point is written as "p" so that the only dot in the file name is the one
// before the extension.
func strokeSuffix(w float64) string {
	if w <= 0 {
		return ""
	}
	return "-stroke" + strings.Replace(strconv.FormatFloat(w, 'f', -1, 64), ".", "p", 1)
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
