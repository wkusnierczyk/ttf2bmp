# ttf2bmp
TTF to Bitmap Font Converter.

# ttf2bmp

[![CI Status](https://github.com/your-username/repo-name/actions/workflows/ci.yml/badge.svg)](https://github.com/your-username/repo-name/actions)
[![Go Report Card](https://goreportcard.com/badge/github.com/your-username/repo-name)](https://goreportcard.com/report/github.com/your-username/repo-name)
[![Go Reference](https://pkg.go.dev/badge/github.com/your-username/repo-name.svg)](https://pkg.go.dev/github.com/your-username/repo-name)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](https://opensource.org/licenses/MIT)

`ttf2bmp` is a Go library and command-line tool...

`ttf2bmp` is a Go library and command-line tool for converting TrueType Fonts (.ttf) into Bitmap Fonts. It generates a texture atlas (`.png`) and a descriptor file (`.fnt`), compatible with [AngelCode BMFont](https://www.angelcode.com/products/bmfont/) and standard game engines (Unity, Godot, LibGDX, etc.).



## Features

* **Pure Go**: No C bindings, uses `golang.org/x/image` and `freetype`.
* **Customizable**: Set font size, DPI, texture size, and custom character sets.
* **Standard Output**: Generates compliant text-based `.fnt` files.

## Installation

### As a CLI Tool

```bash
go install ./cmd/ttf2bmp