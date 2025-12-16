# ttf2bmp

[![Code Quality](https://github.com/yourusername/ttf2bmp/actions/workflows/test_code_quality.yml/badge.svg)](https://github.com/yourusername/ttf2bmp/actions/workflows/test_code_quality.yml)
[![Functional Verification](https://github.com/yourusername/ttf2bmp/actions/workflows/test_functional_integration.yml/badge.svg)](https://github.com/yourusername/ttf2bmp/actions/workflows/test_functional_integration.yml)

**ttf2bmp** is a robust command-line tool written in Go that converts TrueType Fonts (TTF) into AngelCode BMFont format (BMP image + FNT descriptor). It is designed for high-volume batch processing, featuring a rolling progress dashboard, parallel-ready architecture, and automated regression testing.

## ✨ Features

* **Batch Processing**: Accepts glob patterns (e.g., `fonts/*.ttf`) to process hundreds of fonts in one go.
* **Multi-Size Support**: Generate multiple font sizes (e.g., 12, 24, 32px) in a single run.
* **Smart Dashboard**: A rolling UI providing real-time progress bars and log windows without cluttering the terminal.
* **Verification Suite**: Includes built-in tools for visual inspection and pixel-perfect regression testing against the original TTF.
* **Cross-Platform**: Compiles for Linux, Windows, and macOS (Intel/Apple Silicon) with zero dependencies.

## 🚀 Installation

### Prerequisites
* [Go 1.25](https://go.dev/dl/) or higher.

### Build from Source
```bash
git clone [https://github.com/yourusername/ttf2bmp.git](https://github.com/yourusername/ttf2bmp.git)
cd ttf2bmp
make build
```
The binary will be placed in `bin/ttf2bmp`.

## 📖 Usage

Run the tool using the flags below. The font pattern must be quoted to prevent shell expansion if using wildcards.

```bash
./bin/ttf2bmp [options]
```

| Flag | Short | Description | Required | Example |
| :--- | :--- | :--- | :--- | :--- |
| `--fonts` | `-f` | Glob pattern for input fonts | Yes | `"assets/*.ttf"` |
| `--sizes` | `-s` | Comma-separated list of sizes | Yes | `"16, 24, 32"` |
| `--chars` | `-c` | String of characters to include | Yes | `"ABCabc123"` |
| `--out` | `-o` | Output directory | No (Default: `.`) | `build/fonts` |

### Example
```bash
./bin/ttf2bmp -f "assets/fonts/*.ttf" -s "12,24" -c "Hello World" -o output/
```

## 📂 Project Structure

The project is organized into a modular structure separating the CLI, the core library, and the verification tools.

```text
/ttf2bmp
  ├── main.go                # Main CLI entry point (Batch Processor & UI)
  ├── converter/             # Core Library
  │   ├── lib.go             # Font rendering & FNT generation logic
  │   └── lib_test.go        # Unit tests & Benchmarks
  ├── tools/                 # Quality Assurance Tools
  │   ├── verifier/          # Visual Inspector (FNT -> PNG)
  │   │   └── main.go
  │   └── validator/         # Regression Tester (TTF vs FNT Diff)
  │       └── main.go
  ├── .github/workflows/     # CI Pipelines
  └── Makefile               # Build & Test Automation
```

## 🛠 Development & Testing

We use a comprehensive `Makefile` to manage builds, tests, and verification.

### Common Commands

| Command | Description |
| :--- | :--- |
| `make all` | Runs dependencies, static checks, tests, and builds the binary. |
| `make build` | Compiles the main CLI to `bin/ttf2bmp`. |
| `make test` | Runs unit tests for the core converter library. |
| `make bench` | Runs performance benchmarks. |
| `make check` | Runs `go vet` and `golangci-lint` (Static Analysis). |
| `make clean` | Removes build artifacts and output directories. |

### Verification Tools

This project includes two specialized tools in the `tools/` directory to ensure rendering quality.

#### 1. Visual Verifier (`make verify`)
Reads a generated `.fnt` and `.bmp` pair and renders them onto a single PNG canvas for easy manual inspection.
```bash
make verify -- -fnt output/arial-16.fnt
# Output: output/arial-16_verify.png
```

#### 2. Regression Validator (`make validate`)
Mathematically compares the output of the **Generated FNT/BMP** against the **Original TTF** using Go's native FreeType rendering. It calculates a "Difference Score" and fails if it exceeds the tolerance.
```bash
make validate -- -ttf arial.ttf -fnt output/arial-16.fnt -chars "ABC" -size 16
# Output: PASS / FAIL + debug_diff.png showing mismatches in red.
```

## 📦 Cross-Compilation

To build binaries for all supported platforms at once:
```bash
make build-all
```
Artifacts created in `bin/`:
* `ttf2bmp-linux`
* `ttf2bmp-windows.exe`
* `ttf2bmp-darwin-amd64` (Intel Mac)
* `ttf2bmp-darwin-arm64` (Apple Silicon)

## 🤖 CI/CD Pipelines

We use GitHub Actions to enforce quality standards:

1.  **Code Quality** (`test_code_quality.yml`):
    * Runs `golangci-lint`.
    * Runs unit tests (`go test -race`).
    * Runs benchmarks.

2.  **Functional Verification** (`test_functional_integration.yml`):
    * Builds the CLI.
    * Downloads a real open-source font (Roboto).
    * Runs the `ttf2bmp` tool to generate assets.
    * Runs the **Validator** tool to ensure the output matches the original TTF mathematically.
    * Uploads debug images as artifacts if the test fails.

## License
[MIT](LICENSE)