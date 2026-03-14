// Package preview fetches a remote pCloud file and renders it for terminal
// display inside a bubbletea viewport.
//
// Format detection is centralised in GetPreviewType (type.go). Renderers:
//
//   - PreviewMarkdown → goldmark + chroma (render_markdown.go)
//   - PreviewCode     → chroma syntax highlight (render_code.go)
//   - PreviewText     → raw string (text.go)
//   - PreviewPDF      → ledongthuc/pdf text extraction (pdf.go)
//   - PreviewImage    → image2ascii colored ASCII art (image.go)
//   - PreviewCSV      → aligned text table (csv.go)
package preview

import (
	"fmt"
	"io"
	"net/http"

	"charm.land/lipgloss/v2"
)

// RenderFromURL downloads the file at downloadURL and renders its content to
// a string suitable for display inside a bubbletea viewport.
// name is the original filename (used for format detection).
// width and height define the target terminal dimensions.
func RenderFromURL(downloadURL, name string, width, height int) (string, error) {
	data, err := fetchBytes(downloadURL)
	if err != nil {
		return "", fmt.Errorf("download: %w", err)
	}

	return Render(data, name, width, height)
}

// Render renders raw file bytes to a string for terminal display.
// name is used purely for format detection via GetPreviewType.
// width and height are the available viewport dimensions; a 2-column border
// offset is subtracted from width here so all renderers stay within bounds.
func Render(data []byte, name string, width, height int) (string, error) {
	if width > 2 {
		width -= 2
	}

	switch GetPreviewType(name) {
	case PreviewMarkdown:
		return renderMarkdown(data)
	case PreviewPDF:
		return renderPDF(data)
	case PreviewImage:
		return renderImage(data, name, width, height)
	case PreviewCSV:
		return renderCSV(data, width)
	case PreviewText:
		return renderText(data)
	case PreviewCode:
		return renderCode(data, name)
	default: // PreviewUnsupported
		return RenderError("No preview available for this file type."), nil
	}
}

// fetchBytes downloads the URL and returns the raw body bytes (capped at 4 MB).
func fetchBytes(u string) ([]byte, error) {
	resp, err := http.Get(u) //nolint:noctx,gosec // preview download; URL sourced from pCloud API
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(io.LimitReader(resp.Body, 4*1024*1024))
}

var previewErrorStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("240")).
	Italic(true).
	Padding(0, 1)

// RenderError returns a styled terminal string for situations where a preview
// cannot be shown. The returned string is intended to be set as viewport
// content — callers should return it alongside a nil error.
func RenderError(format string, args ...any) string {
	msg := format
	if len(args) > 0 {
		msg = fmt.Sprintf(format, args...)
	}
	return previewErrorStyle.Render(msg)
}
