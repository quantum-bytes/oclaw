package ui

import (
	"strings"

	"github.com/charmbracelet/glamour"
	"github.com/muesli/reflow/wordwrap"
)

var (
	mdRenderer *glamour.TermRenderer
	mdWidth    int
)

func init() {
	// Initialize with a reasonable default
	SetMarkdownWidth(100)
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
	var err error
	mdRenderer, err = glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(width-2),
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
	return wordwrap.String(text, w-2)
}
