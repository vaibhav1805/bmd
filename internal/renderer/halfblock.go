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
// decoded image so half-block rendering preserves its visual proportions.
const terminalCellAspect = 2.0

// renderHalfBlockImage decodes imageData (PNG/JPEG/GIF) and renders it as
// Unicode half-block ANSI art at targetCols terminal columns wide. This is
// the technique used by pixterm/viu/chafa's terminal-agnostic fallback:
// each character cell encodes 2 vertical pixel samples via the U+2584
// lower-half-block glyph, with the top sample painted as the background
// color and the bottom sample as the foreground color, using 24-bit
// truecolor ANSI SGR codes.
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
	subRows := rows * 2

	var sb strings.Builder
	for r := 0; r < rows; r++ {
		for c := 0; c < targetCols; c++ {
			topR, topG, topB := averageBoxColor(img, bounds, c, targetCols, r*2, subRows)
			botR, botG, botB := averageBoxColor(img, bounds, c, targetCols, r*2+1, subRows)
			fmt.Fprintf(&sb, "\x1b[48;2;%d;%d;%dm\x1b[38;2;%d;%d;%dm▄", topR, topG, topB, botR, botG, botB)
		}
		sb.WriteString("\x1b[0m")
		if r < rows-1 {
			sb.WriteString("\n")
		}
	}

	return sb.String(), nil
}

// averageBoxColor box-averages the source pixels falling under output
// cell (col, subRow) — where the source image is conceptually divided into
// a totalCols x totalSubRows grid — composited over a black background to
// handle transparency, and returns the resulting 8-bit RGB values.
func averageBoxColor(img image.Image, bounds image.Rectangle, col, totalCols, subRow, totalSubRows int) (uint8, uint8, uint8) {
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
		return 0, 0, 0
	}
	return uint8(rSum / count), uint8(gSum / count), uint8(bSum / count)
}
