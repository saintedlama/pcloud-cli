package preview

import (
	"fmt"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

// renderMarkdown parses the Markdown AST with goldmark and walks it directly,
// applying terminal styles from text_styles.go to each semantic node kind.
// This replaces the old goldmark→HTML→stripHTML pipeline.
func renderMarkdown(data []byte) (string, error) {
	md := goldmark.New(
		goldmark.WithExtensions(extension.GFM),
		goldmark.WithParserOptions(parser.WithAutoHeadingID()),
	)
	reader := text.NewReader(data)
	root := md.Parser().Parse(reader)

	var sb strings.Builder
	for n := root.FirstChild(); n != nil; n = n.NextSibling() {
		renderMDBlock(n, data, &sb, 0)
	}
	return sb.String(), nil
}

// renderMDBlock renders a single block-level AST node into sb.
// indent is the current list nesting depth (in spaces).
func renderMDBlock(n ast.Node, src []byte, sb *strings.Builder, indent int) {
	switch n.Kind() {
	case ast.KindHeading:
		h := n.(*ast.Heading)
		sb.WriteString(headingStyle(h.Level).Render(collectMDInline(n, src)))
		sb.WriteString("\n\n")

	case ast.KindParagraph, ast.KindTextBlock:
		sb.WriteString(collectMDInline(n, src))
		sb.WriteString("\n\n")

	case ast.KindCodeBlock, ast.KindFencedCodeBlock:
		sb.WriteString(styleCodeBlock.Render(collectMDLines(n, src)))
		sb.WriteString("\n")

	case ast.KindBlockquote:
		var inner strings.Builder
		for c := n.FirstChild(); c != nil; c = c.NextSibling() {
			renderMDBlock(c, src, &inner, indent)
		}
		for _, line := range strings.Split(strings.TrimRight(inner.String(), "\n"), "\n") {
			sb.WriteString(styleBlockquote.Render("│ " + line))
			sb.WriteString("\n")
		}
		sb.WriteString("\n")

	case ast.KindList:
		list := n.(*ast.List)
		i := list.Start
		if i == 0 {
			i = 1
		}
		for c := n.FirstChild(); c != nil; c = c.NextSibling() {
			var itemText strings.Builder
			for ic := c.FirstChild(); ic != nil; ic = ic.NextSibling() {
				if ic.Kind() == ast.KindTextBlock || ic.Kind() == ast.KindParagraph {
					itemText.WriteString(collectMDInline(ic, src))
				} else {
					renderMDBlock(ic, src, &itemText, indent+2)
				}
			}
			var bullet string
			if list.IsOrdered() {
				bullet = fmt.Sprintf("%d. ", i)
				i++
			} else {
				bullet = "• "
			}
			sb.WriteString(strings.Repeat(" ", indent))
			sb.WriteString(styleListBullet.Render(bullet))
			sb.WriteString(" ")
			sb.WriteString(strings.TrimRight(itemText.String(), "\n"))
			sb.WriteString("\n")
		}
		sb.WriteString("\n")

	case ast.KindThematicBreak:
		sb.WriteString(styleFaint.Render(strings.Repeat("─", 40)))
		sb.WriteString("\n\n")

	case ast.KindHTMLBlock:
		sb.WriteString(stripHTML(collectMDLines(n, src)))

	default:
		for c := n.FirstChild(); c != nil; c = c.NextSibling() {
			renderMDBlock(c, src, sb, indent)
		}
	}
}

// collectMDInline collects the styled inline content of a node's children.
func collectMDInline(n ast.Node, src []byte) string {
	var sb strings.Builder
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		collectMDInlineNode(c, src, &sb)
	}
	return sb.String()
}

func collectMDInlineNode(n ast.Node, src []byte, sb *strings.Builder) {
	switch n.Kind() {
	case ast.KindText:
		t := n.(*ast.Text)
		seg := t.Segment
		sb.Write(seg.Value(src))
		if t.HardLineBreak() {
			sb.WriteString("\n")
		} else if t.SoftLineBreak() {
			sb.WriteString(" ")
		}
	case ast.KindString:
		sb.Write(n.(*ast.String).Value)
	case ast.KindEmphasis:
		inner := collectMDInline(n, src)
		if n.(*ast.Emphasis).Level >= 2 {
			sb.WriteString(styleBold.Render(inner))
		} else {
			sb.WriteString(styleItalic.Render(inner))
		}
	case ast.KindCodeSpan:
		sb.WriteString(styleCodeSpan.Render(collectMDCodeSpan(n, src)))
	case ast.KindLink:
		inner := collectMDInline(n, src)
		dest := string(n.(*ast.Link).Destination)
		sb.WriteString(inner)
		if dest != "" && dest != inner {
			sb.WriteString(styleFaint.Render(" (" + dest + ")"))
		}
	case ast.KindImage:
		sb.WriteString(styleFaint.Render("[image: " + collectMDInline(n, src) + "]"))
	default:
		if n.HasChildren() {
			for c := n.FirstChild(); c != nil; c = c.NextSibling() {
				collectMDInlineNode(c, src, sb)
			}
		} else if l := n.Lines(); l != nil {
			for i := 0; i < l.Len(); i++ {
				seg := l.At(i)
				sb.Write(seg.Value(src))
			}
		}
	}
}

// collectMDLines returns the raw source lines of a code block node.
func collectMDLines(n ast.Node, src []byte) string {
	var sb strings.Builder
	if l := n.Lines(); l != nil {
		for i := 0; i < l.Len(); i++ {
			seg := l.At(i)
			sb.Write(seg.Value(src))
		}
	}
	return sb.String()
}

// collectMDCodeSpan extracts the raw text of an inline code span.
func collectMDCodeSpan(n ast.Node, src []byte) string {
	var sb strings.Builder
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		if s, ok := c.(*ast.String); ok {
			sb.Write(s.Value)
		} else if t, ok := c.(*ast.Text); ok {
			seg := t.Segment
			sb.Write(seg.Value(src))
		}
	}
	return sb.String()
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
	out := sb.String()
	out = strings.ReplaceAll(out, "&amp;", "&")
	out = strings.ReplaceAll(out, "&lt;", "<")
	out = strings.ReplaceAll(out, "&gt;", ">")
	out = strings.ReplaceAll(out, "&quot;", `"`)
	out = strings.ReplaceAll(out, "&#39;", "'")
	out = strings.ReplaceAll(out, "&nbsp;", " ")
	return out
}
