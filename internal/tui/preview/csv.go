package preview

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"strings"
)

// renderCSV formats a CSV file as an aligned text table.
func renderCSV(data []byte, width int) (string, error) {
	r := csv.NewReader(bytes.NewReader(data))
	records, err := r.ReadAll()
	if err != nil {
		return renderText(data) // fallback to plain text
	}

	if len(records) == 0 {
		return "(empty CSV)", nil
	}

	cols := len(records[0])
	colW := make([]int, cols)
	for _, row := range records {
		for j, cell := range row {
			if j < cols && len(cell) > colW[j] {
				colW[j] = len(cell)
			}
		}
	}

	// Clamp total width to terminal width.
	totalW := cols - 1
	for _, w := range colW {
		totalW += w + 2
	}
	if totalW > width && width > 0 {
		excess := totalW - width
		for i := range colW {
			if excess <= 0 {
				break
			}
			if colW[i] > 8 {
				trim := colW[i] - 8
				if trim > excess {
					trim = excess
				}
				colW[i] -= trim
				excess -= trim
			}
		}
	}

	var sb strings.Builder
	for rowIdx, row := range records {
		for j := 0; j < cols; j++ {
			cell := ""
			if j < len(row) {
				cell = row[j]
			}
			if len(cell) > colW[j] {
				cell = cell[:colW[j]-1] + "…"
			}
			fmt.Fprintf(&sb, "%-*s", colW[j]+2, cell)
			if j < cols-1 {
				sb.WriteString("|")
			}
		}
		sb.WriteString("\n")
		// Draw separator after header row.
		if rowIdx == 0 {
			for j := 0; j < cols; j++ {
				sb.WriteString(strings.Repeat("-", colW[j]+2))
				if j < cols-1 {
					sb.WriteString("+")
				}
			}
			sb.WriteString("\n")
		}
	}
	return sb.String(), nil
}
