package preview

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetPreviewType(t *testing.T) {
	tests := []struct {
		name string
		want PreviewType
	}{
		{"README.md", PreviewMarkdown},
		{"report.pdf", PreviewPDF},
		{"photo.jpg", PreviewImage},
		{"data.csv", PreviewCSV},
		{"notes.txt", PreviewText},
		{"main.go", PreviewCode},
		{"unknown.exe", PreviewUnsupported},
		// no extension → plain text
		{"Makefile", PreviewText},
		// case-insensitive
		{"IMAGE.PNG", PreviewImage},
		{"SCRIPT.PY", PreviewCode},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, GetPreviewType(tt.name))
		})
	}
}
