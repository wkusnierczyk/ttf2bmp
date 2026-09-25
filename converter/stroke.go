package converter

import (
	"image"
	"math"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

// strokeSupersample is how finely the glyph interior is measured: the atlas is
// rendered again at this many times the size, and the distance to the glyph edge is
// taken on that grid. 8 gives the inner edge of the outline 64 coverage levels, the
// same order as the antialiasing on the outer edge.
const strokeSupersample = 8

// MinStroke is the narrowest outline hollow can draw: one supersample. The band is
// measured in whole supersamples, so a narrower width would round to no band at all
// and erase the glyph rather than thin it.
const MinStroke = 1.0 / strokeSupersample

// placement is one glyph drawn into the atlas: which, and at what x.
type placement struct {
	char rune
	x    int
}

// hollow removes the interior of every glyph in img, keeping only the band within
// stroke pixels of the glyph edge. The band lies inside the glyph, so the atlas
// layout, the glyph bounds and every metric stay exactly as they are.
//
// img is the filled atlas as Generate draws it, and it keeps its outer edge: the
// result is filled minus interior, where the interior is the set of points more than
// stroke pixels inside the glyph, measured on a supersampled render and box-filtered
// back down. So the outer edge is the rasterizer's own antialiasing, the inner edge
// is antialiased too, and stroke may be fractional.
//
// One glyph at a time, each on a supersampled canvas just large enough for its ink,
// so memory follows the largest glyph rather than the whole atlas.
func hollow(img *image.RGBA, f *opentype.Font, size int, h font.Hinting, placed []placement, ascent int, stroke float64) error {
	s := strokeSupersample
	face, err := opentype.NewFace(f, &opentype.FaceOptions{
		Size:    float64(size * s),
		DPI:     72,
		Hinting: h,
	})
	if err != nil {
		return err
	}
	defer func() { _ = face.Close() }()

	// dist is measured between pixel centres, so a pixel touching the edge is 1 away.
	// Half a pixel of that is the pixel itself, not the band.
	limit := stroke*float64(s) + 0.5
	limitSq := limit * limit

	// Every glyph in the atlas, a character given twice included: the fnt points at the
	// last copy, but the earlier one is in the image too and is hollowed the same way.
	for _, p := range placed {
		char, x := p.char, p.x
		ink, _, ok := face.GlyphBounds(char)
		if !ok {
			continue
		}

		// The glyph's ink in atlas pixels, from the same origin as the filled atlas at s
		// times the size, widened by a pixel so the canvas has outside all round, and
		// clipped to the atlas so a glyph cut by the atlas edge is measured from that
		// edge, as the filled atlas cuts it.
		cell := image.Rect(
			floorDiv(x*s+ink.Min.X.Floor(), s)-1,
			floorDiv(ascent*s+ink.Min.Y.Floor(), s)-1,
			floorDiv(x*s+ink.Max.X.Ceil()+s-1, s)+1,
			floorDiv(ascent*s+ink.Max.Y.Ceil()+s-1, s)+1,
		).Intersect(img.Bounds())
		if cell.Empty() {
			continue
		}

		w, ht := cell.Dx()*s, cell.Dy()*s
		canvas := image.NewAlpha(image.Rect(0, 0, w, ht))
		drawer := &font.Drawer{Dst: canvas, Src: image.White, Face: face}
		drawer.Dot = fixed.P((x-cell.Min.X)*s, (ascent-cell.Min.Y)*s)
		drawer.DrawString(string(char))

		inside := make([]bool, w*ht)
		for i, a := range canvas.Pix {
			inside[i] = a >= 128
		}
		dist := insideDistance(inside, w, ht)

		for cy := 0; cy < cell.Dy(); cy++ {
			for cx := 0; cx < cell.Dx(); cx++ {
				interior := 0
				for sy := 0; sy < s; sy++ {
					row := (cy*s + sy) * w
					for sx := 0; sx < s; sx++ {
						if dist[row+cx*s+sx] >= limitSq {
							interior++
						}
					}
				}
				if interior == 0 {
					continue
				}
				// Coverage of the interior, on the 0-255 scale of the alpha it is taken from.
				cut := (interior*255 + s*s/2) / (s * s)
				i := img.PixOffset(cell.Min.X+cx, cell.Min.Y+cy)
				keep := int(img.Pix[i+3]) - cut
				if keep < 0 {
					keep = 0
				}
				// Generate draws white, so the colour channels are the premultiplied alpha.
				v := uint8(keep)
				img.Pix[i], img.Pix[i+1], img.Pix[i+2], img.Pix[i+3] = v, v, v, v
			}
		}
	}
	return nil
}

// floorDiv is integer division rounding towards negative infinity, for glyph ink that
// reaches left of or above the atlas origin.
func floorDiv(a, b int) int {
	q := a / b
	if a%b != 0 && (a < 0) != (b < 0) {
		q--
	}
	return q
}

// insideDistance returns, for every pixel of a w x h mask, the squared Euclidean
// distance to the nearest pixel that is not inside, and 0 for pixels outside. The
// area beyond the image counts as outside, so a glyph clipped by the atlas edge is
// still measured from that edge.
//
// Exact, in linear time: the two-pass separable transform of Felzenszwalb and
// Huttenlocher, "Distance Transforms of Sampled Functions" (2012).
func insideDistance(inside []bool, w, h int) []float64 {
	// The mask is padded by one pixel of outside on every side, so the image border
	// is an edge. Inside pixels start at "infinity", outside ones at 0.
	pw, ph := w+2, h+2
	big := float64(pw*pw + ph*ph)
	grid := make([]float64, pw*ph)
	for y := 0; y < ph; y++ {
		for x := 0; x < pw; x++ {
			if x > 0 && y > 0 && x <= w && y <= h && inside[(y-1)*w+(x-1)] {
				grid[y*pw+x] = big
			}
		}
	}

	n := pw
	if ph > n {
		n = ph
	}
	f := make([]float64, n)
	d := make([]float64, n)
	v := make([]int, n)
	z := make([]float64, n+1)

	for x := 0; x < pw; x++ {
		for y := 0; y < ph; y++ {
			f[y] = grid[y*pw+x]
		}
		distance1D(f[:ph], d[:ph], v, z)
		for y := 0; y < ph; y++ {
			grid[y*pw+x] = d[y]
		}
	}
	for y := 0; y < ph; y++ {
		copy(f[:pw], grid[y*pw:(y+1)*pw])
		distance1D(f[:pw], d[:pw], v, z)
		copy(grid[y*pw:(y+1)*pw], d[:pw])
	}

	out := make([]float64, w*h)
	for y := 0; y < h; y++ {
		copy(out[y*w:(y+1)*w], grid[(y+1)*pw+1:(y+1)*pw+1+w])
	}
	return out
}

// distance1D is the one-dimensional squared distance transform of f into d: the
// lower envelope of the parabolas rooted at each sample. v and z are scratch space.
func distance1D(f, d []float64, v []int, z []float64) {
	n := len(f)
	k := 0
	v[0] = 0
	z[0] = math.Inf(-1)
	z[1] = math.Inf(1)
	intersect := func(q, p int) float64 {
		return ((f[q] + float64(q*q)) - (f[p] + float64(p*p))) / float64(2*q-2*p)
	}
	for q := 1; q < n; q++ {
		// z[0] is -Inf, so this stops at k == 0 at the latest. The values are finite
		// (insideDistance uses a large number, not Inf, for "inside"), so no Inf - Inf.
		s := intersect(q, v[k])
		for s <= z[k] {
			k--
			s = intersect(q, v[k])
		}
		k++
		v[k] = q
		z[k] = s
		z[k+1] = math.Inf(1)
	}
	k = 0
	for q := 0; q < n; q++ {
		for z[k+1] < float64(q) {
			k++
		}
		dq := float64(q - v[k])
		d[q] = dq*dq + f[v[k]]
	}
}
