package preview

import (
	"bytes"
	"fmt"
	"os"

	"github.com/ledongthuc/pdf"
)

// renderPDF extracts plain text from a PDF file using ledongthuc/pdf.
func renderPDF(data []byte) (result string, err error) {
	defer func() {
		if r := recover(); r != nil {
			result = ""
			err = fmt.Errorf("pdf decode failed: %v", r)
		}
	}()
	// ledongthuc/pdf requires a ReadSeeker, so write to a temp file.
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

	textReader, err := reader.GetPlainText()
	if err != nil {
		return "", fmt.Errorf("pdf text extraction: %w", err)
	}

	var buf bytes.Buffer
	if _, err := buf.ReadFrom(textReader); err != nil {
		return "", fmt.Errorf("pdf read: %w", err)
	}
	return buf.String(), nil
}
