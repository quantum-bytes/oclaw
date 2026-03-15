package ui

import (
	"github.com/charmbracelet/glamour"
)

var mdRenderer *glamour.TermRenderer

func init() {
	var err error
	mdRenderer, err = glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(0), // We handle wrapping ourselves
	)
	if err != nil {
		// Fallback to no rendering
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
