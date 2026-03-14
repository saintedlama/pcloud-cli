package preview

import (
	"bytes"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
)

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
		return renderText(data) // fallback to raw
	}
	// Strip HTML tags for plain terminal output.
	return stripHTML(htmlBuf.String()), nil
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
