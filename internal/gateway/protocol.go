package gateway

import (
	"encoding/json"
	"fmt"
)

// Frame types for the OpenClaw gateway WebSocket protocol.

// RequestFrame is sent by the client to invoke an RPC method.
type RequestFrame struct {
	Type   string      `json:"type"`   // always "req"
	ID     string      `json:"id"`     // unique request ID (UUID)
	Method string      `json:"method"` // RPC method name
	Params interface{} `json:"params,omitempty"`
}

// ResponseFrame is returned by the gateway for a request.
type ResponseFrame struct {
	Type    string          `json:"type"`              // always "res"
	ID      string          `json:"id"`                // matches request ID
	OK      bool            `json:"ok"`                // success flag
	Payload json.RawMessage `json:"payload,omitempty"` // response data
	Error   *RPCError       `json:"error,omitempty"`
}

// EventFrame is pushed by the gateway for async events.
type EventFrame struct {
	Type    string          `json:"type"`              // always "event"
	Event   string          `json:"event"`             // event name
	Payload json.RawMessage `json:"payload,omitempty"` // event data
	Seq     int             `json:"seq,omitempty"`
}

// RPCError represents an error from the gateway.
type RPCError struct {
	Message string `json:"message"`
	Code    string `json:"code,omitempty"`
}

func (e *RPCError) Error() string {
	if e.Code != "" {
		return e.Code + ": " + e.Message
	}
	return e.Message
}

// RawFrame is used for initial JSON parsing to determine frame type.
type RawFrame struct {
	Type  string `json:"type"`
	Event string `json:"event,omitempty"`
}

// ConnectParams are sent in the "connect" request after receiving the challenge.
type ConnectParams struct {
	MinProtocol int                    `json:"minProtocol"`
	MaxProtocol int                    `json:"maxProtocol"`
	Client      ClientInfo             `json:"client"`
	Caps        []string               `json:"caps"`
	Auth        map[string]interface{} `json:"auth"`
	Role        string                 `json:"role"`
	Scopes      []string               `json:"scopes"`
}

// ClientInfo identifies this TUI client to the gateway.
type ClientInfo struct {
	ID         string `json:"id"`
	Version    string `json:"version"`
	Platform   string `json:"platform"`
	Mode       string `json:"mode"`
	InstanceID string `json:"instanceId"`
}

// ChallengePayload is the payload of a connect.challenge event.
type ChallengePayload struct {
	Nonce string `json:"nonce"`
}

// HelloPayload is the response payload from a successful connect.
type HelloPayload struct {
	Type     string       `json:"type"` // "hello-ok"
	Protocol int          `json:"protocol"`
	Policy   HelloPolicy  `json:"policy,omitempty"`
	Features HelloFeatures `json:"features,omitempty"`
}

// HelloPolicy contains connection policy from the gateway.
type HelloPolicy struct {
	TickIntervalMs int `json:"tickIntervalMs"`
	MaxPayload     int `json:"maxPayload"`
}

// HelloFeatures contains the gateway's supported methods and events.
type HelloFeatures struct {
	Methods []string `json:"methods,omitempty"`
	Events  []string `json:"events,omitempty"`
}

// ChatSendParams are the parameters for chat.send RPC.
type ChatSendParams struct {
	SessionKey     string `json:"sessionKey"`
	Message        string `json:"message"`
	Thinking       string `json:"thinking,omitempty"`
	TimeoutMs      int    `json:"timeoutMs,omitempty"`
	IdempotencyKey string `json:"idempotencyKey"`
}

// ChatAbortParams are the parameters for chat.abort RPC.
type ChatAbortParams struct {
	SessionKey string `json:"sessionKey"`
	RunID      string `json:"runId"`
}

// ChatHistoryParams are the parameters for chat.history RPC.
type ChatHistoryParams struct {
	SessionKey string `json:"sessionKey"`
	Limit      int    `json:"limit,omitempty"`
}

// SessionsListParams are the parameters for sessions.list RPC.
type SessionsListParams struct {
	Limit               int    `json:"limit,omitempty"`
	ActiveMinutes       int    `json:"activeMinutes,omitempty"`
	AgentID             string `json:"agentId,omitempty"`
	IncludeDerivedTitle bool   `json:"includeDerivedTitles,omitempty"`
	IncludeLastMessage  bool   `json:"includeLastMessage,omitempty"`
}

// SessionsResetParams are the parameters for sessions.reset RPC.
type SessionsResetParams struct {
	Key    string `json:"key"`
	Reason string `json:"reason,omitempty"`
}

// ChatEvent represents a streaming chat event payload.
type ChatEvent struct {
	SessionKey   string          `json:"sessionKey"`
	RunID        string          `json:"runId"`
	State        string          `json:"state"` // "delta", "final", "aborted", "error"
	Message      json.RawMessage `json:"message,omitempty"`
	ErrorMessage string          `json:"errorMessage,omitempty"`
	Usage        json.RawMessage `json:"usage,omitempty"`
	StopReason   string          `json:"stopReason,omitempty"`
}

// ChatMessage represents a message in chat events.
type ChatMessage struct {
	Role     string          `json:"role"`
	Content  json.RawMessage `json:"content"` // string or []ContentBlock
	Thinking string          `json:"thinking,omitempty"`
}

// ContentBlock is a typed content block in a message.
type ContentBlock struct {
	Type  string          `json:"type"`            // "text", "tool_use", "tool_result"
	Text  string          `json:"text,omitempty"`  // for type "text"
	Name  string          `json:"name,omitempty"`  // for type "tool_use"
	Input json.RawMessage `json:"input,omitempty"` // for type "tool_use"
}

// AgentsListResponse wraps the agents.list response.
type AgentsListResponse struct {
	DefaultID string      `json:"defaultId"`
	Agents    []AgentInfo `json:"agents"`
}

// AgentInfo represents an agent from agents.list.
type AgentInfo struct {
	ID       string        `json:"id"`
	Name     string        `json:"name"`
	Identity AgentIdentity `json:"identity,omitempty"`
	Model    string        `json:"-"` // populated from config, not from gateway
}

// AgentIdentity contains display info for an agent.
type AgentIdentity struct {
	Name  string `json:"name"`
	Emoji string `json:"emoji"`
}

// SessionsListResponse wraps the sessions.list response.
type SessionsListResponse struct {
	Sessions []SessionInfo `json:"sessions"`
}

// SessionInfo represents a session from sessions.list.
type SessionInfo struct {
	Key          string      `json:"key"`
	AgentID      string      `json:"agentId,omitempty"`
	Title        string      `json:"title,omitempty"`
	LastMessage  string      `json:"lastMessage,omitempty"`
	UpdatedAt    interface{} `json:"updatedAt,omitempty"` // can be string or number
	Active       bool        `json:"active,omitempty"`
	MessageCount int         `json:"messageCount,omitempty"`
}

// UpdatedAtString returns the updatedAt as a string.
func (s SessionInfo) UpdatedAtString() string {
	switch v := s.UpdatedAt.(type) {
	case string:
		return v
	case float64:
		return fmt.Sprintf("%.0f", v)
	default:
		return ""
	}
}

// ChatHistoryResponse wraps the chat.history response.
type ChatHistoryResponse struct {
	Messages []HistoryMessage `json:"messages"`
}

// HistoryMessage represents a message from chat.history.
type HistoryMessage struct {
	Role      string          `json:"role"`
	Content   json.RawMessage `json:"content"`
	Thinking  string          `json:"thinking,omitempty"`
	Timestamp json.Number     `json:"timestamp,omitempty"`
	Model     string          `json:"model,omitempty"`
	Provider  string          `json:"provider,omitempty"`
}
