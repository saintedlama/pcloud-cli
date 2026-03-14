package preview

import (
	"bytes"
	"fmt"
	"math"
	"os"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/ledongthuc/pdf"
)

// renderPDF extracts styled text from a PDF using GetStyledTexts and maps each
// run's font size and name to a terminal style. Falls back to GetPlainText when
// styled extraction returns no results.
func renderPDF(data []byte) (result string, err error) {
	defer func() {
		if r := recover(); r != nil {
			result = ""
			err = fmt.Errorf("pdf decode failed: %v", r)
		}
	}()

	tmp, err := os.CreateTemp("", "pcloud-preview-*.pdf")
	if err != nil {
		return "", fmt.Errorf("pdf temp file: %w", err)
	}
	defer os.Remove(tmp.Name())
	defer tmp.Close()

	if _, err := tmp.Write(data); err != nil {
		return "", fmt.Errorf("pdf write temp: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		return "", fmt.Errorf("pdf sync temp: %w", err)
	}

	f, reader, err := pdf.Open(tmp.Name())
	if err != nil {
		return "", fmt.Errorf("pdf open: %w", err)
	}
	defer f.Close()

	sentences, styledErr := reader.GetStyledTexts()
	if styledErr != nil || len(sentences) == 0 {
		textReader, err2 := reader.GetPlainText()
		if err2 != nil {
			if styledErr != nil {
				return "", fmt.Errorf("pdf text extraction: %w", styledErr)
			}
			return "", fmt.Errorf("pdf text extraction: %w", err2)
		}
		var buf bytes.Buffer
		if _, err2 := buf.ReadFrom(textReader); err2 != nil {
			return "", fmt.Errorf("pdf read: %w", err2)
		}
		return buf.String(), nil
	}

	return pdfStyledRender(sentences), nil
}

// pdfStyledRender converts a slice of styled PDF text runs into a terminal
// string. Line and paragraph breaks are inferred from Y-coordinate deltas.
func pdfStyledRender(sentences []pdf.Text) string {
	if len(sentences) == 0 {
		return ""
	}

	bodySize := pdfBodySize(sentences)
	lineHeight := bodySize * 1.4

	var sb strings.Builder
	prevY := sentences[0].Y

	for _, s := range sentences {
		// In PDF coordinates Y increases bottom-to-top, so going down the page
		// means Y decreases. A jump back up signals a new page or column.
		dy := prevY - s.Y
		switch {
		case dy < -lineHeight*0.5:
			// Y increased: new page / column — paragraph gap.
			sb.WriteString("\n\n")
		case math.Abs(dy) > lineHeight*1.8:
			// Large downward gap: blank line between paragraphs.
			sb.WriteString("\n\n")
		case math.Abs(dy) > 2.0:
			// Normal line advance.
			sb.WriteString("\n")
		}
		prevY = s.Y

		sb.WriteString(pdfMapStyle(s, bodySize).Render(s.S))
	}
	return sb.String()
}

// pdfBodySize returns the most-frequent font size across all runs, used as the
// baseline for heading-level heuristics.
func pdfBodySize(sentences []pdf.Text) float64 {
	freq := map[float64]int{}
	for _, s := range sentences {
		rounded := math.Round(s.FontSize*2) / 2 // bucket to nearest 0.5 pt
		freq[rounded]++
	}
	var bodySize float64
	maxCount := 0
	for size, count := range freq {
		if count > maxCount {
			maxCount = count
			bodySize = size
		}
	}
	if bodySize == 0 {
		return 12.0
	}
	return bodySize
}

// pdfMapStyle maps a single PDF text run to a lipgloss.Style by inspecting its
// size ratio relative to the body size and bold/italic keywords in the font name.
func pdfMapStyle(t pdf.Text, bodySize float64) lipgloss.Style {
	ratio := t.FontSize / bodySize
	font := strings.ToLower(t.Font)

	isBold := strings.Contains(font, "bold") ||
		strings.Contains(font, "heavy") ||
		strings.Contains(font, "black") ||
		strings.Contains(font, "extrabold") ||
		strings.Contains(font, "semibold")
	isItalic := strings.Contains(font, "italic") || strings.Contains(font, "oblique")

	switch {
	case ratio >= 2.0:
		return styleH1
	case ratio >= 1.5:
		return styleH2
	case ratio >= 1.2:
		return styleH3
	case ratio < 0.85:
		return styleFaint
	default:
		st := lipgloss.NewStyle()
		if isBold {
			st = st.Bold(true)
		}
		if isItalic {
			st = st.Italic(true)
		}
		return st
	}
}
