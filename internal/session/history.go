package session

import (
	"github.com/quantum-bytes/oclaw/internal/chat"
	"github.com/quantum-bytes/oclaw/internal/gateway"
)

// Message is a simplified message for display.
type Message struct {
	Role     string
	Text     string
	Thinking string
	Tools    []string
}

// ConvertHistory converts gateway history messages to display messages.
func ConvertHistory(history []gateway.HistoryMessage) []Message {
	var messages []Message
	for _, h := range history {
		messages = append(messages, Message{
			Role:     h.Role,
			Text:     chat.ExtractText(h.Content),
			Thinking: chat.ExtractThinking(h.Content),
			Tools:    chat.ExtractToolCalls(h.Content),
		})
	}
	return messages
}
