package preview

import (
	"bytes"

	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
)

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
		return renderText(data) // fallback to plain text
	}

	var buf bytes.Buffer
	if err := formatter.Format(&buf, style, iter); err != nil {
		return renderText(data)
	}
	return buf.String(), nil
}
