# Variables
BINARY_NAME=ttf2bmp
# TODO: may need update
CMD_PATH=./main.go
OUTPUT_DIR=bin

# Go related variables
GOBASE=$(shell pwd)
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod

# .PHONY ensures these targets are treated as commands, not files
.PHONY: all build clean test run deps build-linux build-windows build-mac build-all help

all: check test build

# Install dependencies
deps:
	$(GOMOD) download
	$(GOMOD) tidy

# Build the binary for the local architecture
build:
	@echo "  >  Building binary..."
	@mkdir -p $(OUTPUT_DIR)
	$(GOBUILD) -o $(OUTPUT_DIR)/$(BINARY_NAME) $(CMD_PATH)
	@echo "  >  Done! Binary located at $(OUTPUT_DIR)/$(BINARY_NAME)"

# Run unit tests
test:
	@echo "  >  Running tests..."
	$(GOTEST) -v .

# Run benchamrks
bench:
	@echo "  >  Running benchmarks..."
	$(GOTEST) -bench=. .

# Formats code and checks for common errors
check:
	@echo "  >  Formatting code..."
	$(GOCMD) fmt . ./cmd/ttf2bmp
	@echo "  >  Vetting code..."
	$(GOCMD) vet .  ./cmd/ttf2bmp

# Run linter
lint:
	@echo "Running Linter..."
	# Checks code style, logic errors, and complexity
	# Requires: https://golangci-lint.run/usage/install/
	golangci-lint run

# Run the tool (example usage)
run: build
	@echo "  >  Running $(BINARY_NAME)..."
	./$(OUTPUT_DIR)/$(BINARY_NAME) -help

# Clean build artifacts
clean:
	@echo "  >  Cleaning build cache and binaries..."
	$(GOCLEAN)
	rm -rf $(OUTPUT_DIR)
	rm -f *.png *.fnt

help:
	@echo "Make options:"
	@echo "  make          - Run tests and build local binary"
	@echo "  make deps     - Download dependencies (go mod tidy)"
	@echo "  make build    - Build binary to ./bin"
	@echo "  make test     - Run unit tests and benchmarks"
	@echo "  make clean    - Remove binary and output files"
	@echo "  make build-linux   - Cross-compile for Linux"
	@echo "  make build-mac - Cross-compile for MacOS"
	@echo "  make build-windows - Cross-compile for Windows"


# --- Cross Compilation Targets ---

# Build for Linux (AMD64)
build-linux:
	@echo "  >  Building for Linux..."
	GOOS=linux GOARCH=amd64 $(GOBUILD) -o $(OUTPUT_DIR)/$(BINARY_NAME)-linux $(CMD_PATH)

# Build for Windows (AMD64)
build-windows:
	@echo "  >  Building for Windows..."
	GOOS=windows GOARCH=amd64 $(GOBUILD) -o $(OUTPUT_DIR)/$(BINARY_NAME)-windows.exe $(CMD_PATH)

# Build for MacOS (Apple Silicon & Intel)
build-mac:
	@echo "  >  Building for MacOS (Intel)..."
	GOOS=darwin GOARCH=amd64 $(GOBUILD) -o $(OUTPUT_DIR)/$(BINARY_NAME)-darwin-amd64 $(CMD_PATH)
	@echo "  >  Building for MacOS (M1/M2)..."
	GOOS=darwin GOARCH=arm64 $(GOBUILD) -o $(OUTPUT_DIR)/$(BINARY_NAME)-darwin-arm64 $(CMD_PATH)

build-all: build-linux build-mac build-windows
