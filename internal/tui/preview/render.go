// Package preview fetches a remote pCloud file and renders it for terminal
// display inside a bubbletea viewport.
//
// Format detection is centralised in GetPreviewType (type.go). Renderers:
//
//   - PreviewMarkdown → goldmark + chroma
//   - PreviewCode     → chroma syntax highlight
//   - PreviewText     → raw string
//   - PreviewPDF      → ledongthuc/pdf text extraction
//   - PreviewImage    → image2ascii colored ASCII art
//   - PreviewCSV      → aligned text table
package preview

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/ledongthuc/pdf"
	"github.com/qeesung/image2ascii/convert"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
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
		return string(data), nil
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

// renderMarkdown converts markdown to a simple plain-text representation
// suitable for terminal display. It uses goldmark to parse the document and
// produces an annotated text output with ANSI bold/underline headings.
func renderMarkdown(data []byte) (string, error) {
	// Render to HTML as an intermediate step, then strip tags for plain text.
	// We produce lightweight ANSI output instead of full HTML.
	md := goldmark.New(
		goldmark.WithExtensions(extension.GFM),
		goldmark.WithParserOptions(parser.WithAutoHeadingID()),
		goldmark.WithRendererOptions(html.WithUnsafe()),
	)
	var htmlBuf bytes.Buffer
	if err := md.Convert(data, &htmlBuf); err != nil {
		return string(data), nil // fallback to raw
	}
	// Strip HTML tags for plain terminal output.
	return stripHTML(htmlBuf.String()), nil
}

// renderCode uses chroma to syntax-highlight source code and structured text.
func renderCode(data []byte, name string) (string, error) {
	src := string(data)

	// Try filename first, then content analysis, then fall back.
	l := lexers.Match(name)
	if l == nil {
		l = lexers.Analyse(src)
	}
	if l == nil {
		l = lexers.Fallback
	}

	style := styles.Get("monokai")
	if style == nil {
		style = styles.Fallback
	}

	formatter := formatters.Get("terminal256")
	if formatter == nil {
		formatter = formatters.Fallback
	}

	iter, err := l.Tokenise(nil, src)
	if err != nil {
		return src, nil // fallback to plain text
	}

	var buf bytes.Buffer
	if err := formatter.Format(&buf, style, iter); err != nil {
		return src, nil
	}
	return buf.String(), nil
}

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

// renderImage converts a raster image to colored ASCII art via image2ascii.
func renderImage(data []byte, name string, width, height int) (result string, err error) {
	// Recover from any panics in the resize/conversion step.
	defer func() {
		if r := recover(); r != nil {
			result = ""
			err = fmt.Errorf("image decode failed: %v", r)
		}
	}()

	// Decode the image from raw bytes using the standard library.
	// This returns a proper error instead of calling log.Fatal like
	// image2ascii's ImageFile2ASCIIString does.
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("image decode: %w", err)
	}

	opts := convert.DefaultOptions
	opts.Colored = true
	opts.FitScreen = false
	if width > 0 {
		opts.FixedWidth = width
	}

	if height > 0 {
		opts.FixedHeight = height
	}

	c := convert.NewImageConverter()
	return c.Image2ASCIIString(img, &opts), nil
}

// renderCSV formats a CSV file as an aligned text table.
func renderCSV(data []byte, width int) (string, error) {
	r := csv.NewReader(bytes.NewReader(data))
	records, err := r.ReadAll()
	if err != nil {
		return string(data), nil // fallback to plain text
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

// stripHTML removes HTML tags and decodes common entities for plain-text display.
func stripHTML(s string) string {
	var sb strings.Builder
	inTag := false
	for _, r := range s {
		switch {
		case r == '<':
			inTag = true
		case r == '>':
			inTag = false
		case !inTag:
			sb.WriteRune(r)
		}
	}
	// Decode common HTML entities.
	out := sb.String()
	out = strings.ReplaceAll(out, "&amp;", "&")
	out = strings.ReplaceAll(out, "&lt;", "<")
	out = strings.ReplaceAll(out, "&gt;", ">")
	out = strings.ReplaceAll(out, "&quot;", `"`)
	out = strings.ReplaceAll(out, "&#39;", "'")
	out = strings.ReplaceAll(out, "&nbsp;", " ")
	return out
}
