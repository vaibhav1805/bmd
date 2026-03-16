package renderer

import (
	"strings"

	"github.com/bmd/bmd/internal/ast"
	"github.com/bmd/bmd/internal/theme"
)

// RenderTable renders a markdown table with box-drawing characters.
// Calculates column widths to align all cells, distinguishes the header row,
// and honors column alignment (left, right, center).
func (r *Renderer) RenderTable(t *ast.Table) string {
	// --- Step 1: Collect all rows and their raw (ANSI-stripped) cell content ---
	type cellData struct {
		rendered string // rendered string (may contain ANSI)
		visible  int    // visible character width
	}
	type rowData struct {
		cells    []cellData
		isHeader bool
	}

	var rows []rowData
	for _, child := range t.Children() {
		row, ok := child.(*ast.TableRow)
		if !ok {
			continue
		}
		var rd rowData
		rd.isHeader = row.IsHeader
		for _, cellChild := range row.Children() {
			cell, ok := cellChild.(*ast.TableCell)
			if !ok {
				continue
			}
			rendered := r.renderInlineChildren(cell.Children())
			vis := visibleLength(rendered)
			rd.cells = append(rd.cells, cellData{rendered: rendered, visible: vis})
		}
		rows = append(rows, rd)
	}

	if len(rows) == 0 {
		return ""
	}

	// --- Step 2: Determine number of columns and max width per column ---
	numCols := 0
	for _, row := range rows {
		if len(row.cells) > numCols {
			numCols = len(row.cells)
		}
	}
	if numCols == 0 {
		return ""
	}

	colWidths := make([]int, numCols)
	for _, row := range rows {
		for ci, cell := range row.cells {
			if cell.visible > colWidths[ci] {
				colWidths[ci] = cell.visible
			}
		}
	}
	// Minimum column width of 3 for readability
	for ci := range colWidths {
		if colWidths[ci] < 3 {
			colWidths[ci] = 3
		}
	}

	// --- Step 3: Collect column alignments from Table.Alignments ---
	alignments := t.Alignments
	// Pad alignments slice to numCols if needed
	for len(alignments) < numCols {
		alignments = append(alignments, "")
	}

	// --- Step 4: Helper functions ---
	borderColor := theme.FgCode(r.theme.TableBorderColor())
	headerColor := "\x1b[1m" // bold for header text
	reset := theme.Reset

	// buildHLine creates a horizontal separator line with corner/tee chars.
	// left, mid, right, fill are box-drawing characters.
	buildHLine := func(left, mid, right, fill string) string {
		var sb strings.Builder
		sb.WriteString(borderColor + left)
		for ci, w := range colWidths {
			sb.WriteString(strings.Repeat(fill, w+2)) // +2 for cell padding
			if ci < numCols-1 {
				sb.WriteString(mid)
			}
		}
		sb.WriteString(right + reset)
		return sb.String()
	}

	// padCell pads/aligns a cell value to the target width.
	padCell := func(rendered string, visible int, width int, alignment string) string {
		space := width - visible
		if space < 0 {
			space = 0
		}
		switch alignment {
		case "right":
			return strings.Repeat(" ", space) + rendered
		case "center":
			left := space / 2
			right := space - left
			return strings.Repeat(" ", left) + rendered + strings.Repeat(" ", right)
		default: // left or empty
			return rendered + strings.Repeat(" ", space)
		}
	}

	// buildRow renders a single table row with │ borders between cells.
	buildRow := func(row rowData, isHeader bool) string {
		var sb strings.Builder
		sb.WriteString(borderColor + "│" + reset)
		for ci := 0; ci < numCols; ci++ {
			var rendered string
			var vis int
			if ci < len(row.cells) {
				rendered = row.cells[ci].rendered
				vis = row.cells[ci].visible
			}
			alignment := ""
			if ci < len(alignments) {
				alignment = alignments[ci]
			}
			padded := padCell(rendered, vis, colWidths[ci], alignment)
			if isHeader {
				sb.WriteString(" " + headerColor + padded + reset + " ")
			} else {
				sb.WriteString(" " + padded + " ")
			}
			sb.WriteString(borderColor + "│" + reset)
		}
		return sb.String()
	}

	// --- Step 5: Render table ---
	var sb strings.Builder
	sb.WriteString("\n")

	topLine := buildHLine("┌", "┬", "┐", "─")
	sb.WriteString(topLine + "\n")

	for i, row := range rows {
		sb.WriteString(buildRow(row, row.isHeader) + "\n")

		if row.isHeader && i < len(rows)-1 {
			// After header row: separator with tee chars
			sepLine := buildHLine("├", "┼", "┤", "─")
			sb.WriteString(sepLine + "\n")
		}
	}

	bottomLine := buildHLine("└", "┴", "┘", "─")
	sb.WriteString(bottomLine)

	return sb.String()
}
