package ui

import (
	"regexp"
	"strings"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/glamour/ansi"
	"github.com/charmbracelet/glamour/styles"
	"github.com/muesli/reflow/wordwrap"
)

var (
	mdRenderer *glamour.TermRenderer
	mdWidth    int

	// Matches numbered list items like "  1. " or "  10. " (with ANSI codes around the number)
	numberedListRe = regexp.MustCompile(`^(\s*(?:\x1b\[[0-9;]*m)*)(\d+\.\s)`)
	// Matches bullet list items like "  • " or "  - "
	bulletListRe = regexp.MustCompile(`^(\s*(?:\x1b\[[0-9;]*m)*)[•\-\*]\s`)
)

func init() {
	SetMarkdownWidth(100)
}

// customStyle returns a compact Glamour style.
func customStyle() ansi.StyleConfig {
	s := styles.DarkStyleConfig

	textColor := "#9CA3AF"
	zero := uint(0)

	// Tight document margins
	s.Document.Color = &textColor
	s.Document.Margin = &zero

	// Compact paragraphs
	s.Paragraph.Color = &textColor
	s.Paragraph.Margin = &zero

	// Lists
	s.List.LevelIndent = 2
	s.List.Color = &textColor
	s.List.Margin = &zero

	s.Item.BlockPrefix = ""
	s.Item.Color = &textColor

	// Headings — compact
	h3Color := "#06B6D4"
	s.H3.Color = &h3Color
	s.H3.Margin = &zero

	h2Color := "#22D3EE"
	s.H2.Color = &h2Color
	s.H2.Margin = &zero

	h1Color := "#67E8F9"
	s.H1.Color = &h1Color
	s.H1.Margin = &zero

	// Emphasis
	boldColor := "#E5E7EB"
	s.Emph.Color = &textColor
	s.Strong.Color = &boldColor

	// Code
	codeColor := "#A78BFA"
	s.Code.Color = &codeColor

	return s
}

// SetMarkdownWidth updates the renderer to wrap at the given width.
func SetMarkdownWidth(width int) {
	if width < 20 {
		width = 80
	}
	if width == mdWidth && mdRenderer != nil {
		return
	}
	mdWidth = width

	style := customStyle()
	var err error
	mdRenderer, err = glamour.NewTermRenderer(
		glamour.WithStyles(style),
		glamour.WithWordWrap(width-6),
	)
	if err != nil {
		mdRenderer = nil
	}
}

// RenderMarkdown renders markdown text for terminal display.
func RenderMarkdown(text string) string {
	if mdRenderer == nil || text == "" {
		return WrapText(text)
	}

	rendered, err := mdRenderer.Render(text)
	if err != nil {
		return WrapText(text)
	}

	rendered = strings.TrimRight(rendered, "\n")
	rendered = fixListIndentation(rendered)
	rendered = compactBlankLines(rendered)
	return rendered
}

// fixListIndentation re-wraps list items so continuation lines
// indent under the text, not back at column 0.
func fixListIndentation(text string) string {
	lines := strings.Split(text, "\n")
	var result []string
	indent := ""
	inListItem := false

	for _, line := range lines {
		stripped := stripAnsi(line)

		// Check if this is a new numbered list item
		if m := numberedListRe.FindStringIndex(line); m != nil {
			inListItem = true
			// Calculate indent: width of prefix + number + ". "
			prefixLen := visibleLen(stripped[:findContentStart(stripped)])
			indent = strings.Repeat(" ", prefixLen)
			result = append(result, line)
			continue
		}

		// Check if this is a new bullet list item
		if m := bulletListRe.FindStringIndex(line); m != nil {
			inListItem = true
			prefixLen := visibleLen(stripped[:findContentStart(stripped)])
			indent = strings.Repeat(" ", prefixLen)
			result = append(result, line)
			continue
		}

		// If we're in a list item and this is a continuation line (non-empty, no indent)
		if inListItem && strings.TrimSpace(stripped) != "" && !strings.HasPrefix(stripped, " ") {
			result = append(result, indent+line)
			continue
		}

		// Empty line ends list context
		if strings.TrimSpace(stripped) == "" {
			inListItem = false
			indent = ""
		}

		result = append(result, line)
	}

	return strings.Join(result, "\n")
}

// findContentStart finds where the actual text content starts in a list item.
// e.g., "  1. Text" -> returns index after "1. "
func findContentStart(s string) int {
	// Find the pattern: optional spaces, digit(s), dot, space
	re := regexp.MustCompile(`^(\s*\d+\.\s)`)
	if m := re.FindStringIndex(s); m != nil {
		return m[1]
	}
	// Bullet: optional spaces, bullet char, space
	re2 := regexp.MustCompile(`^(\s*[•\-\*]\s)`)
	if m := re2.FindStringIndex(s); m != nil {
		return m[1]
	}
	return 0
}

// stripAnsi removes ANSI escape codes from a string.
func stripAnsi(s string) string {
	re := regexp.MustCompile(`\x1b\[[0-9;]*m`)
	return re.ReplaceAllString(s, "")
}

// visibleLen returns the visible length of a string (ignoring ANSI codes).
func visibleLen(s string) int {
	return len(stripAnsi(s))
}

// compactBlankLines reduces multiple consecutive blank lines to one.
func compactBlankLines(text string) string {
	re := regexp.MustCompile(`\n{3,}`)
	return re.ReplaceAllString(text, "\n\n")
}

// WrapText wraps plain text to the current width.
func WrapText(text string) string {
	w := mdWidth
	if w <= 0 {
		w = 80
	}
	return wordwrap.String(text, w-6)
}
