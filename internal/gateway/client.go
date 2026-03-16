package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const (
	defaultTickInterval = 30 * time.Second
	maxReconnectBackoff = 30 * time.Second
	eventBufferSize     = 256
	Version             = "0.1.0"
)

// Client manages a WebSocket connection to the OpenClaw gateway.
type Client struct {
	url   string
	token string

	conn     *websocket.Conn
	connMu   sync.Mutex
	writeMu  sync.Mutex // serializes all writes to the websocket
	pending  map[string]chan *ResponseFrame
	pendMu   sync.Mutex
	eventCh  chan *EventFrame
	closeCh  chan struct{}
	closed   bool
	closeMu  sync.Mutex

	tickInterval time.Duration
	lastTick     time.Time
	tickMu       sync.Mutex

	reconnectBackoff time.Duration
	connected        bool
	connectedMu      sync.RWMutex

	onConnect    func()
	onDisconnect func()
}

// NewClient creates a new gateway client.
func NewClient(url, token string) *Client {
	return &Client{
		url:              url,
		token:            token,
		pending:          make(map[string]chan *ResponseFrame),
		eventCh:          make(chan *EventFrame, eventBufferSize),
		closeCh:          make(chan struct{}),
		tickInterval:     defaultTickInterval,
		reconnectBackoff: time.Second,
	}
}

// OnConnect sets a callback for when the gateway connection is established.
func (c *Client) OnConnect(fn func()) { c.onConnect = fn }

// OnDisconnect sets a callback for when the gateway connection is lost.
func (c *Client) OnDisconnect(fn func()) { c.onDisconnect = fn }

// Events returns the channel for receiving gateway events.
func (c *Client) Events() <-chan *EventFrame { return c.eventCh }

// Connected returns whether the client is currently connected.
func (c *Client) Connected() bool {
	c.connectedMu.RLock()
	defer c.connectedMu.RUnlock()
	return c.connected
}

func (c *Client) setConnected(v bool) {
	c.connectedMu.Lock()
	prev := c.connected
	c.connected = v
	c.connectedMu.Unlock()

	if v && !prev && c.onConnect != nil {
		c.onConnect()
	} else if !v && prev && c.onDisconnect != nil {
		c.onDisconnect()
	}
}

// Connect establishes the WebSocket connection and runs the read loop.
// It blocks until the connection is closed or the context is cancelled.
func (c *Client) Connect(ctx context.Context) error {
	for {
		err := c.connectOnce(ctx)
		if err != nil {
			c.setConnected(false)
		}

		// Check if we're shut down
		c.closeMu.Lock()
		if c.closed {
			c.closeMu.Unlock()
			return nil
		}
		c.closeMu.Unlock()

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-c.closeCh:
			return nil
		case <-time.After(c.reconnectBackoff):
			// Exponential backoff
			c.reconnectBackoff = c.reconnectBackoff * 2
			if c.reconnectBackoff > maxReconnectBackoff {
				c.reconnectBackoff = maxReconnectBackoff
			}
		}
	}
}

func (c *Client) connectOnce(ctx context.Context) error {
	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}

	header := http.Header{}
	conn, _, err := dialer.DialContext(ctx, c.url, header)
	if err != nil {
		return fmt.Errorf("dial gateway: %w", err)
	}

	c.connMu.Lock()
	c.conn = conn
	c.connMu.Unlock()

	// Wait for challenge event
	nonce, err := c.waitForChallenge(ctx)
	if err != nil {
		conn.Close()
		return fmt.Errorf("challenge: %w", err)
	}

	// Send connect request
	hello, err := c.sendConnect(nonce)
	if err != nil {
		conn.Close()
		return fmt.Errorf("connect: %w", err)
	}

	if hello.Policy.TickIntervalMs > 0 {
		c.tickInterval = time.Duration(hello.Policy.TickIntervalMs) * time.Millisecond
	}

	c.tickMu.Lock()
	c.lastTick = time.Now()
	c.tickMu.Unlock()

	c.reconnectBackoff = time.Second // Reset backoff on success
	c.setConnected(true)

	// Run read loop and tick watchdog
	errCh := make(chan error, 1)
	go func() {
		errCh <- c.readLoop(ctx)
	}()

	go c.tickWatchdog(ctx)

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		conn.Close()
		return ctx.Err()
	case <-c.closeCh:
		conn.Close()
		return nil
	}
}

func (c *Client) waitForChallenge(ctx context.Context) (string, error) {
	c.connMu.Lock()
	conn := c.conn
	c.connMu.Unlock()

	conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	defer conn.SetReadDeadline(time.Time{})

	_, data, err := conn.ReadMessage()
	if err != nil {
		return "", fmt.Errorf("read challenge: %w", err)
	}

	var raw RawFrame
	if err := json.Unmarshal(data, &raw); err != nil {
		return "", fmt.Errorf("parse challenge frame: %w", err)
	}

	if raw.Type != "event" || raw.Event != "connect.challenge" {
		return "", fmt.Errorf("expected connect.challenge, got %s/%s", raw.Type, raw.Event)
	}

	var evt EventFrame
	if err := json.Unmarshal(data, &evt); err != nil {
		return "", fmt.Errorf("parse challenge event: %w", err)
	}

	var challenge ChallengePayload
	if err := json.Unmarshal(evt.Payload, &challenge); err != nil {
		return "", fmt.Errorf("parse challenge payload: %w", err)
	}

	return challenge.Nonce, nil
}

func (c *Client) sendConnect(nonce string) (*HelloPayload, error) {
	params := ConnectParams{
		MinProtocol: 3,
		MaxProtocol: 3,
		Client: ClientInfo{
			ID:         "cli",
			Version:    Version,
			Platform:   runtime.GOOS,
			Mode:       "cli",
			InstanceID: uuid.New().String(),
		},
		Caps: []string{},
		Auth: map[string]interface{}{
			"token": c.token,
		},
		Role:   "operator",
		Scopes: []string{"operator.admin"},
	}

	// Send connect request directly (readLoop not running yet)
	id := uuid.New().String()
	frame := RequestFrame{
		Type:   "req",
		ID:     id,
		Method: "connect",
		Params: params,
	}
	data, err := json.Marshal(frame)
	if err != nil {
		return nil, fmt.Errorf("marshal connect: %w", err)
	}

	c.connMu.Lock()
	conn := c.conn
	c.connMu.Unlock()

	c.writeMu.Lock()
	writeErr := conn.WriteMessage(websocket.TextMessage, data)
	c.writeMu.Unlock()
	if writeErr != nil {
		return nil, fmt.Errorf("write connect: %w", writeErr)
	}

	// Read response directly
	conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	defer conn.SetReadDeadline(time.Time{})

	_, respData, err := conn.ReadMessage()
	if err != nil {
		return nil, fmt.Errorf("read connect response: %w", err)
	}

	var resp ResponseFrame
	if err := json.Unmarshal(respData, &resp); err != nil {
		return nil, fmt.Errorf("parse connect response: %w", err)
	}

	if !resp.OK {
		if resp.Error != nil {
			return nil, resp.Error
		}
		return nil, fmt.Errorf("connect rejected")
	}

	var hello HelloPayload
	if resp.Payload != nil {
		if err := json.Unmarshal(resp.Payload, &hello); err != nil {
			return nil, fmt.Errorf("parse hello: %w", err)
		}
	}

	return &hello, nil
}

func (c *Client) readLoop(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-c.closeCh:
			return nil
		default:
		}

		c.connMu.Lock()
		conn := c.conn
		c.connMu.Unlock()

		if conn == nil {
			return fmt.Errorf("connection closed")
		}

		_, data, err := conn.ReadMessage()
		if err != nil {
			return fmt.Errorf("read: %w", err)
		}

		var raw RawFrame
		if err := json.Unmarshal(data, &raw); err != nil {
			continue // Skip malformed frames
		}

		switch raw.Type {
		case "res":
			var resp ResponseFrame
			if err := json.Unmarshal(data, &resp); err != nil {
				continue
			}
			c.pendMu.Lock()
			if ch, ok := c.pending[resp.ID]; ok {
				ch <- &resp
				delete(c.pending, resp.ID)
			}
			c.pendMu.Unlock()

		case "event":
			var evt EventFrame
			if err := json.Unmarshal(data, &evt); err != nil {
				continue
			}

			if evt.Event == "tick" {
				c.tickMu.Lock()
				c.lastTick = time.Now()
				c.tickMu.Unlock()
				continue
			}

			// Non-blocking send to event channel
			select {
			case c.eventCh <- &evt:
			default:
				// Drop if buffer full
			}
		}
	}
}

func (c *Client) tickWatchdog(ctx context.Context) {
	ticker := time.NewTicker(c.tickInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-c.closeCh:
			return
		case <-ticker.C:
			c.tickMu.Lock()
			elapsed := time.Since(c.lastTick)
			c.tickMu.Unlock()

			if elapsed > 2*c.tickInterval {
				// Tick timeout — close connection to trigger reconnect
				c.connMu.Lock()
				if c.conn != nil {
					c.conn.Close()
				}
				c.connMu.Unlock()
				return
			}
		}
	}
}

// Request sends an RPC request and waits for the response.
func (c *Client) Request(method string, params interface{}) (*ResponseFrame, error) {
	id := uuid.New().String()
	frame := RequestFrame{
		Type:   "req",
		ID:     id,
		Method: method,
		Params: params,
	}

	data, err := json.Marshal(frame)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	ch := make(chan *ResponseFrame, 1)
	c.pendMu.Lock()
	c.pending[id] = ch
	c.pendMu.Unlock()

	c.connMu.Lock()
	conn := c.conn
	c.connMu.Unlock()

	if conn == nil {
		c.pendMu.Lock()
		delete(c.pending, id)
		c.pendMu.Unlock()
		return nil, fmt.Errorf("not connected")
	}

	c.writeMu.Lock()
	writeErr := conn.WriteMessage(websocket.TextMessage, data)
	c.writeMu.Unlock()
	if writeErr != nil {
		c.pendMu.Lock()
		delete(c.pending, id)
		c.pendMu.Unlock()
		return nil, fmt.Errorf("write: %w", writeErr)
	}

	select {
	case resp := <-ch:
		return resp, nil
	case <-time.After(120 * time.Second):
		c.pendMu.Lock()
		delete(c.pending, id)
		c.pendMu.Unlock()
		return nil, fmt.Errorf("request timeout: %s", method)
	}
}

// SendChat sends a message to an agent and returns the request ID.
func (c *Client) SendChat(sessionKey, message, thinking string, timeoutMs int) (string, error) {
	idempotencyKey := uuid.New().String()
	params := ChatSendParams{
		SessionKey:     sessionKey,
		Message:        message,
		Thinking:       thinking,
		TimeoutMs:      timeoutMs,
		IdempotencyKey: idempotencyKey,
	}

	resp, err := c.Request("chat.send", params)
	if err != nil {
		return "", err
	}

	if !resp.OK {
		if resp.Error != nil {
			return "", resp.Error
		}
		return "", fmt.Errorf("chat.send failed")
	}

	return idempotencyKey, nil
}

// AbortChat cancels a running chat.
func (c *Client) AbortChat(sessionKey, runID string) error {
	resp, err := c.Request("chat.abort", ChatAbortParams{
		SessionKey: sessionKey,
		RunID:      runID,
	})
	if err != nil {
		return err
	}
	if !resp.OK && resp.Error != nil {
		return resp.Error
	}
	return nil
}

// LoadHistory fetches chat history for a session.
func (c *Client) LoadHistory(sessionKey string, limit int) ([]HistoryMessage, error) {
	resp, err := c.Request("chat.history", ChatHistoryParams{
		SessionKey: sessionKey,
		Limit:      limit,
	})
	if err != nil {
		return nil, err
	}
	if !resp.OK {
		if resp.Error != nil {
			return nil, resp.Error
		}
		return nil, fmt.Errorf("chat.history failed")
	}

	if resp.Payload == nil {
		return nil, nil
	}
	// The gateway returns {sessionKey, sessionId, messages: [...]}
	var result ChatHistoryResponse
	if err := json.Unmarshal(resp.Payload, &result); err != nil {
		return nil, fmt.Errorf("parse history: %w", err)
	}
	return result.Messages, nil
}

// ListSessions lists sessions from the gateway.
func (c *Client) ListSessions(params SessionsListParams) ([]SessionInfo, error) {
	resp, err := c.Request("sessions.list", params)
	if err != nil {
		return nil, err
	}
	if !resp.OK {
		if resp.Error != nil {
			return nil, resp.Error
		}
		return nil, fmt.Errorf("sessions.list failed")
	}

	var result SessionsListResponse
	if resp.Payload != nil {
		if err := json.Unmarshal(resp.Payload, &result); err != nil {
			return nil, fmt.Errorf("parse sessions: %w", err)
		}
	}
	return result.Sessions, nil
}

// ListAgents lists configured agents.
func (c *Client) ListAgents() ([]AgentInfo, error) {
	resp, err := c.Request("agents.list", nil)
	if err != nil {
		return nil, err
	}
	if !resp.OK {
		if resp.Error != nil {
			return nil, resp.Error
		}
		return nil, fmt.Errorf("agents.list failed")
	}

	var result AgentsListResponse
	if resp.Payload != nil {
		if err := json.Unmarshal(resp.Payload, &result); err != nil {
			return nil, fmt.Errorf("parse agents: %w", err)
		}
	}
	return result.Agents, nil
}

// ResetSession resets a session.
func (c *Client) ResetSession(key, reason string) error {
	resp, err := c.Request("sessions.reset", SessionsResetParams{
		Key:    key,
		Reason: reason,
	})
	if err != nil {
		return err
	}
	if !resp.OK && resp.Error != nil {
		return resp.Error
	}
	return nil
}

// Close shuts down the client.
func (c *Client) Close() {
	c.closeMu.Lock()
	if c.closed {
		c.closeMu.Unlock()
		return
	}
	c.closed = true
	close(c.closeCh)
	c.closeMu.Unlock()

	c.connMu.Lock()
	if c.conn != nil {
		c.conn.Close()
	}
	c.connMu.Unlock()
}
