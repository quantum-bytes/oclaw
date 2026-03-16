package ui

import (
	"github.com/charmbracelet/glamour"
)

var (
	mdRenderer *glamour.TermRenderer
	mdWidth    int
)

// SetMarkdownWidth updates the renderer to wrap at the given width.
func SetMarkdownWidth(width int) {
	if width == mdWidth && mdRenderer != nil {
		return
	}
	if width < 20 {
		width = 80
	}
	mdWidth = width
	var err error
	mdRenderer, err = glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(width-4), // leave margin for padding
	)
	if err != nil {
		mdRenderer = nil
	}
}

// RenderMarkdown renders markdown text for terminal display.
func RenderMarkdown(text string) string {
	if mdRenderer == nil || text == "" {
		return text
	}

	rendered, err := mdRenderer.Render(text)
	if err != nil {
		return text
	}

	return rendered
}
