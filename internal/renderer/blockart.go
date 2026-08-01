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
// This ratio determines the output row count regardless of how many dot
// samples each cell is subdivided into internally — subdividing further
// (quadrants, braille) adds detail within each row/column, not more of
// them, so this constant doesn't change across techniques.
const terminalCellAspect = 2.0

// brailleDotsWide and brailleDotsTall are the sample-grid dimensions
// encoded by a single Unicode Braille Pattern character (U+2800-U+28FF):
// 2 columns x 4 rows of independently-settable dots per cell. This gives
// double the vertical detail of a 2x2 quadrant-block cell for the same
// terminal row/column count, and Braille Patterns (Unicode since 1.0, in
// active use since long before graphical terminals) have far more
// consistent monospace font coverage than newer sextant/octant glyphs.
const (
	brailleDotsWide = 2
	brailleDotsTall = 4
)

// brailleBit maps a (row, col) position in the 2x4 dot grid to its bit
// index in the Braille Pattern codepoint offset (U+2800 + mask), per the
// standard Unicode dot numbering (columns fill rows 0-2 first, then row 3
// is appended at bits 6-7).
var brailleBit = [brailleDotsTall][brailleDotsWide]uint{
	{0, 3},
	{1, 4},
	{2, 5},
	{6, 7},
}

// rgb is a simple 8-bit color sample used for cell color clustering.
type rgb struct{ r, g, b uint8 }

// renderHalfBlockImage decodes imageData (PNG/JPEG/GIF) and renders it as
// Unicode Braille-pattern ANSI art at targetCols terminal columns wide.
// Each terminal cell encodes an 8-dot (2 wide x 4 tall) grid of pixel
// samples, clustered into a "foreground" (dots on) and "background" (dots
// off) color pair, and rendered as the single Braille character matching
// that dot pattern — the same general terminal-agnostic technique used by
// tools like img2braille/chafa's braille symbol mode for detailed,
// text/line-art-heavy content, at twice the sample density of a 2x2
// quadrant-block cell.
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

	totalSubCols := targetCols * brailleDotsWide
	totalSubRows := rows * brailleDotsTall

	var sb strings.Builder
	var samples [brailleDotsWide * brailleDotsTall]rgb
	for r := 0; r < rows; r++ {
		for c := 0; c < targetCols; c++ {
			for dr := 0; dr < brailleDotsTall; dr++ {
				for dc := 0; dc < brailleDotsWide; dc++ {
					samples[brailleBit[dr][dc]] = averageBoxColor(
						img, bounds,
						c*brailleDotsWide+dc, totalSubCols,
						r*brailleDotsTall+dr, totalSubRows,
					)
				}
			}

			mask, bg, fg := quantizeCell(samples[:])
			glyph := rune(0x2800 + mask)
			fmt.Fprintf(&sb, "\x1b[48;2;%d;%d;%dm\x1b[38;2;%d;%d;%dm%c", bg.r, bg.g, bg.b, fg.r, fg.g, fg.b, glyph)
		}
		sb.WriteString("\x1b[0m")
		if r < rows-1 {
			sb.WriteString("\n")
		}
	}

	return sb.String(), nil
}

// quantizeCell clusters samples into two color groups — the pair with the
// greatest color distance seeds the two clusters, and every other sample
// joins whichever seed it's closer to — returning a bitmask of which
// sample indices joined the second ("foreground") cluster, along with each
// cluster's average color (background = the first seed's cluster,
// foreground = the second's). When all samples are identical (a flat
// region), every sample joins the first cluster, yielding an all-zero mask
// — a blank glyph showing only the solid background color, as intended.
func quantizeCell(samples []rgb) (mask uint16, bg, fg rgb) {
	n := len(samples)
	bestI, bestJ, bestDist := 0, 1, -1
	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			d := colorDistSq(samples[i], samples[j])
			if d > bestDist {
				bestDist = d
				bestI, bestJ = i, j
			}
		}
	}
	seedA, seedB := samples[bestI], samples[bestJ]

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
	return mask, bg, fg
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
