package preview

import (
	"fmt"
	"path/filepath"
	"strings"

	"charm.land/lipgloss/v2"
)

// PreviewType identifies which renderer handles a given file.
type PreviewType string

const (
	PreviewText        PreviewType = "text"
	PreviewMarkdown    PreviewType = "markdown"
	PreviewCode        PreviewType = "code"
	PreviewImage       PreviewType = "image"
	PreviewPDF         PreviewType = "pdf"
	PreviewCSV         PreviewType = "csv"
	PreviewUnsupported PreviewType = "unsupported"
)

// GetPreviewType returns the PreviewType for the given filename based on its
// extension. It is the single authoritative source for extension→renderer
// mapping, replacing both the old CanPreview function and the ext switch in
// Render.
func GetPreviewType(name string) PreviewType {
	ext := strings.ToLower(filepath.Ext(name))
	switch ext {
	// Documents
	case ".md", ".markdown", ".mdx", ".rst", ".adoc", ".asciidoc", ".org", ".tex", ".bib":
		return PreviewMarkdown
	// Binary formats with dedicated renderers
	case ".pdf":
		return PreviewPDF
	case ".jpg", ".jpeg", ".png", ".gif", ".bmp", ".webp":
		return PreviewImage
	// Tabular data
	case ".csv", ".tsv":
		return PreviewCSV
	// Plain text and logs
	case ".txt", ".log", ".diff", ".patch", "":
		return PreviewText
	// Data / config formats — rendered via chroma
	case ".json", ".yaml", ".yml", ".toml", ".ini", ".xml",
		".env", ".envrc", ".conf", ".cfg", ".config", ".properties",
		".tf", ".tfvars", ".nix", ".lock":
		return PreviewCode
	// Web and templating
	case ".html", ".htm", ".css", ".scss", ".sass", ".less",
		".vue", ".svelte",
		".pug", ".jade", ".hbs", ".mustache":
		return PreviewCode
	// General scripting and compiled languages
	case ".go", ".py", ".rb", ".php", ".lua", ".pl", ".pm", ".awk",
		".js", ".ts", ".jsx", ".tsx", ".coffee",
		".cs", ".vb", ".fs", ".fsx",
		".java", ".kt", ".scala", ".groovy", ".gradle",
		".swift", ".dart",
		".rs", ".c", ".cpp", ".h", ".hpp", ".zig", ".nim", ".d",
		".hs", ".clj", ".cljs", ".erl", ".hrl", ".ex", ".exs",
		".r", ".jl",
		".ml", ".mli",
		".sh", ".bash", ".zsh", ".ps1", ".bat", ".cmd",
		".sql",
		".proto", ".graphql", ".gql",
		".dockerfile", ".makefile", ".gitignore":
		return PreviewCode
	}
	return PreviewUnsupported
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
