# ==============================================================================
# Variables
# ==============================================================================
BINARY_NAME=ttf2bmp
OUTPUT_DIR=bin
ARTIFACTS_DIR=output
TEST_DATA_DIR=test_data

# Tools Source
TOOL_VERIFIER_SRC=./tools/verifier/main.go
TOOL_VALIDATOR_SRC=./tools/validator/main.go

# Local Build Binaries (Host OS)
TOOL_VERIFIER_BIN=$(OUTPUT_DIR)/verifier
TOOL_VALIDATOR_BIN=$(OUTPUT_DIR)/validator

# ------------------------------------------------------------------------------
# Test Configuration (Override these at runtime!)
# Example: make run TEST_FONT_NAME=MyFont TEST_SIZE=48
# ------------------------------------------------------------------------------
TEST_FONT_NAME ?= Go-Regular
TEST_FONT_URL  ?= https://raw.githubusercontent.com/golang/image/master/font/gofont/ttfs/Go-Regular.ttf
TEST_CHARS     ?= "Abdfghjlpqty123"
TEST_SIZE      ?= 32

# Derived Paths
TEST_FONT_PATH  ?= $(TEST_DATA_DIR)/$(TEST_FONT_NAME).ttf
TEST_OUTPUT_FNT  = $(ARTIFACTS_DIR)/$(TEST_FONT_NAME)-$(TEST_SIZE).fnt

# Go Commands
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOMOD=$(GOCMD) mod
GOVET=$(GOCMD) vet
GOFMT=$(GOCMD) fmt
GOLINT=golangci-lint

# ==============================================================================
# Main Targets
# ==============================================================================

.PHONY: all build clean test bench deps vet lint format run help verify validate fetch-test-data \
        build-cli build-verifier build-validator build-lib build-tools \
        build-linux build-windows build-mac build-all

# Default target: Dependencies -> Code Quality -> Tests -> Build
all: deps check test build

deps:
	@echo "  >  Downloading dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy

# ==============================================================================
# Build Targets (Local Host)
# ==============================================================================

# Master build target: builds everything for the current OS
build: build-lib build-cli build-tools

# 1. Build the Main CLI
build-cli:
	@echo "  >  Building CLI ($(BINARY_NAME))..."
	@mkdir -p $(OUTPUT_DIR)
	$(GOBUILD) -o $(OUTPUT_DIR)/$(BINARY_NAME) ./main.go

# 2. Build the Library (Compilation Check)
build-lib:
	@echo "  >  Verifying Library compilation..."
	$(GOBUILD) ./converter/...

# 3. Build Helper Tools Wrapper
build-tools: build-verifier build-validator

build-verifier:
	@echo "  >  Building Verifier Tool..."
	@mkdir -p $(OUTPUT_DIR)
	$(GOBUILD) -o $(TOOL_VERIFIER_BIN) $(TOOL_VERIFIER_SRC)

build-validator:
	@echo "  >  Building Validator Tool..."
	@mkdir -p $(OUTPUT_DIR)
	$(GOBUILD) -o $(TOOL_VALIDATOR_BIN) $(TOOL_VALIDATOR_SRC)

# ==============================================================================
# Test Data
# ==============================================================================

# Fetch test data if it doesn't exist locally
fetch-test-data:
	@if [ ! -f "$(TEST_FONT_PATH)" ]; then \
		echo "  >  Downloading test font ($(TEST_FONT_NAME))..."; \
		mkdir -p $(TEST_DATA_DIR); \
		curl -L -f -S -o "$(TEST_FONT_PATH)" "$(TEST_FONT_URL)"; \
	else \
		echo "  >  Using existing font at $(TEST_FONT_PATH)"; \
	fi

# ==============================================================================
# Testing & Benchmarking
# ==============================================================================

test:
	@echo "  >  Running tests..."
	$(GOTEST) -v ./converter/...

bench:
	@echo "  >  Running benchmarks..."
	$(GOTEST) -bench=. ./converter/...

# ==============================================================================
# Code Quality
# ==============================================================================

check: vet lint

format:
	@echo "  >  Formatting code..."
	$(GOFMT) ./...

vet:
	@echo "  >  Vetting code..."
	$(GOVET) ./...

lint:
	@echo "  >  Running Linter..."
	$(GOLINT) run ./...

# ==============================================================================
# Execution & Demo
# ==============================================================================

# Run the CLI using the default (PNG) to verify downstream compatibility
run: build-cli fetch-test-data
	@echo "  >  Running Conversion (Default PNG) for $(TEST_FONT_NAME)..."
	@rm -rf output
	@./$(OUTPUT_DIR)/$(BINARY_NAME) \
		-f "test_data/$(TEST_FONT_NAME).ttf" \
		-s "$(TEST_SIZE)" \
		-c $(TEST_CHARS) \
		-o output

verify: build-verifier run
	@echo "  >  Running Visual Verification..."
	./$(TOOL_VERIFIER_BIN) \
		-fnt "$(TEST_OUTPUT_FNT)"

# Validate must use BMP to exercise the custom encoder
validate: build-cli build-validator fetch-test-data
	@echo "  >  Running Validation Pipeline (Force BMP)..."
	@mkdir -p output_val
	@./$(OUTPUT_DIR)/$(BINARY_NAME) \
		-f "test_data/$(TEST_FONT_NAME).ttf" \
		-s "$(TEST_SIZE)" \
		-c $(TEST_CHARS) \
		-t "bmp" \
		-o output_val
	@echo "  >  Verifying output..."
	@./$(OUTPUT_DIR)/validator \
		-ttf "test_data/$(TEST_FONT_NAME).ttf" \
		-fnt "output_val/$(TEST_FONT_NAME)-$(TEST_SIZE).fnt" \
		-chars $(TEST_CHARS) \
		-size $(TEST_SIZE) \
		-out output_val


# ==============================================================================
# Cleanup
# ==============================================================================

clean:
	@echo "  >  Cleaning build cache, artifacts, and test data..."
	$(GOCLEAN)
	rm -rf $(OUTPUT_DIR)
	rm -rf $(ARTIFACTS_DIR)
	rm -rf $(TEST_DATA_DIR)
	rm -f *.png *.fnt *.bmp

# ==============================================================================
# Cross Compilation (Builds CLI + Tools for every platform)
# ==============================================================================

build-linux:
	@echo "  >  Building for Linux (AMD64)..."
	GOOS=linux GOARCH=amd64 $(GOBUILD) -o $(OUTPUT_DIR)/$(BINARY_NAME)-linux ./main.go
	GOOS=linux GOARCH=amd64 $(GOBUILD) -o $(OUTPUT_DIR)/verifier-linux $(TOOL_VERIFIER_SRC)
	GOOS=linux GOARCH=amd64 $(GOBUILD) -o $(OUTPUT_DIR)/validator-linux $(TOOL_VALIDATOR_SRC)

build-windows:
	@echo "  >  Building for Windows (AMD64)..."
	GOOS=windows GOARCH=amd64 $(GOBUILD) -o $(OUTPUT_DIR)/$(BINARY_NAME)-windows.exe ./main.go
	GOOS=windows GOARCH=amd64 $(GOBUILD) -o $(OUTPUT_DIR)/verifier-windows.exe $(TOOL_VERIFIER_SRC)
	GOOS=windows GOARCH=amd64 $(GOBUILD) -o $(OUTPUT_DIR)/validator-windows.exe $(TOOL_VALIDATOR_SRC)

build-mac:
	@echo "  >  Building for MacOS (Intel)..."
	GOOS=darwin GOARCH=amd64 $(GOBUILD) -o $(OUTPUT_DIR)/$(BINARY_NAME)-darwin-amd64 ./main.go
	GOOS=darwin GOARCH=amd64 $(GOBUILD) -o $(OUTPUT_DIR)/verifier-darwin-amd64 $(TOOL_VERIFIER_SRC)
	GOOS=darwin GOARCH=amd64 $(GOBUILD) -o $(OUTPUT_DIR)/validator-darwin-amd64 $(TOOL_VALIDATOR_SRC)
	@echo "  >  Building for MacOS (Apple Silicon)..."
	GOOS=darwin GOARCH=arm64 $(GOBUILD) -o $(OUTPUT_DIR)/$(BINARY_NAME)-darwin-arm64 ./main.go
	GOOS=darwin GOARCH=arm64 $(GOBUILD) -o $(OUTPUT_DIR)/verifier-darwin-arm64 $(TOOL_VERIFIER_SRC)
	GOOS=darwin GOARCH=arm64 $(GOBUILD) -o $(OUTPUT_DIR)/validator-darwin-arm64 $(TOOL_VALIDATOR_SRC)

build-all: build-linux build-mac build-windows

help:
	@echo "Usage: make [target] [VARIABLES]"
	@echo ""
	@echo "Targets:"
	@echo "  all          - Deps, check, test, build (everything)"
	@echo "  build        - Build CLI, Lib check, and Tools (Host OS)"
	@echo "  build-all    - Cross-compile everything for Linux, Win, Mac"
	@echo "  run          - Run conversion (defaults to RobotoSlab)"
	@echo "  verify       - Run conversion + visual check"
	@echo "  validate     - Run conversion + math validation"
	@echo "  test         - Run unit tests"
	@echo "  clean        - Remove all binaries and outputs"
	@echo ""
	@echo "Variables (Override with make <target> VAR=VAL):"
	@echo "  TEST_FONT_NAME  (Default: RobotoSlab-Regular)"
	@echo "  TEST_SIZE       (Default: 32)"
	@echo "  TEST_CHARS      (Default: \"Abdfghjlpqty123\")"