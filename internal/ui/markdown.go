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

	ansiRe     = regexp.MustCompile(`\x1b\[[0-9;]*m`)
	listItemRe = regexp.MustCompile(`^\d+\.\s`)
	bulletRe   = regexp.MustCompile(`^[•\-\*]\s`)
)

func init() {
	SetMarkdownWidth(100)
}

func customStyle() ansi.StyleConfig {
	s := styles.DarkStyleConfig

	textColor := "#9CA3AF"
	zero := uint(0)

	s.Document.Color = &textColor
	s.Document.Margin = &zero
	s.Paragraph.Color = &textColor
	s.Paragraph.Margin = &zero

	s.List.LevelIndent = 2
	s.List.Color = &textColor
	s.List.Margin = &zero
	s.Item.BlockPrefix = ""
	s.Item.Color = &textColor

	h3Color := "#06B6D4"
	s.H3.Color = &h3Color
	s.H3.Margin = &zero
	h2Color := "#22D3EE"
	s.H2.Color = &h2Color
	s.H2.Margin = &zero
	h1Color := "#67E8F9"
	s.H1.Color = &h1Color
	s.H1.Margin = &zero

	boldColor := "#E5E7EB"
	s.Emph.Color = &textColor
	s.Strong.Color = &boldColor

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
		glamour.WithWordWrap(width-10), // narrower to leave room for indent
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

// fixListIndentation adds indentation to continuation lines of list items.
func fixListIndentation(text string) string {
	lines := strings.Split(text, "\n")
	var result []string
	inListItem := false
	indentStr := "   " // 3 spaces to align under text after "1. "

	for _, line := range lines {
		visible := stripAnsi(line)
		trimmed := strings.TrimSpace(visible)

		// Empty line — end list context
		if trimmed == "" {
			inListItem = false
			result = append(result, line)
			continue
		}

		// New numbered list item (e.g., "1. Cancer:")
		if listItemRe.MatchString(trimmed) {
			inListItem = true
			result = append(result, line)
			continue
		}

		// New bullet list item
		if bulletRe.MatchString(trimmed) {
			inListItem = true
			result = append(result, line)
			continue
		}

		// Continuation of a list item — indent it
		if inListItem {
			result = append(result, indentStr+line)
			continue
		}

		result = append(result, line)
	}

	return strings.Join(result, "\n")
}

func stripAnsi(s string) string {
	return ansiRe.ReplaceAllString(s, "")
}

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
