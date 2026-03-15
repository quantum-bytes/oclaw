package chat

import (
	"encoding/json"
	"strings"
)

// ExtractText extracts the text content from a message content field.
// Content can be either a plain string or an array of content blocks.
func ExtractText(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}

	// Try as plain string first
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}

	// Try as array of content blocks
	var blocks []ContentBlock
	if err := json.Unmarshal(raw, &blocks); err == nil {
		var parts []string
		for _, b := range blocks {
			switch b.Type {
			case "text":
				parts = append(parts, b.Text)
			case "tool_use":
				parts = append(parts, "[tool: "+b.Name+"]")
			case "tool_result":
				if b.Text != "" {
					parts = append(parts, b.Text)
				}
			}
		}
		return strings.Join(parts, "\n")
	}

	return string(raw)
}

// ContentBlock represents a typed content block in a message.
type ContentBlock struct {
	Type     string          `json:"type"`
	Text     string          `json:"text,omitempty"`
	Thinking string          `json:"thinking,omitempty"` // for type "thinking"
	Name     string          `json:"name,omitempty"`
	Input    json.RawMessage `json:"input,omitempty"`
}

// ExtractThinking extracts thinking text from content blocks.
func ExtractThinking(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}

	var blocks []ContentBlock
	if err := json.Unmarshal(raw, &blocks); err != nil {
		return ""
	}

	var parts []string
	for _, b := range blocks {
		if b.Type == "thinking" && b.Thinking != "" {
			parts = append(parts, b.Thinking)
		}
	}
	return strings.Join(parts, "\n")
}

// ExtractToolCalls extracts tool call names from content blocks.
func ExtractToolCalls(raw json.RawMessage) []string {
	if len(raw) == 0 {
		return nil
	}

	var blocks []ContentBlock
	if err := json.Unmarshal(raw, &blocks); err != nil {
		return nil
	}

	var tools []string
	for _, b := range blocks {
		if b.Type == "tool_use" {
			tools = append(tools, b.Name)
		}
	}
	return tools
}
