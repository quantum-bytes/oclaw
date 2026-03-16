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
	// Match URLs in plain text (not already inside an OSC 8 sequence)
	urlRe = regexp.MustCompile(`(https?://[^\s\)>\]]+|file://[^\s\)>\]]+)`)
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

// sanitizeInput strips raw terminal escape sequences from untrusted input text
// before it reaches the Glamour renderer. This prevents model-injected ANSI/OSC
// sequences from being passed through to the terminal.
func sanitizeInput(text string) string {
	// Strip OSC sequences (e.g., \x1b]8;; ... \x07)
	oscRe := regexp.MustCompile(`\x1b\][^\x07]*\x07`)
	text = oscRe.ReplaceAllString(text, "")
	// Strip CSI sequences (e.g., \x1b[31m)
	text = ansiRe.ReplaceAllString(text, "")
	// Strip bare ESC, BEL, and other control chars except newline/tab
	var sb strings.Builder
	sb.Grow(len(text))
	for _, r := range text {
		if r == '\n' || r == '\t' || r == '\r' || r >= 0x20 {
			sb.WriteRune(r)
		}
	}
	return sb.String()
}

// RenderMarkdown renders markdown text for terminal display.
func RenderMarkdown(text string) string {
	if mdRenderer == nil || text == "" {
		return WrapText(text)
	}

	// Sanitize untrusted input before rendering
	text = sanitizeInput(text)

	rendered, err := mdRenderer.Render(text)
	if err != nil {
		return WrapText(text)
	}

	rendered = strings.TrimRight(rendered, "\n")
	rendered = fixListIndentation(rendered)
	rendered = compactBlankLines(rendered)
	rendered = makeHyperlinks(rendered)
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

// sanitizeURL strips control characters (bytes < 0x20, 0x7F, and ESC) from a URL
// to prevent injection into OSC 8 escape sequences.
func sanitizeURL(u string) string {
	var sb strings.Builder
	sb.Grow(len(u))
	for _, r := range u {
		if r < 0x20 || r == 0x7F || r == 0x1B {
			continue
		}
		sb.WriteRune(r)
	}
	return sb.String()
}

// makeHyperlinks converts URLs into clickable OSC 8 terminal hyperlinks.
// Format: \x1b]8;;URL\x07DISPLAY_TEXT\x1b]8;;\x07
func makeHyperlinks(text string) string {
	return urlRe.ReplaceAllStringFunc(text, func(url string) string {
		// Strip any trailing ANSI codes that got captured
		cleanURL := ansiRe.ReplaceAllString(url, "")
		// Sanitize control characters to prevent OSC 8 injection
		cleanURL = sanitizeURL(cleanURL)
		// OSC 8 hyperlink: clickable in iTerm2, Kitty, WezTerm, GNOME Terminal, etc.
		return "\x1b]8;;" + cleanURL + "\x07" + url + "\x1b]8;;\x07"
	})
}

// WrapText wraps plain text to the current width.
func WrapText(text string) string {
	w := mdWidth
	if w <= 0 {
		w = 80
	}
	text = sanitizeInput(text)
	wrapped := wordwrap.String(text, w-6)
	return makeHyperlinks(wrapped)
}
