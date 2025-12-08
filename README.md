# ttf2bmp
TTF to Bitmap Font Converter.

`ttf2bmp` is a Go library and command-line tool for converting TrueType Fonts (.ttf) into Bitmap Fonts. It generates a texture atlas (`.png`) and a descriptor file (`.fnt`), compatible with [AngelCode BMFont](https://www.angelcode.com/products/bmfont/) and standard game engines (Unity, Godot, LibGDX, etc.).



## Features

* **Pure Go**: No C bindings, uses `golang.org/x/image` and `freetype`.
* **Customizable**: Set font size, DPI, texture size, and custom character sets.
* **Standard Output**: Generates compliant text-based `.fnt` files.

## Installation

### As a CLI Tool

```bash
go install ./cmd/ttf2bmp