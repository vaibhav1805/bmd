package renderer

import (
	"bytes"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"strings"
)

// terminalCellAspect approximates the height:width ratio of a monospace
// terminal character cell (e.g. an 8x16px font is 2:1). Used to scale a
// decoded image so block-art rendering preserves its visual proportions.
const terminalCellAspect = 2.0

// quadrantGlyphs maps a 4-bit mask (bit0=top-left, bit1=top-right,
// bit2=bottom-left, bit3=bottom-right; a set bit means that quadrant
// belongs to the "foreground" color cluster) to the Unicode Block Elements
// glyph covering exactly that set of quadrants. All 16 combinations are
// covered by long-standing, universally-supported codepoints (Unicode 1.0
// Block Elements), unlike newer sextant/octant glyphs that risk missing
// from some monospace fonts.
var quadrantGlyphs = [16]rune{
	0b0000: ' ', // none
	0b0001: '▘', // top-left
	0b0010: '▝', // top-right
	0b0011: '▀', // top-left + top-right
	0b0100: '▖', // bottom-left
	0b0101: '▌', // top-left + bottom-left
	0b0110: '▞', // top-right + bottom-left
	0b0111: '▛', // top-left + top-right + bottom-left
	0b1000: '▗', // bottom-right
	0b1001: '▚', // top-left + bottom-right
	0b1010: '▐', // top-right + bottom-right
	0b1011: '▜', // top-left + top-right + bottom-right
	0b1100: '▄', // bottom-left + bottom-right
	0b1101: '▙', // top-left + bottom-left + bottom-right
	0b1110: '▟', // top-right + bottom-left + bottom-right
	0b1111: '█', // all four
}

// rgb is a simple 8-bit color sample used for quadrant clustering.
type rgb struct{ r, g, b uint8 }

// renderHalfBlockImage decodes imageData (PNG/JPEG/GIF) and renders it as
// Unicode block-art at targetCols terminal columns wide. Each terminal cell
// encodes a 2x2 grid of pixel samples (quadrants), matched to the closest
// pair of a "foreground" and "background" color and rendered with the
// corresponding Block Elements glyph — the same general technique used by
// chafa/timg's higher-quality symbol modes, doubling the effective detail
// of a plain top/bottom half-block over the same terminal cell grid.
//
// Returns an error if imageData can't be decoded as a supported format.
func renderHalfBlockImage(imageData []byte, targetCols int) (string, error) {
	img, _, err := image.Decode(bytes.NewReader(imageData))
	if err != nil {
		return "", fmt.Errorf("decode image: %w", err)
	}
	if targetCols < 1 {
		targetCols = 1
	}

	bounds := img.Bounds()
	srcW, srcH := bounds.Dx(), bounds.Dy()
	if srcW == 0 || srcH == 0 {
		return "", fmt.Errorf("decode image: empty bounds")
	}

	rows := int(float64(targetCols) * float64(srcH) / float64(srcW) / terminalCellAspect)
	if rows < 1 {
		rows = 1
	}

	var sb strings.Builder
	for r := 0; r < rows; r++ {
		for c := 0; c < targetCols; c++ {
			tl := averageBoxColor(img, bounds, c*2, targetCols*2, r*2, rows*2)
			tr := averageBoxColor(img, bounds, c*2+1, targetCols*2, r*2, rows*2)
			bl := averageBoxColor(img, bounds, c*2, targetCols*2, r*2+1, rows*2)
			br := averageBoxColor(img, bounds, c*2+1, targetCols*2, r*2+1, rows*2)

			glyph, bg, fg := quantizeQuadrant(tl, tr, bl, br)
			fmt.Fprintf(&sb, "\x1b[48;2;%d;%d;%dm\x1b[38;2;%d;%d;%dm%c", bg.r, bg.g, bg.b, fg.r, fg.g, fg.b, glyph)
		}
		sb.WriteString("\x1b[0m")
		if r < rows-1 {
			sb.WriteString("\n")
		}
	}

	return sb.String(), nil
}

// quantizeQuadrant clusters the 4 quadrant samples into two color groups —
// the pair with the greatest color distance seeds the two clusters, and
// each sample joins whichever seed it's closer to — then returns the
// Block Elements glyph for the resulting quadrant pattern along with each
// cluster's average color (background = the first seed's cluster,
// foreground = the second's). When all 4 samples are identical (a flat
// region), every sample joins the first cluster, yielding a plain space
// glyph — visually just the solid background color, as intended.
func quantizeQuadrant(tl, tr, bl, br rgb) (glyph rune, bg, fg rgb) {
	samples := [4]rgb{tl, tr, bl, br}

	bestI, bestJ, bestDist := 0, 1, -1
	for i := 0; i < 4; i++ {
		for j := i + 1; j < 4; j++ {
			d := colorDistSq(samples[i], samples[j])
			if d > bestDist {
				bestDist = d
				bestI, bestJ = i, j
			}
		}
	}
	seedA, seedB := samples[bestI], samples[bestJ]

	var mask uint8
	var aSum, bSum [3]uint64
	var aCount, bCount uint64
	for i, s := range samples {
		if colorDistSq(s, seedA) <= colorDistSq(s, seedB) {
			aSum[0] += uint64(s.r)
			aSum[1] += uint64(s.g)
			aSum[2] += uint64(s.b)
			aCount++
		} else {
			mask |= 1 << uint(i)
			bSum[0] += uint64(s.r)
			bSum[1] += uint64(s.g)
			bSum[2] += uint64(s.b)
			bCount++
		}
	}

	bg = rgb{uint8(aSum[0] / aCount), uint8(aSum[1] / aCount), uint8(aSum[2] / aCount)}
	if bCount == 0 {
		fg = bg
	} else {
		fg = rgb{uint8(bSum[0] / bCount), uint8(bSum[1] / bCount), uint8(bSum[2] / bCount)}
	}
	return quadrantGlyphs[mask], bg, fg
}

func colorDistSq(a, b rgb) int {
	dr := int(a.r) - int(b.r)
	dg := int(a.g) - int(b.g)
	db := int(a.b) - int(b.b)
	return dr*dr + dg*dg + db*db
}

// averageBoxColor box-averages the source pixels falling under output
// sub-cell (col, subRow) — where the source image is conceptually divided
// into a totalCols x totalSubRows grid — composited over a black background
// to handle transparency.
func averageBoxColor(img image.Image, bounds image.Rectangle, col, totalCols, subRow, totalSubRows int) rgb {
	srcW, srcH := bounds.Dx(), bounds.Dy()

	xStart := bounds.Min.X + col*srcW/totalCols
	xEnd := bounds.Min.X + (col+1)*srcW/totalCols
	yStart := bounds.Min.Y + subRow*srcH/totalSubRows
	yEnd := bounds.Min.Y + (subRow+1)*srcH/totalSubRows
	if xEnd <= xStart {
		xEnd = xStart + 1
	}
	if yEnd <= yStart {
		yEnd = yStart + 1
	}

	var rSum, gSum, bSum uint64
	count := uint64(0)
	for y := yStart; y < yEnd && y < bounds.Max.Y; y++ {
		for x := xStart; x < xEnd && x < bounds.Max.X; x++ {
			// color.Color.RGBA() returns 16-bit alpha-premultiplied
			// components; for a fully-transparent pixel these are already
			// (0,0,0), which is exactly "composited over black" — no
			// separate un-premultiply/composite step needed, and every
			// pixel (including transparent ones) must count toward the
			// average so a mostly-transparent region reads as mostly black
			// rather than an artificially brightened average of the few
			// opaque pixels.
			r, g, b, _ := img.At(x, y).RGBA()
			rSum += uint64(r) * 255 / 65535
			gSum += uint64(g) * 255 / 65535
			bSum += uint64(b) * 255 / 65535
			count++
		}
	}
	if count == 0 {
		return rgb{}
	}
	return rgb{uint8(rSum / count), uint8(gSum / count), uint8(bSum / count)}
}
