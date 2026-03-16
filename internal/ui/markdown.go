package ui

import (
	"strings"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/glamour/ansi"
	"github.com/charmbracelet/glamour/styles"
	"github.com/muesli/reflow/wordwrap"
)

var (
	mdRenderer *glamour.TermRenderer
	mdWidth    int
)

func init() {
	SetMarkdownWidth(100)
}

// customStyle returns a Glamour style with proper list indentation and softer text.
func customStyle() ansi.StyleConfig {
	s := styles.DarkStyleConfig

	// Softer text color for body
	textColor := "#9CA3AF"
	s.Document.Color = &textColor
	s.Paragraph.Color = &textColor

	// List items: proper indent so wrapped text aligns under content, not bullet
	indent := uint(4)
	s.List.LevelIndent = indent
	s.List.Color = &textColor

	s.Item.BlockPrefix = ""
	s.Item.Color = &textColor

	// Headings
	h3Color := "#06B6D4" // Cyan for h3 (matching screenshot)
	s.H3.Color = &h3Color

	// Bold
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
		glamour.WithWordWrap(width-4),
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

	return strings.TrimRight(rendered, "\n")
}

// WrapText wraps plain text to the current width.
func WrapText(text string) string {
	w := mdWidth
	if w <= 0 {
		w = 80
	}
	return wordwrap.String(text, w-4)
}
