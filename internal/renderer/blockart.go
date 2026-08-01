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
// 2 columns x 4 rows of independently-settable dots per cell. Braille
// Patterns have been part of Unicode since 1.0 and have far more
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

// rgb is a simple 8-bit color sample.
type rgb struct{ r, g, b uint8 }

// luminance returns the standard perceptual grayscale weighting of c.
func luminance(c rgb) float64 {
	return 0.299*float64(c.r) + 0.587*float64(c.g) + 0.114*float64(c.b)
}

// renderHalfBlockImage decodes imageData (PNG/JPEG/GIF) and renders it as
// Unicode Braille-pattern ANSI art at targetCols terminal columns wide.
//
// This mirrors how mature terminal-art tools (chafa, dedicated
// image-to-Braille converters) actually do it: convert to grayscale, then
// binarize the WHOLE image at once via Floyd-Steinberg error-diffusion
// dithering (not an independent decision per cell), and only then slice
// the result into 2x4 dot groups per terminal cell. An earlier version of
// this renderer instead re-clustered each cell's 8 raw color samples into
// two groups from scratch, with no relationship between cells — on a
// flat, ~uniform region (like a mostly-white photo/screenshot background)
// that always found *some* two "most different" samples among ordinary
// sampling noise and forced a high-contrast split there too, so the whole
// image looked like static rather than a coherent shape. Diffusing the
// quantization error forward instead keeps the dot pattern's local density
// tracking the true local brightness, which is what makes dithered output
// read as a recognizable image instead of noise.
//
// Per-cell color still varies (background = average color of the cell's
// "paper" dots, foreground = average of its "ink" dots) since which
// samples end up "ink" vs "paper" now comes from that globally-consistent
// classification rather than an arbitrary local RGB split.
//
// Note: like all terminal character-art techniques, this has a hard
// resolution ceiling — small printed text in a screenshot will not become
// legible this way (a page's letterforms are often only a few source
// pixels tall after downscaling to ~100 terminal columns). This renders a
// recognizable shape/photo/diagram preview, not a document reader; for
// reading actual text out of an image, see the saved-temp-file fallback.
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

	dotCols := targetCols * brailleDotsWide
	dotRows := rows * brailleDotsTall

	colorGrid, grayGrid := sampleGrids(img, bounds, dotCols, dotRows)
	inkGrid := ditherFloydSteinberg(grayGrid, dotCols, dotRows)

	var sb strings.Builder
	for r := 0; r < rows; r++ {
		for c := 0; c < targetCols; c++ {
			mask, bg, fg := cellFromDots(colorGrid, inkGrid, dotCols, c, r)
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

// sampleGrids box-averages img down to a dotCols x dotRows grid, returning
// both the sampled colors (for later fg/bg averaging) and their luminance
// (as the mutable working buffer for dithering).
func sampleGrids(img image.Image, bounds image.Rectangle, dotCols, dotRows int) (colorGrid [][]rgb, grayGrid [][]float64) {
	colorGrid = make([][]rgb, dotRows)
	grayGrid = make([][]float64, dotRows)
	for y := 0; y < dotRows; y++ {
		colorGrid[y] = make([]rgb, dotCols)
		grayGrid[y] = make([]float64, dotCols)
		for x := 0; x < dotCols; x++ {
			c := averageBoxColor(img, bounds, x, dotCols, y, dotRows)
			colorGrid[y][x] = c
			grayGrid[y][x] = luminance(c)
		}
	}
	return colorGrid, grayGrid
}

// ditherFloydSteinberg binarizes grayGrid (mutated in place as the
// diffusion working buffer) into a dotCols x dotRows grid marking which
// dots are "ink" (should render as a visible Braille dot) vs "paper"
// (background only, dot off).
//
// Standard Floyd-Steinberg: each pixel is quantized to the nearer of the
// two target levels (0/255), the resulting error is pushed forward into
// not-yet-visited neighbors (right 7/16, below-left 3/16, below 5/16,
// below-right 1/16), and processing continues in scan order. This keeps
// locally-averaged brightness correct even though each individual pixel's
// decision is a hard binary choice, which is what makes the dithered
// pattern read as a smooth gradient/shape instead of speckled noise.
//
// "Ink" is defined as whichever of the two quantization levels is the
// image-wide minority (so, e.g., a mostly-white page's sparse dark text
// becomes ink, and equally a mostly-dark UI screenshot's sparse bright
// text becomes ink) — this keeps large flat majority regions rendering as
// blank/background instead of densely speckled.
func ditherFloydSteinberg(grayGrid [][]float64, dotCols, dotRows int) [][]bool {
	var sum float64
	for y := 0; y < dotRows; y++ {
		for x := 0; x < dotCols; x++ {
			sum += grayGrid[y][x]
		}
	}
	meanLuminance := sum / float64(dotRows*dotCols)
	darkIsInk := meanLuminance >= 127.5 // light is the majority, so dark is minority/ink

	inkGrid := make([][]bool, dotRows)
	for y := range inkGrid {
		inkGrid[y] = make([]bool, dotCols)
	}

	for y := 0; y < dotRows; y++ {
		for x := 0; x < dotCols; x++ {
			old := grayGrid[y][x]
			isDark := old < 127.5
			var quantized float64
			if isDark {
				quantized = 0
			} else {
				quantized = 255
			}
			inkGrid[y][x] = isDark == darkIsInk

			errVal := old - quantized
			if x+1 < dotCols {
				grayGrid[y][x+1] += errVal * 7.0 / 16
			}
			if y+1 < dotRows {
				if x-1 >= 0 {
					grayGrid[y+1][x-1] += errVal * 3.0 / 16
				}
				grayGrid[y+1][x] += errVal * 5.0 / 16
				if x+1 < dotCols {
					grayGrid[y+1][x+1] += errVal * 1.0 / 16
				}
			}
		}
	}

	return inkGrid
}

// cellFromDots builds the Braille dot mask and fg/bg colors for the
// terminal cell at (col, row), from the already-dithered ink/paper
// classification and sampled colors of its 2x4 dot block.
func cellFromDots(colorGrid [][]rgb, inkGrid [][]bool, dotCols, col, row int) (mask uint16, bg, fg rgb) {
	var inkSum, paperSum [3]uint64
	var inkCount, paperCount uint64

	for dr := 0; dr < brailleDotsTall; dr++ {
		for dc := 0; dc < brailleDotsWide; dc++ {
			x := col*brailleDotsWide + dc
			y := row*brailleDotsTall + dr
			c := colorGrid[y][x]
			if inkGrid[y][x] {
				mask |= 1 << brailleBit[dr][dc]
				inkSum[0] += uint64(c.r)
				inkSum[1] += uint64(c.g)
				inkSum[2] += uint64(c.b)
				inkCount++
			} else {
				paperSum[0] += uint64(c.r)
				paperSum[1] += uint64(c.g)
				paperSum[2] += uint64(c.b)
				paperCount++
			}
		}
	}

	if paperCount > 0 {
		bg = rgb{uint8(paperSum[0] / paperCount), uint8(paperSum[1] / paperCount), uint8(paperSum[2] / paperCount)}
	}
	if inkCount > 0 {
		fg = rgb{uint8(inkSum[0] / inkCount), uint8(inkSum[1] / inkCount), uint8(inkSum[2] / inkCount)}
	} else {
		fg = bg
	}
	if paperCount == 0 {
		bg = fg
	}
	return mask, bg, fg
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
