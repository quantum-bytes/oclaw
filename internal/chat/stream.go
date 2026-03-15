package chat

import (
	"encoding/json"
	"strings"
	"sync"
)

// StreamState represents the current state of a response stream.
type StreamState int

const (
	StreamIdle     StreamState = iota
	StreamActive               // Receiving deltas
	StreamFinal                // Completed
	StreamAborted              // Cancelled
	StreamError                // Error occurred
)

// StreamAssembler accumulates streaming chat deltas into a complete response.
type StreamAssembler struct {
	mu sync.Mutex

	runID        string
	state        StreamState
	thinkingText strings.Builder
	contentText  strings.Builder
	toolCalls    []string
	stopReason   string
	errorMsg     string
}

// NewStreamAssembler creates a new stream assembler.
func NewStreamAssembler() *StreamAssembler {
	return &StreamAssembler{
		state: StreamIdle,
	}
}

// StreamDelta represents an incremental update from the assembler.
type StreamDelta struct {
	RunID    string
	State    StreamState
	Text     string // Current accumulated text
	Thinking string // Current accumulated thinking
	Tools    []string
	Error    string
}

// Reset clears the assembler state for a new response.
func (s *StreamAssembler) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.runID = ""
	s.state = StreamIdle
	s.thinkingText.Reset()
	s.contentText.Reset()
	s.toolCalls = nil
	s.stopReason = ""
	s.errorMsg = ""
}

// HandleEvent processes a chat event and returns the current state.
func (s *StreamAssembler) HandleEvent(event ChatEventPayload) StreamDelta {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.runID = event.RunID

	switch event.State {
	case "delta":
		s.state = StreamActive
		s.processDelta(event)

	case "final":
		s.state = StreamFinal
		s.processDelta(event)
		s.stopReason = event.StopReason

	case "aborted":
		s.state = StreamAborted

	case "error":
		s.state = StreamError
		s.errorMsg = event.ErrorMessage
	}

	return StreamDelta{
		RunID:    s.runID,
		State:    s.state,
		Text:     s.contentText.String(),
		Thinking: s.thinkingText.String(),
		Tools:    s.toolCalls,
		Error:    s.errorMsg,
	}
}

func (s *StreamAssembler) processDelta(event ChatEventPayload) {
	if event.Message == nil {
		return
	}

	var msg struct {
		Content  json.RawMessage `json:"content"`
		Thinking string          `json:"thinking"`
	}
	if err := json.Unmarshal(event.Message, &msg); err != nil {
		return
	}

	// Accumulate thinking text
	if msg.Thinking != "" {
		// For delta events, the thinking field contains the full accumulated text
		s.thinkingText.Reset()
		s.thinkingText.WriteString(msg.Thinking)
	}

	// Extract content text
	text := ExtractText(msg.Content)
	if text != "" {
		s.contentText.Reset()
		s.contentText.WriteString(text)
	}

	// Extract tool calls
	tools := ExtractToolCalls(msg.Content)
	if len(tools) > 0 {
		s.toolCalls = tools
	}
}

// State returns the current stream state.
func (s *StreamAssembler) State() StreamState {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state
}

// ChatEventPayload mirrors the chat event structure for the assembler.
type ChatEventPayload struct {
	SessionKey   string          `json:"sessionKey"`
	RunID        string          `json:"runId"`
	State        string          `json:"state"`
	Message      json.RawMessage `json:"message,omitempty"`
	ErrorMessage string          `json:"errorMessage,omitempty"`
	StopReason   string          `json:"stopReason,omitempty"`
}
